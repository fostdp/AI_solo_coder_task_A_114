package dtu_receiver

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/messaging"
	"ancient-bridge-system/internal/models"
)

type DTUReceiver struct {
	bus       *messaging.MessageBus
	sensorMap map[string]*models.Sensor
	mapMu     sync.RWMutex
}

type SensorReading struct {
	SensorCode  string  `json:"sensor_code"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	QualityFlag int     `json:"quality_flag"`
}

type DTUPayload struct {
	DTUDeviceID string          `json:"dtu_device_id"`
	Timestamp   string          `json:"timestamp"`
	Readings    []SensorReading `json:"readings"`
	BridgeID    int             `json:"bridge_id"`
}

func NewDTUReceiver(bus *messaging.MessageBus) *DTUReceiver {
	recv := &DTUReceiver{
		bus:       bus,
		sensorMap: make(map[string]*models.Sensor),
	}
	go recv.watchSensorData()
	return recv
}

func (r *DTUReceiver) ProcessIngest(payload *DTUPayload) error {
	ts, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		ts = time.Now()
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, reading := range payload.Readings {
		wg.Add(1)
		go func(rd SensorReading) {
			defer wg.Done()

			sensor := r.getSensorByCode(rd.SensorCode, payload.BridgeID)
			if sensor == nil {
				mu.Lock()
				errs = append(errs, nil)
				mu.Unlock()
				return
			}

			if err := r.saveSensorData(sensor.SensorID, rd.Value, ts, rd.QualityFlag); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			msg := messaging.NewMessage(messaging.MsgTypeSensorData, &messaging.SensorDataPayload{
				BridgeID:    payload.BridgeID,
				SensorCode:  rd.SensorCode,
				Value:       rd.Value,
				Timestamp:   ts,
				QualityFlag: rd.QualityFlag,
				Unit:        rd.Unit,
			})

			select {
			case r.bus.SensorDataChan <- msg:
			default:
				log.Printf("DTUReceiver: SensorDataChan full, dropping sensor %s data", rd.SensorCode)
			}
		}(reading)
	}

	wg.Wait()

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (r *DTUReceiver) getSensorByCode(sensorCode string, bridgeID int) *models.Sensor {
	r.mapMu.RLock()
	s, ok := r.sensorMap[sensorCode]
	r.mapMu.RUnlock()
	if ok && s != nil {
		return s
	}

	var sensor models.Sensor
	query := `SELECT * FROM sensors WHERE sensor_code = $1 AND is_active = true AND bridge_id = $2 LIMIT 1`
	err := database.DB.Get(&sensor, query, sensorCode, bridgeID)
	if err != nil {
		log.Printf("DTUReceiver: Sensor not found: %s (bridge %d): %v", sensorCode, bridgeID, err)
		return nil
	}

	r.mapMu.Lock()
	r.sensorMap[sensorCode] = &sensor
	r.mapMu.Unlock()
	return &sensor
}

func (r *DTUReceiver) saveSensorData(sensorID int, value float64, ts time.Time, qualityFlag int) error {
	query := `
		INSERT INTO sensor_data (sensor_id, timestamp, value, quality_flag)
		VALUES ($1, $2, $3, $4)
	`
	_, err := database.DB.Exec(query, sensorID, ts, value, qualityFlag)
	return err
}

func (r *DTUReceiver) watchSensorData() {
	for {
		select {
		case msg := <-r.bus.SensorDataChan:
			_ = msg
		}
	}
}

func (r *DTUReceiver) ParsePayload(data []byte) (*DTUPayload, error) {
	var payload DTUPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return nil, err
	}
	return &payload, nil
}
