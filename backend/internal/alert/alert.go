package alert

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ancient-bridge-system/internal/config"
	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type AlertService struct {
	client mqtt.Client
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
	service := &AlertService{}

	opts := mqtt.NewClientOptions()
	brokerURL := fmt.Sprintf("tcp://%s:%d", config.AppConfig.MQTTBroker, config.AppConfig.MQTTPort)
	opts.AddBroker(brokerURL)
	opts.SetClientID(config.AppConfig.MQTTClientID)

	if config.AppConfig.MQTTUsername != "" {
		opts.SetUsername(config.AppConfig.MQTTUsername)
	}
	if config.AppConfig.MQTTPassword != "" {
		opts.SetPassword(config.AppConfig.MQTTPassword)
	}

	opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("MQTT client connected successfully")
	})

	client := mqtt.NewClient(opts)
	service.client = client

	go func() {
		token := client.Connect()
		if token.Wait() && token.Error() != nil {
			log.Printf("Failed to connect to MQTT broker: %v, alert service will work in degraded mode", token.Error())
		} else {
			log.Println("MQTT alert service initialized")
		}
	}()

	return service
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
	if as.client == nil || !as.client.IsConnected() {
		log.Println("MQTT client not connected, skipping alert publish")
		return
	}

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

	token := as.client.Publish(topic, 1, false, payload)
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to publish alert: %v", token.Error())
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
	if as.client != nil && as.client.IsConnected() {
		as.client.Disconnect(250)
		log.Println("MQTT alert service disconnected")
	}
}
