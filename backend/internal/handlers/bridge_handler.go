package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/models"

	"github.com/gin-gonic/gin"
)

type BridgeHandler struct{}

func NewBridgeHandler() *BridgeHandler {
	return &BridgeHandler{}
}

func (h *BridgeHandler) GetAllBridges(c *gin.Context) {
	var bridges []models.Bridge

	query := `
		SELECT * FROM bridges
		WHERE status = 'active'
		ORDER BY bridge_id
	`

	err := database.DB.Select(&bridges, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(bridges),
		"data":  bridges,
	})
}

func (h *BridgeHandler) GetBridge(c *gin.Context) {
	id := c.Param("id")

	var bridge models.Bridge
	query := `SELECT * FROM bridges WHERE bridge_id = $1`

	err := database.DB.Get(&bridge, query, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bridge not found"})
		return
	}

	c.JSON(http.StatusOK, bridge)
}

func (h *BridgeHandler) GetBridgeMembers(c *gin.Context) {
	id := c.Param("id")

	var members []models.BridgeMember
	query := `
		SELECT * FROM bridge_members
		WHERE bridge_id = $1
		ORDER BY position_order, member_code
	`

	err := database.DB.Select(&members, query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(members),
		"data":  members,
	})
}

func (h *BridgeHandler) GetBridgeNodes(c *gin.Context) {
	id := c.Param("id")

	var nodes []models.BridgeNode
	query := `
		SELECT * FROM bridge_nodes
		WHERE bridge_id = $1
		ORDER BY node_code
	`

	err := database.DB.Select(&nodes, query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(nodes),
		"data":  nodes,
	})
}

func (h *BridgeHandler) GetBridgeSensors(c *gin.Context) {
	id := c.Param("id")

	var sensors []models.Sensor
	query := `
		SELECT * FROM sensors
		WHERE bridge_id = $1 AND status = 'active'
		ORDER BY sensor_type, sensor_code
	`

	err := database.DB.Select(&sensors, query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(sensors),
		"data":  sensors,
	})
}

func (h *BridgeHandler) CreateBridge(c *gin.Context) {
	var bridge models.Bridge
	if err := c.ShouldBindJSON(&bridge); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		INSERT INTO bridges (name, alias, dynasty, location, span_length, arch_rise,
			deck_width, total_length, material_type, construction_method,
			historical_record, documentation_source, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'active')
		RETURNING bridge_id, created_at, updated_at
	`

	err := database.DB.QueryRow(
		query,
		bridge.Name,
		bridge.Alias,
		bridge.Dynasty,
		bridge.Location,
		bridge.SpanLength,
		bridge.ArchRise,
		bridge.DeckWidth,
		bridge.TotalLength,
		bridge.MaterialType,
		bridge.ConstructionMethod,
		bridge.HistoricalRecord,
		bridge.DocumentationSource,
	).Scan(&bridge.BridgeID, &bridge.CreatedAt, &bridge.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bridge)
}

func (h *BridgeHandler) GetStressOverview(c *gin.Context) {
	id := c.Param("id")

	var result struct {
		BridgeID     int     `db:"bridge_id"`
		Name         string  `db:"name"`
		SpanLength   float64 `db:"span_length"`
		TotalMembers int     `db:"total_members"`
		TotalSensors int     `db:"total_sensors"`
		MaxStressRatio *float64 `db:"max_stress_ratio"`
		OverspeedCount int    `db:"overspeed_count"`
	}

	query := `
		SELECT 
			b.bridge_id,
			b.name,
			b.span_length,
			COUNT(DISTINCT m.member_id) AS total_members,
			COUNT(DISTINCT s.sensor_id) AS total_sensors,
			MAX(mf.stress_ratio) AS max_stress_ratio,
			SUM(CASE WHEN mf.is_overspeed THEN 1 ELSE 0 END) AS overspeed_count
		FROM bridges b
		LEFT JOIN bridge_members m ON b.bridge_id = m.bridge_id
		LEFT JOIN sensors s ON b.bridge_id = s.bridge_id
		LEFT JOIN (
			SELECT a.bridge_id, mf.*
			FROM member_forces mf
			JOIN analysis_results a ON mf.analysis_id = a.analysis_id
			WHERE a.analysis_id = (
				SELECT MAX(analysis_id) FROM analysis_results WHERE bridge_id = b.bridge_id
			)
		) mf ON b.bridge_id = mf.bridge_id
		WHERE b.bridge_id = $1
		GROUP BY b.bridge_id, b.name, b.span_length
	`

	err := database.DB.Get(&result, query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *BridgeHandler) GetLatestAnalysis(c *gin.Context) {
	id := c.Param("id")

	var analysis models.AnalysisResult
	query := `
		SELECT * FROM analysis_results
		WHERE bridge_id = $1 AND status = 'completed'
		ORDER BY analysis_time DESC
		LIMIT 1
	`

	err := database.DB.Get(&analysis, query, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No analysis found"})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

func (h *BridgeHandler) GetSensorData(c *gin.Context) {
	sensorID := c.Param("sensorId")
	startTime := c.DefaultQuery("start", time.Now().AddDate(0, 0, -7).Format(time.RFC3339))
	endTime := c.DefaultQuery("end", time.Now().Format(time.RFC3339))
	limit := c.DefaultQuery("limit", "1000")

	start, _ := time.Parse(time.RFC3339, startTime)
	end, _ := time.Parse(time.RFC3339, endTime)
	limitInt, _ := strconv.Atoi(limit)

	var data []models.SensorData
	query := `
		SELECT * FROM sensor_data
		WHERE sensor_id = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC
		LIMIT $4
	`

	err := database.DB.Select(&data, query, sensorID, start, end, limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(data),
		"data":  data,
	})
}

func (h *BridgeHandler) GetAlerts(c *gin.Context) {
	id := c.Param("id")
	limit := c.DefaultQuery("limit", "50")
	acknowledged := c.DefaultQuery("acknowledged", "false")

	limitInt, _ := strconv.Atoi(limit)

	var alerts []models.Alert
	query := `
		SELECT * FROM alerts
		WHERE bridge_id = $1
	`

	if acknowledged == "false" {
		query += " AND is_acknowledged = false"
	}

	query += " ORDER BY timestamp DESC LIMIT $" + strconv.Itoa(2)

	err := database.DB.Select(&alerts, query, id, limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(alerts),
		"data":  alerts,
	})
}

func (h *BridgeHandler) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("alertId")
	acknowledgedBy := c.DefaultQuery("by", "system")

	query := `
		UPDATE alerts
		SET is_acknowledged = true,
			acknowledged_at = CURRENT_TIMESTAMP,
			acknowledged_by = $1
		WHERE alert_id = $2
		RETURNING *
	`

	var alert models.Alert
	err := database.DB.Get(&alert, query, acknowledgedBy, alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

func (h *BridgeHandler) GetVehicleLoads(c *gin.Context) {
	var loads []models.VehicleLoad
	query := `SELECT * FROM vehicle_loads ORDER BY load_id`

	err := database.DB.Select(&loads, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, loads)
}

func (h *BridgeHandler) GetYingzaoSpecs(c *gin.Context) {
	var specs []models.YingzaoFashiSpec
	query := `SELECT * FROM yingzao_fashi_specs ORDER BY spec_id`

	err := database.DB.Select(&specs, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, specs)
}
