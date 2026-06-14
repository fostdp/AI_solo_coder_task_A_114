package sensor

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ancient-bridge-system/internal/alert"
	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/models"
)

type SensorDataIngestor struct{}

func NewSensorDataIngestor() *SensorDataIngestor {
	return &SensorDataIngestor{}
}

type DTUPayload struct {
	DTUDeviceID string      `json:"dtu_device_id"`
	Timestamp   string      `json:"timestamp"`
	Sensors     []SensorReading `json:"sensors"`
	RawData     interface{} `json:"raw_data,omitempty"`
}

type SensorReading struct {
	SensorCode string  `json:"sensor_code"`
	Value      float64 `json:"value"`
	Quality    int     `json:"quality"`
	Unit       string  `json:"unit"`
}

func (ing *SensorDataIngestor) IngestDTUData(payload DTUPayload) error {
	timestamp, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	for _, reading := range payload.Sensors {
		sensor, err := getSensorByCode(reading.SensorCode)
		if err != nil {
			log.Printf("Sensor not found: %s, error: %v", reading.SensorCode, err)
			continue
		}

		sensorData := &models.SensorData{
			SensorID:    sensor.SensorID,
			Timestamp:   timestamp,
			Value:       reading.Value,
			QualityFlag: reading.Quality,
		}

		rawJSON, _ := json.Marshal(payload.RawData)
		rawStr := string(rawJSON)
		sensorData.RawData = &rawStr

		err = saveSensorData(sensorData)
		if err != nil {
			log.Printf("Failed to save sensor data: %v", err)
			continue
		}

		if alert.GlobalAlertService != nil {
			err = alert.GlobalAlertService.CheckAndAlertSensor(sensor, reading.Value)
			if err != nil {
				log.Printf("Failed to check sensor alert: %v", err)
			}
		}
	}

	return nil
}

func getSensorByCode(sensorCode string) (*models.Sensor, error) {
	var sensor models.Sensor

	query := `
		SELECT * FROM sensors
		WHERE sensor_code = $1 AND status = 'active'
		LIMIT 1
	`

	err := database.DB.Get(&sensor, query, sensorCode)
	if err != nil {
		return nil, fmt.Errorf("sensor not found: %s", sensorCode)
	}

	return &sensor, nil
}

func saveSensorData(data *models.SensorData) error {
	query := `
		INSERT INTO sensor_data (sensor_id, timestamp, value, quality_flag, raw_data)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := database.DB.Exec(
		query,
		data.SensorID,
		data.Timestamp,
		data.Value,
		data.QualityFlag,
		data.RawData,
	)

	return err
}

func GetSensorData(sensorID int, startTime, endTime time.Time, limit int) ([]models.SensorData, error) {
	var data []models.SensorData

	query := `
		SELECT * FROM sensor_data
		WHERE sensor_id = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC
		LIMIT $4
	`

	err := database.DB.Select(&data, query, sensorID, startTime, endTime, limit)
	return data, err
}

func GetLatestSensorData(sensorID int) (*models.SensorData, error) {
	var data models.SensorData

	query := `
		SELECT * FROM sensor_data
		WHERE sensor_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	err := database.DB.Get(&data, query, sensorID)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func GetSensorDataHourly(sensorID int, startTime, endTime time.Time) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	query := `
		SELECT 
			bucket as timestamp,
			avg_value,
			min_value,
			max_value,
			sample_count
		FROM sensor_data_hourly
		WHERE sensor_id = $1 AND bucket BETWEEN $2 AND $3
		ORDER BY bucket
	`

	rows, err := database.DB.Queryx(query, sensorID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		row := make(map[string]interface{})
		err = rows.MapScan(row)
		if err != nil {
			continue
		}
		results = append(results, row)
	}

	return results, nil
}

func SaveEnvironmentalData(data *models.EnvironmentalData) error {
	query := `
		INSERT INTO environmental_data 
		(bridge_id, timestamp, temperature, humidity, wind_speed, wind_direction, rainfall)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := database.DB.Exec(
		query,
		data.BridgeID,
		data.Timestamp,
		data.Temperature,
		data.Humidity,
		data.WindSpeed,
		data.WindDirection,
		data.Rainfall,
	)

	return err
}

func GetEnvironmentalData(bridgeID int, startTime, endTime time.Time) ([]models.EnvironmentalData, error) {
	var data []models.EnvironmentalData

	query := `
		SELECT * FROM environmental_data
		WHERE bridge_id = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp
	`

	err := database.DB.Select(&data, query, bridgeID, startTime, endTime)
	return data, err
}

func GetAllSensors(bridgeID int) ([]models.Sensor, error) {
	var sensors []models.Sensor

	query := `
		SELECT * FROM sensors
		WHERE bridge_id = $1 AND status = 'active'
		ORDER BY sensor_code
	`

	err := database.DB.Select(&sensors, query, bridgeID)
	return sensors, err
}
