package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ancient-bridge-system/internal/craft"
	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/models"
	"ancient-bridge-system/internal/sensor"

	"github.com/gin-gonic/gin"
)

type CraftHandler struct{}

func NewCraftHandler() *CraftHandler {
	return &CraftHandler{}
}

type CraftAnalysisRequest struct {
	BridgeID          int     `json:"bridge_id" binding:"required"`
	WoodSpecies       string  `json:"wood_species"`
	GrainDensity      float64 `json:"grain_density"`
	GrainAngle        float64 `json:"grain_angle"`
	LatewoodRatio     float64 `json:"latewood_ratio"`
	KnotsCount        float64 `json:"knots_count"`
	AverageKnotSize   float64 `json:"average_knot_size"`
	Density           float64 `json:"density"`
	Hardness          float64 `json:"hardness"`
	JointType         string  `json:"joint_type"`
	CraftsmanshipRating float64 `json:"craftsmanship_rating"`
}

func (h *CraftHandler) AnalyzeCraft(c *gin.Context) {
	var req CraftAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var bridge models.Bridge
	err := database.DB.Get(&bridge, "SELECT * FROM bridges WHERE bridge_id = $1", req.BridgeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bridge not found"})
		return
	}

	woodFeatures := &craft.WoodFeature{
		GrainDensity:    req.GrainDensity,
		GrainAngle:      req.GrainAngle,
		LatewoodRatio:   req.LatewoodRatio,
		KnotsCount:      req.KnotsCount,
		AverageKnotSize: req.AverageKnotSize,
		Density:         req.Density,
		Hardness:        req.Hardness,
	}

	if req.WoodSpecies != "" {
		woodFeatures = craft.GenerateTypicalWoodFeatures(req.WoodSpecies)
	} else if req.Density == 0 {
		woodFeatures = craft.GenerateTypicalWoodFeatures("杉木")
	}

	joineryFeatures := &craft.JoineryFeature{
		JointType:           req.JointType,
		CraftsmanshipRating: req.CraftsmanshipRating,
	}

	if req.CraftsmanshipRating == 0 {
		joineryFeatures.CraftsmanshipRating = 3.5
	}

	result := craft.AnalyzeCraft(woodFeatures, joineryFeatures, bridge.ConstructionMethod)

	var analysisID int
	err = database.DB.QueryRow(`
		INSERT INTO craft_analysis 
		(bridge_id, wood_species_predicted, wood_grade_predicted, construction_sequence,
			joinery_type_predicted, confidence_score, method_used)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING analysis_id
	`, req.BridgeID, result.WoodSpecies, result.WoodGrade, result.ConstructionSequence,
		result.JoineryType, result.ConfidenceScore, result.MethodUsed).Scan(&analysisID)

	if err == nil {
		go saveWoodTextureFeatures(req.BridgeID, woodFeatures)
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis_id":   analysisID,
		"bridge_id":     req.BridgeID,
		"bridge_name":   bridge.Name,
		"wood_species":  result.WoodSpecies,
		"wood_grade":    result.WoodGrade,
		"construction_sequence": result.ConstructionSequence,
		"joinery_type":  result.JoineryType,
		"confidence_score": result.ConfidenceScore,
		"feature_importance": result.FeatureImportance,
		"method_used":   result.MethodUsed,
		"wood_features": woodFeatures,
	})
}

func (h *CraftHandler) GetCraftHistory(c *gin.Context) {
	bridgeID := c.Param("id")
	limit := c.DefaultQuery("limit", "10")

	limitInt, _ := strconv.Atoi(limit)

	var analyses []models.CraftAnalysis
	query := `
		SELECT * FROM craft_analysis
		WHERE bridge_id = $1
		ORDER BY analysis_date DESC
		LIMIT $2
	`

	err := database.DB.Select(&analyses, query, bridgeID, limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(analyses),
		"data":  analyses,
	})
}

func (h *CraftHandler) GetWoodSpeciesList(c *gin.Context) {
	species := []map[string]interface{}{
		{"name": "杉木", "density": 0.42, "hardness": 2.5, "grade": "二等材", "description": "中国南方常用建筑木材，质轻易加工"},
		{"name": "松木", "density": 0.48, "hardness": 3.0, "grade": "二等材", "description": "分布广泛的针叶材，强度适中"},
		{"name": "柏木", "density": 0.55, "hardness": 3.8, "grade": "一等材", "description": "耐腐性强，常用于重要建筑"},
		{"name": "樟木", "density": 0.58, "hardness": 4.2, "grade": "一等材", "description": "香气浓郁，防虫防蛀"},
		{"name": "楠木", "density": 0.62, "hardness": 4.8, "grade": "一等材", "description": "名贵木材，纹理美观"},
		{"name": "黄花梨", "density": 0.85, "hardness": 6.5, "grade": "特等材", "description": "名贵硬木，质地细密"},
		{"name": "紫檀", "density": 1.05, "hardness": 8.5, "grade": "特等材", "description": "最名贵硬木之一，密度极高"},
		{"name": "铁力木", "density": 0.95, "hardness": 7.2, "grade": "特等材", "description": "质地坚硬，强度极高"},
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(species),
		"data":  species,
	})
}

func (h *CraftHandler) GetJoineryTypes(c *gin.Context) {
	joineryTypes := []map[string]interface{}{
		{"name": "燕尾榫", "category": "搭接", "description": "木构件横向连接的经典榫卯", "strength": "高"},
		{"name": "齐肩榫", "category": "对接", "description": "梁柱连接常用榫卯", "strength": "中"},
		{"name": "平肩榫", "category": "对接", "description": "简单的平肩对接榫", "strength": "低"},
		{"name": "搭掌榫", "category": "斜接", "description": "斜向构件连接榫卯", "strength": "中"},
		{"name": "榫卯结合", "category": "综合", "description": "多种榫卯组合连接", "strength": "高"},
		{"name": "十字榫", "category": "交叉", "description": "交叉构件连接榫卯", "strength": "中"},
		{"name": "霸王拳", "category": "装饰", "description": "梁头装饰性榫头", "strength": "低"},
		{"name": "蚂蚱头", "category": "装饰", "description": "斗拱构件装饰榫头", "strength": "低"},
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(joineryTypes),
		"data":  joineryTypes,
	})
}

func saveWoodTextureFeatures(bridgeID int, features *craft.WoodFeature) {
	database.DB.Exec(`
		INSERT INTO wood_texture_features
		(bridge_id, grain_density, grain_angle, latewood_ratio, knots_count,
			average_knot_size, density, hardness)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, bridgeID, features.GrainDensity, features.GrainAngle, features.LatewoodRatio,
		int(features.KnotsCount), features.AverageKnotSize, features.Density, features.Hardness)
}

type SensorDataHandler struct {
	ingestor *sensor.SensorDataIngestor
}

func NewSensorDataHandler() *SensorDataHandler {
	return &SensorDataHandler{
		ingestor: sensor.NewSensorDataIngestor(),
	}
}

type DTUIngestRequest struct {
	DTUDeviceID string                 `json:"dtu_device_id" binding:"required"`
	Timestamp   string                 `json:"timestamp"`
	Sensors     []sensor.SensorReading `json:"sensors" binding:"required"`
	RawData     interface{}            `json:"raw_data"`
}

func (h *SensorDataHandler) IngestDTUData(c *gin.Context) {
	var req DTUIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid DTU payload"})
		return
	}

	timestamp := req.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	}

	payload := sensor.DTUPayload{
		DTUDeviceID: req.DTUDeviceID,
		Timestamp:   timestamp,
		Sensors:     req.Sensors,
		RawData:     req.RawData,
	}

	err := h.ingestor.IngestDTUData(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Data ingested successfully",
		"sensors": len(req.Sensors),
	})
}

func (h *SensorDataHandler) GetSensorData(c *gin.Context) {
	sensorID := c.Param("sensorId")
	startTime := c.DefaultQuery("start", time.Now().AddDate(0, 0, -7).Format(time.RFC3339))
	endTime := c.DefaultQuery("end", time.Now().Format(time.RFC3339))
	limit := c.DefaultQuery("limit", "1000")

	start, _ := time.Parse(time.RFC3339, startTime)
	end, _ := time.Parse(time.RFC3339, endTime)
	limitInt, _ := strconv.Atoi(limit)

	data, err := sensor.GetSensorData(sensorIDToInt(sensorID), start, end, limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(data),
		"data":  data,
	})
}

func (h *SensorDataHandler) GetLatestSensorData(c *gin.Context) {
	sensorID := c.Param("sensorId")

	data, err := sensor.GetLatestSensorData(sensorIDToInt(sensorID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No data found"})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *SensorDataHandler) GetEnvironmentalData(c *gin.Context) {
	bridgeID := c.Param("id")
	startTime := c.DefaultQuery("start", time.Now().AddDate(0, 0, -7).Format(time.RFC3339))
	endTime := c.DefaultQuery("end", time.Now().Format(time.RFC3339))

	start, _ := time.Parse(time.RFC3339, startTime)
	end, _ := time.Parse(time.RFC3339, endTime)
	bridgeIDInt, _ := strconv.Atoi(bridgeID)

	data, err := sensor.GetEnvironmentalData(bridgeIDInt, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(data),
		"data":  data,
	})
}

func sensorIDToInt(s string) int {
	id, _ := strconv.Atoi(s)
	return id
}
