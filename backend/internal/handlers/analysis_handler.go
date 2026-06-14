package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ancient-bridge-system/internal/alarm_mqtt"
	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/fea"
	"ancient-bridge-system/internal/messaging"
	"ancient-bridge-system/internal/models"

	"github.com/gin-gonic/gin"
)

type AnalysisHandler struct {
	bus       *messaging.MessageBus
	alarmSvc  *alarm_mqtt.AlarmMQTTService
}

func NewAnalysisHandler(bus *messaging.MessageBus, alarmSvc *alarm_mqtt.AlarmMQTTService) *AnalysisHandler {
	return &AnalysisHandler{
		bus:      bus,
		alarmSvc: alarmSvc,
	}
}

type StaticLoadRequest struct {
	BridgeID     int     `json:"bridge_id" binding:"required"`
	LoadValue    float64 `json:"load_value" binding:"required"`
	LoadPosition float64 `json:"load_position"`
	AnalysisName string  `json:"analysis_name"`
	LoadCase     string  `json:"load_case"`
}

type MovingLoadRequest struct {
	BridgeID     int     `json:"bridge_id" binding:"required"`
	TotalWeight  float64 `json:"total_weight" binding:"required"`
	Steps        int     `json:"steps"`
	AnalysisName string  `json:"analysis_name"`
}

type AnalysisResponse struct {
	AnalysisID        int                     `json:"analysis_id"`
	BridgeID          int                     `json:"bridge_id"`
	AnalysisType      string                  `json:"analysis_type"`
	MemberForces      []fea.MemberForces      `json:"member_forces"`
	Displacements     []fea.NodeDisplacement  `json:"displacements"`
	MaxStressRatio    float64                 `json:"max_stress_ratio"`
	MaxDisplacement   float64                 `json:"max_displacement"`
	YingzaoComparison []fea.YingzaoComparison `json:"yingzao_comparison"`
}

type MovingLoadAnalysisResponse struct {
	AnalysisID      int                   `json:"analysis_id"`
	BridgeID        int                   `json:"bridge_id"`
	AnalysisType    string                `json:"analysis_type"`
	Results         []fea.MovingLoadResult `json:"results"`
	MaxStressRatio  float64               `json:"max_stress_ratio"`
	MaxDisplacement float64               `json:"max_displacement"`
}

func (h *AnalysisHandler) StaticLoadAnalysis(c *gin.Context) {
	var req StaticLoadRequest
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

	loadValue := req.LoadValue
	if loadValue <= 0 {
		loadValue = 50
	}

	if req.AnalysisName == "" {
		req.AnalysisName = "静载分析"
	}
	if req.LoadCase == "" {
		req.LoadCase = "deck_load"
	}

	busReq := &messaging.StaticLoadRequest{
		BridgeID:     req.BridgeID,
		LoadValue:    loadValue,
		LoadPosition: req.LoadPosition,
		AnalysisName: req.AnalysisName,
		LoadCase:     req.LoadCase,
	}
	busReq.BridgeInfo.SpanLength = bridge.SpanLength
	busReq.BridgeInfo.ArchRise = bridge.ArchRise
	busReq.BridgeInfo.DeckWidth = bridge.DeckWidth

	replyChan := make(chan *messaging.Message, 1)
	msg := messaging.NewMessage(messaging.MsgTypeStaticLoadReq, busReq)
	msg.ReplyTo = replyChan

	select {
	case h.bus.StaticLoadReqChan <- msg:
	case <-time.After(5 * time.Second):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Structural simulator busy"})
		return
	}

	select {
	case respMsg := <-replyChan:
		resp, ok := respMsg.Payload.(*messaging.StaticLoadResponse)
		if !ok || resp.Error != "" {
			errMsg := "Analysis failed"
			if resp != nil && resp.Error != "" {
				errMsg = resp.Error
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
			return
		}

		comparison, _ := resp.ComparisonResult.([]fea.YingzaoComparison)

		c.JSON(http.StatusOK, AnalysisResponse{
			AnalysisID:        resp.AnalysisID,
			BridgeID:          resp.BridgeID,
			AnalysisType:      "static",
			MemberForces:      resp.MemberForces,
			Displacements:     resp.Displacements,
			MaxStressRatio:    resp.MaxStressRatio,
			MaxDisplacement:   resp.MaxDisplacement,
			YingzaoComparison: comparison,
		})

	case <-time.After(30 * time.Second):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Analysis timeout"})
	}
}

func (h *AnalysisHandler) MovingLoadAnalysis(c *gin.Context) {
	var req MovingLoadRequest
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

	steps := req.Steps
	if steps <= 0 {
		steps = 20
	}
	if req.AnalysisName == "" {
		req.AnalysisName = "移动荷载分析"
	}

	busReq := &messaging.MovingLoadRequest{
		BridgeID:    req.BridgeID,
		TotalWeight: req.TotalWeight,
		Steps:       steps,
	}
	busReq.BridgeInfo.SpanLength = bridge.SpanLength
	busReq.BridgeInfo.ArchRise = bridge.ArchRise
	busReq.BridgeInfo.DeckWidth = bridge.DeckWidth

	replyChan := make(chan *messaging.Message, 1)
	msg := messaging.NewMessage(messaging.MsgTypeMovingLoadReq, busReq)
	msg.ReplyTo = replyChan

	select {
	case h.bus.MovingLoadReqChan <- msg:
	case <-time.After(5 * time.Second):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Structural simulator busy"})
		return
	}

	select {
	case respMsg := <-replyChan:
		resp, ok := respMsg.Payload.(*messaging.MovingLoadResponse)
		if !ok || resp.Error != "" {
			errMsg := "Analysis failed"
			if resp != nil && resp.Error != "" {
				errMsg = resp.Error
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
			return
		}

		c.JSON(http.StatusOK, MovingLoadAnalysisResponse{
			AnalysisID:      resp.AnalysisID,
			BridgeID:        resp.BridgeID,
			AnalysisType:    "moving",
			Results:         resp.Results,
			MaxStressRatio:  resp.MaxStressRatio,
			MaxDisplacement: resp.MaxDisplacement,
		})

	case <-time.After(60 * time.Second):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Moving load analysis timeout"})
	}
}

func (h *AnalysisHandler) GetStructure(c *gin.Context) {
	bridgeIDStr := c.Param("id")
	bridgeID, err := strconv.Atoi(bridgeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bridge ID"})
		return
	}

	var bridge models.Bridge
	err = database.DB.Get(&bridge, "SELECT * FROM bridges WHERE bridge_id = $1", bridgeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bridge not found"})
		return
	}

	busReq := &messaging.StructureRequest{
		BridgeID: bridgeID,
	}
	busReq.BridgeInfo.SpanLength = bridge.SpanLength
	busReq.BridgeInfo.ArchRise = bridge.ArchRise
	busReq.BridgeInfo.DeckWidth = bridge.DeckWidth

	replyChan := make(chan *messaging.Message, 1)
	msg := messaging.NewMessage(messaging.MsgTypeStructureReq, busReq)
	msg.ReplyTo = replyChan

	select {
	case h.bus.StructureReqChan <- msg:
	case <-time.After(5 * time.Second):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Structural simulator busy"})
		return
	}

	select {
	case respMsg := <-replyChan:
		resp, ok := respMsg.Payload.(*messaging.StructureResponse)
		if !ok || resp.Error != "" {
			errMsg := "Failed to get structure"
			if resp != nil && resp.Error != "" {
				errMsg = resp.Error
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"bridge_id":    resp.BridgeID,
			"bridge_name":  bridge.Name,
			"nodes":        resp.Nodes,
			"members":      resp.Members,
			"member_types": resp.MemberTypes,
		})

	case <-time.After(30 * time.Second):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Structure request timeout"})
	}
}

func (h *AnalysisHandler) GetAnalysisHistory(c *gin.Context) {
	bridgeID := c.Param("id")
	limit := c.DefaultQuery("limit", "20")

	limitInt, _ := strconv.Atoi(limit)

	var analyses []models.AnalysisResult
	query := `
		SELECT * FROM analysis_results
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

func (h *AnalysisHandler) AcknowledgeAlert(c *gin.Context) {
	alertIDStr := c.Param("alertId")
	alertID, _ := strconv.Atoi(alertIDStr)
	acknowledgedBy := c.DefaultQuery("acknowledged_by", "system")

	err := h.alarmSvc.AcknowledgeAlert(alertID, acknowledgedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
