package alert

import (
	"container/list"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"ancient-bridge-system/internal/config"
	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	maxOfflineQueueSize = 10000
	maxRetryAttempts    = 5
	initialRetryDelay   = 1 * time.Second
	persistenceFile     = "mqtt_offline_alerts.gob"
)

type QueuedAlert struct {
	Topic     string
	Payload   []byte
	AlertID   int
	Attempts  int
	NextRetry time.Time
}

type AlertService struct {
	client       mqtt.Client
	offlineQueue *list.List
	queueMutex   sync.Mutex
	retryTicker  *time.Ticker
	stopRetry    chan struct{}
	connected    bool
	connectedMu  sync.RWMutex
}

type AlertMessage struct {
	AlertID        int       `json:"alert_id"`
	BridgeID       int       `json:"bridge_id"`
	BridgeName     string    `json:"bridge_name"`
	MemberID       *int      `json:"member_id,omitempty"`
	MemberCode     string    `json:"member_code,omitempty"`
	AlertType      string    `json:"alert_type"`
	AlertLevel     string    `json:"alert_level"`
	AlertMessage   string    `json:"alert_message"`
	MeasuredValue  float64   `json:"measured_value"`
	ThresholdValue float64   `json:"threshold_value"`
	Unit           string    `json:"unit"`
	Timestamp      time.Time `json:"timestamp"`
}

var GlobalAlertService *AlertService

func InitAlertService() {
	GlobalAlertService = NewAlertService()
}

func NewAlertService() *AlertService {
	service := &AlertService{
		offlineQueue: list.New(),
		stopRetry:    make(chan struct{}),
	}

	service.loadPersistedQueue()

	opts := mqtt.NewClientOptions()
	brokerURL := fmt.Sprintf("tcp://%s:%d", config.AppConfig.MQTTBroker, config.AppConfig.MQTTPort)
	opts.AddBroker(brokerURL)
	opts.SetClientID(config.AppConfig.MQTTClientID)
	opts.SetCleanSession(false)

	if config.AppConfig.MQTTUsername != "" {
		opts.SetUsername(config.AppConfig.MQTTUsername)
	}
	if config.AppConfig.MQTTPassword != "" {
		opts.SetPassword(config.AppConfig.MQTTPassword)
	}

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetWriteTimeout(10 * time.Second)
	opts.SetMessageChannelDepth(256)

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v, buffering alerts in offline queue", err)
		service.setConnected(false)
	})

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("MQTT client reconnected, flushing offline queue")
		service.setConnected(true)
		go service.flushOfflineQueue()
	})

	opts.SetReconnectingHandler(func(c mqtt.Client, o *mqtt.ClientOptions) {
		log.Printf("MQTT attempting reconnect to %v", o.Servers)
	})

	client := mqtt.NewClient(opts)
	service.client = client

	go func() {
		token := client.Connect()
		if token.WaitTimeout(10*time.Second) && token.Error() != nil {
			log.Printf("Failed to connect to MQTT broker: %v, alert service will buffer locally", token.Error())
			service.setConnected(false)
		} else {
			log.Println("MQTT alert service initialized")
			service.setConnected(true)
			go service.flushOfflineQueue()
		}
	}()

	service.retryTicker = time.NewTicker(5 * time.Second)
	go service.retryLoop()

	return service
}

func (as *AlertService) setConnected(v bool) {
	as.connectedMu.Lock()
	defer as.connectedMu.Unlock()
	as.connected = v
}

func (as *AlertService) isConnected() bool {
	as.connectedMu.RLock()
	defer as.connectedMu.RUnlock()
	return as.connected && as.client != nil && as.client.IsConnected()
}

func (as *AlertService) enqueueAlert(topic string, payload []byte, alertID int) {
	as.queueMutex.Lock()
	defer as.queueMutex.Unlock()

	if as.offlineQueue.Len() >= maxOfflineQueueSize {
		front := as.offlineQueue.Front()
		if front != nil {
			as.offlineQueue.Remove(front)
			log.Printf("Offline queue full, dropped oldest alert")
		}
	}

	as.offlineQueue.PushBack(&QueuedAlert{
		Topic:     topic,
		Payload:   payload,
		AlertID:   alertID,
		Attempts:  0,
		NextRetry: time.Now(),
	})
	log.Printf("Alert %d enqueued offline (queue size: %d)", alertID, as.offlineQueue.Len())
}

func (as *AlertService) flushOfflineQueue() {
	as.queueMutex.Lock()
	if as.offlineQueue.Len() == 0 {
		as.queueMutex.Unlock()
		return
	}

	batch := make([]*QueuedAlert, 0, as.offlineQueue.Len())
	for e := as.offlineQueue.Front(); e != nil; e = e.Next() {
		if qa, ok := e.Value.(*QueuedAlert); ok {
			batch = append(batch, qa)
		}
	}
	as.offlineQueue.Init()
	as.queueMutex.Unlock()

	log.Printf("Flushing %d buffered alerts to MQTT", len(batch))

	sentCount := 0
	failedCount := 0
	for _, qa := range batch {
		if !as.isConnected() {
			as.enqueueAlertDirect(qa)
			failedCount++
			continue
		}
		token := as.client.Publish(qa.Topic, 1, false, qa.Payload)
		if token.WaitTimeout(5*time.Second) && token.Error() != nil {
			qa.Attempts++
			log.Printf("Failed to flush alert %d (attempt %d): %v", qa.AlertID, qa.Attempts, token.Error())
			if qa.Attempts < maxRetryAttempts {
				delay := initialRetryDelay * time.Duration(1<<uint(qa.Attempts-1))
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
				qa.NextRetry = time.Now().Add(delay)
				as.enqueueAlertDirect(qa)
			} else {
				log.Printf("Alert %d exceeded max retries (%d), discarded", qa.AlertID, maxRetryAttempts)
			}
			failedCount++
		} else {
			sentCount++
		}
	}
	log.Printf("Flush complete: sent=%d, remaining=%d", sentCount, failedCount)
}

func (as *AlertService) enqueueAlertDirect(qa *QueuedAlert) {
	as.queueMutex.Lock()
	defer as.queueMutex.Unlock()
	if as.offlineQueue.Len() < maxOfflineQueueSize {
		as.offlineQueue.PushBack(qa)
	}
}

func (as *AlertService) retryLoop() {
	for {
		select {
		case <-as.retryTicker.C:
			if as.isConnected() {
				go as.retryDueAlerts()
			}
		case <-as.stopRetry:
			return
		}
	}
}

func (as *AlertService) retryDueAlerts() {
	as.queueMutex.Lock()
	now := time.Now()
	due := make([]*QueuedAlert, 0)
	remaining := list.New()
	for e := as.offlineQueue.Front(); e != nil; e = e.Next() {
		if qa, ok := e.Value.(*QueuedAlert); ok {
			if qa.NextRetry.Before(now) || qa.NextRetry.Equal(now) {
				due = append(due, qa)
			} else {
				remaining.PushBack(qa)
			}
		}
	}
	as.offlineQueue = remaining
	as.queueMutex.Unlock()

	if len(due) == 0 {
		return
	}

	log.Printf("Retrying %d due alerts", len(due))
	for _, qa := range due {
		if !as.isConnected() {
			as.enqueueAlertDirect(qa)
			continue
		}
		token := as.client.Publish(qa.Topic, 1, false, qa.Payload)
		if token.WaitTimeout(5*time.Second) && token.Error() != nil {
			qa.Attempts++
			if qa.Attempts < maxRetryAttempts {
				delay := initialRetryDelay * time.Duration(1<<uint(qa.Attempts-1))
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
				qa.NextRetry = time.Now().Add(delay)
				as.enqueueAlertDirect(qa)
			} else {
				log.Printf("Alert %d exceeded max retries, discarded", qa.AlertID)
			}
		}
	}
}

func (as *AlertService) persistQueue() {
	as.queueMutex.Lock()
	defer as.queueMutex.Unlock()

	if as.offlineQueue.Len() == 0 {
		os.Remove(persistenceFile)
		return
	}

	items := make([]*QueuedAlert, 0, as.offlineQueue.Len())
	for e := as.offlineQueue.Front(); e != nil; e = e.Next() {
		if qa, ok := e.Value.(*QueuedAlert); ok {
			items = append(items, qa)
		}
	}

	f, err := os.Create(persistenceFile)
	if err != nil {
		log.Printf("Failed to persist offline queue: %v", err)
		return
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	if err := enc.Encode(items); err != nil {
		log.Printf("Failed to encode offline queue: %v", err)
	} else {
		log.Printf("Persisted %d alerts to disk", len(items))
	}
}

func (as *AlertService) loadPersistedQueue() {
	f, err := os.Open(persistenceFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to open persisted queue: %v", err)
		}
		return
	}
	defer f.Close()

	var items []*QueuedAlert
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&items); err != nil {
		log.Printf("Failed to decode persisted queue: %v", err)
		return
	}

	for _, qa := range items {
		as.offlineQueue.PushBack(qa)
	}
	log.Printf("Loaded %d persisted alerts from disk", len(items))
	os.Remove(persistenceFile)
}

func (as *AlertService) CheckAndAlertMemberStress(bridgeID int, memberID int, stressRatio float64, measuredValue float64, thresholdValue float64, memberCode string) error {
	if stressRatio < config.AppConfig.StressWarningRatio {
		return nil
	}

	alertLevel := "warning"
	alertType := "stress_warning"
	alertMsg := fmt.Sprintf("构件 %s 应力比达到 %.2f，接近容许值", memberCode, stressRatio)

	if stressRatio >= config.AppConfig.StressDangerRatio {
		alertLevel = "danger"
		alertType = "stress_exceeded"
		alertMsg = fmt.Sprintf("构件 %s 应力超限！应力比 %.2f，超过容许值", memberCode, stressRatio)
	}

	memberIDPtr := &memberID
	alert := &models.Alert{
		BridgeID:       bridgeID,
		MemberID:       memberIDPtr,
		AlertType:      alertType,
		AlertLevel:     alertLevel,
		AlertMessage:   alertMsg,
		MeasuredValue:  measuredValue,
		ThresholdValue: thresholdValue,
		Timestamp:      time.Now(),
		MQTTTopic:      config.AppConfig.MQTTTopic,
	}

	query := `
		INSERT INTO alerts (bridge_id, member_id, alert_type, alert_level, alert_message,
			measured_value, threshold_value, timestamp, mqtt_topic)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING alert_id
	`

	err := database.DB.QueryRow(
		query,
		alert.BridgeID,
		alert.MemberID,
		alert.AlertType,
		alert.AlertLevel,
		alert.AlertMessage,
		alert.MeasuredValue,
		alert.ThresholdValue,
		alert.Timestamp,
		alert.MQTTTopic,
	).Scan(&alert.AlertID)

	if err != nil {
		return fmt.Errorf("failed to save alert: %v", err)
	}

	as.publishAlert(alert)

	log.Printf("Alert triggered: %s - %s (level: %s)", alertType, alertMsg, alertLevel)

	return nil
}

func (as *AlertService) CheckAndAlertSensor(sensor *models.Sensor, value float64) error {
	threshold := sensor.RangeMax

	if value <= sensor.RangeMax && value >= sensor.RangeMin {
		return nil
	}

	alertLevel := "warning"
	alertType := "sensor_out_of_range"
	alertMsg := fmt.Sprintf("传感器 %s 测量值 %.2f %s 超出量程", sensor.SensorCode, value, sensor.Unit)

	if value > sensor.RangeMax*1.2 || value < sensor.RangeMin*0.8 {
		alertLevel = "danger"
		alertType = "sensor_critical"
		alertMsg = fmt.Sprintf("传感器 %s 测量值严重超限 %.2f %s", sensor.SensorCode, value, sensor.Unit)
	}

	sensorIDPtr := &sensor.SensorID
	alert := &models.Alert{
		BridgeID:       sensor.BridgeID,
		SensorID:       sensorIDPtr,
		AlertType:      alertType,
		AlertLevel:     alertLevel,
		AlertMessage:   alertMsg,
		MeasuredValue:  value,
		ThresholdValue: sensor.RangeMax,
		Timestamp:      time.Now(),
		MQTTTopic:      config.AppConfig.MQTTTopic,
	}

	query := `
		INSERT INTO alerts (bridge_id, sensor_id, alert_type, alert_level, alert_message,
			measured_value, threshold_value, timestamp, mqtt_topic)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING alert_id
	`

	err := database.DB.QueryRow(
		query,
		alert.BridgeID,
		alert.SensorID,
		alert.AlertType,
		alert.AlertLevel,
		alert.AlertMessage,
		alert.MeasuredValue,
		alert.ThresholdValue,
		alert.Timestamp,
		alert.MQTTTopic,
	).Scan(&alert.AlertID)

	if err != nil {
		return fmt.Errorf("failed to save alert: %v", err)
	}

	as.publishAlert(alert)

	return nil
}

func (as *AlertService) publishAlert(alert *models.Alert) {
	alertMsg := AlertMessage{
		AlertID:        alert.AlertID,
		BridgeID:       alert.BridgeID,
		MemberID:       alert.MemberID,
		AlertType:      alert.AlertType,
		AlertLevel:     alert.AlertLevel,
		AlertMessage:   alert.AlertMessage,
		MeasuredValue:  alert.MeasuredValue,
		ThresholdValue: alert.ThresholdValue,
		Timestamp:      alert.Timestamp,
	}

	payload, err := json.Marshal(alertMsg)
	if err != nil {
		log.Printf("Failed to marshal alert message: %v", err)
		return
	}

	topic := fmt.Sprintf("%s/%d/%s", config.AppConfig.MQTTTopic, alert.BridgeID, alert.AlertLevel)

	if !as.isConnected() {
		log.Printf("MQTT not connected, buffering alert %d", alert.AlertID)
		as.enqueueAlert(topic, payload, alert.AlertID)
		return
	}

	token := as.client.Publish(topic, 1, false, payload)
	go func() {
		if token.WaitTimeout(5*time.Second) && token.Error() != nil {
			log.Printf("Failed to publish alert %d: %v, buffering", alert.AlertID, token.Error())
			as.enqueueAlert(topic, payload, alert.AlertID)
		}
	}()
}

func (as *AlertService) GetRecentAlerts(bridgeID int, limit int) ([]models.Alert, error) {
	var alerts []models.Alert

	query := `
		SELECT * FROM alerts
		WHERE bridge_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	err := database.DB.Select(&alerts, query, bridgeID, limit)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func (as *AlertService) AcknowledgeAlert(alertID int, acknowledgedBy string) error {
	query := `
		UPDATE alerts
		SET is_acknowledged = true,
			acknowledged_at = CURRENT_TIMESTAMP,
			acknowledged_by = $1
		WHERE alert_id = $2
	`

	_, err := database.DB.Exec(query, acknowledgedBy, alertID)
	return err
}

func (as *AlertService) GetActiveAlerts(bridgeID int) ([]models.Alert, error) {
	var alerts []models.Alert

	query := `
		SELECT * FROM alerts
		WHERE bridge_id = $1 AND is_acknowledged = false
		ORDER BY timestamp DESC
	`

	err := database.DB.Select(&alerts, query, bridgeID)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func (as *AlertService) Close() {
	close(as.stopRetry)
	if as.retryTicker != nil {
		as.retryTicker.Stop()
	}

	as.persistQueue()

	if as.client != nil && as.client.IsConnected() {
		as.client.Disconnect(500)
		log.Println("MQTT alert service disconnected")
	}
}
