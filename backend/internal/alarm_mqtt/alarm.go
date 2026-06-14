package alarm_mqtt

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
	"ancient-bridge-system/internal/messaging"
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

type AlarmMQTTService struct {
	bus          *messaging.MessageBus
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

func NewAlarmMQTTService(bus *messaging.MessageBus) *AlarmMQTTService {
	service := &AlarmMQTTService{
		bus:          bus,
		offlineQueue: list.New(),
		stopRetry:    make(chan struct{}),
	}

	service.loadPersistedQueue()
	service.initMQTTClient()

	go service.watchAlerts()

	service.retryTicker = time.NewTicker(5 * time.Second)
	go service.retryLoop()

	return service
}

func (as *AlarmMQTTService) initMQTTClient() {
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
		log.Printf("AlarmMQTT: Connection lost: %v, buffering alerts", err)
		as.setConnected(false)
	})

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("AlarmMQTT: Reconnected, flushing offline queue")
		as.setConnected(true)
		go as.flushOfflineQueue()
	})

	opts.SetReconnectingHandler(func(c mqtt.Client, o *mqtt.ClientOptions) {
		log.Printf("AlarmMQTT: Attempting reconnect to %v", o.Servers)
	})

	client := mqtt.NewClient(opts)
	as.client = client

	go func() {
		token := client.Connect()
		if token.WaitTimeout(10*time.Second) && token.Error() != nil {
			log.Printf("AlarmMQTT: Failed to connect to broker: %v, buffering locally", token.Error())
			as.setConnected(false)
		} else {
			log.Println("AlarmMQTT: Alert service initialized")
			as.setConnected(true)
			go as.flushOfflineQueue()
		}
	}()
}

func (as *AlarmMQTTService) setConnected(v bool) {
	as.connectedMu.Lock()
	defer as.connectedMu.Unlock()
	as.connected = v
}

func (as *AlarmMQTTService) isConnected() bool {
	as.connectedMu.RLock()
	defer as.connectedMu.RUnlock()
	return as.connected && as.client != nil && as.client.IsConnected()
}

func (as *AlarmMQTTService) watchAlerts() {
	for {
		select {
		case msg := <-as.bus.AlertChan:
			alert, ok := msg.Payload.(*messaging.AlertPayload)
			if !ok {
				log.Println("AlarmMQTT: Invalid alert payload")
				continue
			}

			alertID := alert.AlertID
			if alertID == 0 {
				alertID = as.saveAlertToDB(alert)
				alert.AlertID = alertID
			}

			as.publishAlert(alert)
		}
	}
}

func (as *AlarmMQTTService) saveAlertToDB(alert *messaging.AlertPayload) int {
	query := `
		INSERT INTO alerts (bridge_id, member_id, alert_type, alert_level, alert_message,
			measured_value, threshold_value, timestamp, mqtt_topic)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING alert_id
	`

	var id int
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
		config.AppConfig.MQTTTopic,
	).Scan(&id)

	if err != nil {
		log.Printf("AlarmMQTT: Failed to save alert to DB: %v", err)
		return 0
	}
	return id
}

func (as *AlarmMQTTService) enqueueAlert(topic string, payload []byte, alertID int) {
	as.queueMutex.Lock()
	defer as.queueMutex.Unlock()

	if as.offlineQueue.Len() >= maxOfflineQueueSize {
		front := as.offlineQueue.Front()
		if front != nil {
			as.offlineQueue.Remove(front)
			log.Printf("AlarmMQTT: Offline queue full, dropped oldest alert")
		}
	}

	as.offlineQueue.PushBack(&QueuedAlert{
		Topic:     topic,
		Payload:   payload,
		AlertID:   alertID,
		Attempts:  0,
		NextRetry: time.Now(),
	})
	log.Printf("AlarmMQTT: Alert %d enqueued offline (queue size: %d)", alertID, as.offlineQueue.Len())
}

func (as *AlarmMQTTService) flushOfflineQueue() {
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

	log.Printf("AlarmMQTT: Flushing %d buffered alerts", len(batch))

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
			log.Printf("AlarmMQTT: Failed to flush alert %d (attempt %d): %v", qa.AlertID, qa.Attempts, token.Error())
			if qa.Attempts < maxRetryAttempts {
				delay := initialRetryDelay * time.Duration(1<<uint(qa.Attempts-1))
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
				qa.NextRetry = time.Now().Add(delay)
				as.enqueueAlertDirect(qa)
			} else {
				log.Printf("AlarmMQTT: Alert %d exceeded max retries, discarded", qa.AlertID)
			}
			failedCount++
		} else {
			sentCount++
		}
	}
	log.Printf("AlarmMQTT: Flush complete: sent=%d, remaining=%d", sentCount, failedCount)
}

func (as *AlarmMQTTService) enqueueAlertDirect(qa *QueuedAlert) {
	as.queueMutex.Lock()
	defer as.queueMutex.Unlock()
	if as.offlineQueue.Len() < maxOfflineQueueSize {
		as.offlineQueue.PushBack(qa)
	}
}

func (as *AlarmMQTTService) retryLoop() {
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

func (as *AlarmMQTTService) retryDueAlerts() {
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

	log.Printf("AlarmMQTT: Retrying %d due alerts", len(due))
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
				log.Printf("AlarmMQTT: Alert %d exceeded max retries, discarded", qa.AlertID)
			}
		}
	}
}

func (as *AlarmMQTTService) persistQueue() {
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
		log.Printf("AlarmMQTT: Failed to persist offline queue: %v", err)
		return
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	if err := enc.Encode(items); err != nil {
		log.Printf("AlarmMQTT: Failed to encode offline queue: %v", err)
	} else {
		log.Printf("AlarmMQTT: Persisted %d alerts to disk", len(items))
	}
}

func (as *AlarmMQTTService) loadPersistedQueue() {
	f, err := os.Open(persistenceFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("AlarmMQTT: Failed to open persisted queue: %v", err)
		}
		return
	}
	defer f.Close()

	var items []*QueuedAlert
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&items); err != nil {
		log.Printf("AlarmMQTT: Failed to decode persisted queue: %v", err)
		return
	}

	for _, qa := range items {
		as.offlineQueue.PushBack(qa)
	}
	log.Printf("AlarmMQTT: Loaded %d persisted alerts from disk", len(items))
	os.Remove(persistenceFile)
}

func (as *AlarmMQTTService) publishAlert(alert *messaging.AlertPayload) {
	alertMsg := AlertMessage{
		AlertID:        alert.AlertID,
		BridgeID:       alert.BridgeID,
		MemberID:       alert.MemberID,
		MemberCode:     alert.MemberCode,
		AlertType:      alert.AlertType,
		AlertLevel:     alert.AlertLevel,
		AlertMessage:   alert.AlertMessage,
		MeasuredValue:  alert.MeasuredValue,
		ThresholdValue: alert.ThresholdValue,
		Timestamp:      alert.Timestamp,
		Unit:           alert.Unit,
	}

	payload, err := json.Marshal(alertMsg)
	if err != nil {
		log.Printf("AlarmMQTT: Failed to marshal alert: %v", err)
		return
	}

	topic := fmt.Sprintf("%s/%d/%s", config.AppConfig.MQTTTopic, alert.BridgeID, alert.AlertLevel)

	if !as.isConnected() {
		log.Printf("AlarmMQTT: Not connected, buffering alert %d", alert.AlertID)
		as.enqueueAlert(topic, payload, alert.AlertID)
		return
	}

	token := as.client.Publish(topic, 1, false, payload)
	go func() {
		if token.WaitTimeout(5*time.Second) && token.Error() != nil {
			log.Printf("AlarmMQTT: Failed to publish alert %d: %v, buffering", alert.AlertID, token.Error())
			as.enqueueAlert(topic, payload, alert.AlertID)
		}
	}()
}

func (as *AlarmMQTTService) GetRecentAlerts(bridgeID int, limit int) ([]models.Alert, error) {
	var alerts []models.Alert
	query := `SELECT * FROM alerts WHERE bridge_id = $1 ORDER BY timestamp DESC LIMIT $2`
	err := database.DB.Select(&alerts, query, bridgeID, limit)
	return alerts, err
}

func (as *AlarmMQTTService) AcknowledgeAlert(alertID int, acknowledgedBy string) error {
	query := `
		UPDATE alerts
		SET is_acknowledged = true, acknowledged_at = CURRENT_TIMESTAMP, acknowledged_by = $1
		WHERE alert_id = $2
	`
	_, err := database.DB.Exec(query, acknowledgedBy, alertID)
	return err
}

func (as *AlarmMQTTService) Close() {
	close(as.stopRetry)
	if as.retryTicker != nil {
		as.retryTicker.Stop()
	}

	as.persistQueue()

	if as.client != nil && as.client.IsConnected() {
		as.client.Disconnect(500)
		log.Println("AlarmMQTT: Service disconnected")
	}
}
