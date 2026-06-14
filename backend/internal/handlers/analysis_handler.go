package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ancient-bridge-system/internal/alert"
	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/fea"
	"ancient-bridge-system/internal/models"

	"github.com/gin-gonic/gin"
)

type AnalysisHandler struct{}

func NewAnalysisHandler() *AnalysisHandler {
	return &AnalysisHandler{}
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
	AnalysisID    int                     `json:"analysis_id"`
	BridgeID      int                     `json:"bridge_id"`
	AnalysisType  string                  `json:"analysis_type"`
	MemberForces  []fea.MemberForces      `json:"member_forces"`
	Displacements []fea.NodeDisplacement  `json:"displacements"`
	MaxStressRatio float64                `json:"max_stress_ratio"`
	MaxDisplacement float64               `json:"max_displacement"`
	YingzaoComparison []fea.YingzaoComparison `json:"yingzao_comparison"`
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

	bridgeModel := fea.GenerateArchBridge(
		req.BridgeID,
		bridge.SpanLength,
		bridge.ArchRise,
		bridge.DeckWidth,
	)

	loadPosition := req.LoadPosition
	if loadPosition <= 0 {
		loadPosition = bridge.SpanLength / 2
	}

	memberForces, displacements := bridgeModel.AnalyzeStaticLoad(req.LoadValue, loadPosition)

	var specs []fea.YingzaoSpec
	database.DB.Select(&specs, "SELECT component_type, grade_level, material_grade, max_span_ratio, min_section_modulus, allowable_stress, safety_factor FROM yingzao_fashi_specs")

	comparisons := bridgeModel.CompareWithYingzaoFashi(memberForces, specs)

	analysisName := req.AnalysisName
	if analysisName == "" {
		analysisName = "静载分析_" + time.Now().Format("20060102_150405")
	}

	loadCase := req.LoadCase
	if loadCase == "" {
		loadCase = "集中荷载"
	}

	var analysisID int
	err = database.DB.QueryRow(`
		INSERT INTO analysis_results 
		(bridge_id, analysis_type, analysis_name, load_case, load_value, load_position, is_moving_load, status)
		VALUES ($1, 'static', $2, $3, $4, $5, false, 'completed')
		RETURNING analysis_id
	`, req.BridgeID, analysisName, loadCase, req.LoadValue, loadPosition).Scan(&analysisID)

	if err == nil {
		go saveMemberForces(analysisID, memberForces)
		go saveNodeDisplacements(analysisID, displacements)
		go checkStressAlerts(req.BridgeID, memberForces, bridgeModel)
	}

	maxStressRatio := 0.0
	for _, mf := range memberForces {
		ratio := 0.0
		for _, comp := range comparisons {
			if comp.MemberID == mf.MemberID {
				ratio = comp.StressRatio
				break
			}
		}
		if ratio > maxStressRatio {
			maxStressRatio = ratio
		}
	}

	maxDisp := 0.0
	for _, d := range displacements {
		if d.TotalDisp > maxDisp {
			maxDisp = d.TotalDisp
		}
	}

	c.JSON(http.StatusOK, AnalysisResponse{
		AnalysisID:        analysisID,
		BridgeID:          req.BridgeID,
		AnalysisType:      "static",
		MemberForces:      memberForces,
		Displacements:     displacements,
		MaxStressRatio:    maxStressRatio,
		MaxDisplacement:   maxDisp,
		YingzaoComparison: comparisons,
	})
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

	bridgeModel := fea.GenerateArchBridge(
		req.BridgeID,
		bridge.SpanLength,
		bridge.ArchRise,
		bridge.DeckWidth,
	)

	results := bridgeModel.AnalyzeMovingLoad(req.TotalWeight, steps)

	analysisName := req.AnalysisName
	if analysisName == "" {
		analysisName = "移动荷载分析_" + time.Now().Format("20060102_150405")
	}

	var analysisID int
	err = database.DB.QueryRow(`
		INSERT INTO analysis_results 
		(bridge_id, analysis_type, analysis_name, load_case, load_value, is_moving_load, status)
		VALUES ($1, 'moving', $2, '移动荷载', $3, true, 'completed')
		RETURNING analysis_id
	`, req.BridgeID, analysisName, req.TotalWeight).Scan(&analysisID)

	maxStressRatio := 0.0
	maxDisplacement := 0.0

	for _, result := range results {
		for _, mf := range result.MemberForces {
			if mf.AxialStress > 1.0 && mf.AxialStress/8.5 > maxStressRatio {
				maxStressRatio = mf.AxialStress / 8.5
			}
		}
		for _, d := range result.Displacements {
			if d.TotalDisp > maxDisplacement {
				maxDisplacement = d.TotalDisp
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis_id":      analysisID,
		"bridge_id":        req.BridgeID,
		"analysis_type":    "moving",
		"steps":            len(results),
		"results":          results,
		"max_stress_ratio": maxStressRatio,
		"max_displacement": maxDisplacement,
	})
}

func (h *AnalysisHandler) GetAnalysisHistory(c *gin.Context) {
	bridgeID := c.Param("id")
	limit := c.DefaultQuery("limit", "20")

	limitInt, _ := strconv.Atoi(limit)

	var analyses []models.AnalysisResult
	query := `
		SELECT * FROM analysis_results
		WHERE bridge_id = $1
		ORDER BY analysis_time DESC
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

func (h *AnalysisHandler) GetAnalysisDetail(c *gin.Context) {
	analysisID := c.Param("analysisId")

	var analysis models.AnalysisResult
	err := database.DB.Get(&analysis, "SELECT * FROM analysis_results WHERE analysis_id = $1", analysisID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Analysis not found"})
		return
	}

	var memberForces []models.MemberForce
	database.DB.Select(&memberForces, "SELECT * FROM member_forces WHERE analysis_id = $1", analysisID)

	var displacements []models.NodeDisplacement
	database.DB.Select(&displacements, "SELECT * FROM node_displacements WHERE analysis_id = $1", analysisID)

	c.JSON(http.StatusOK, gin.H{
		"analysis":       analysis,
		"member_forces":  memberForces,
		"displacements":  displacements,
	})
}

func (h *AnalysisHandler) GetBridgeStructure(c *gin.Context) {
	bridgeID := c.Param("id")

	var bridge models.Bridge
	err := database.DB.Get(&bridge, "SELECT * FROM bridges WHERE bridge_id = $1", bridgeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bridge not found"})
		return
	}

	bridgeModel := fea.GenerateArchBridge(
		bridge.BridgeID,
		bridge.SpanLength,
		bridge.ArchRise,
		bridge.DeckWidth,
	)

	nodes := make([]map[string]interface{}, 0)
	for _, node := range bridgeModel.Structure.Nodes {
		nodes = append(nodes, map[string]interface{}{
			"node_id":   node.ID,
			"x":         node.X,
			"y":         node.Y,
			"fixed_x":   node.FixedX,
			"fixed_y":   node.FixedY,
			"fixed_r":   node.FixedR,
		})
	}

	members := make([]map[string]interface{}, 0)
	for _, member := range bridgeModel.Structure.Members {
		members = append(members, map[string]interface{}{
			"member_id":   member.ID,
			"start_node":  member.Start.ID,
			"end_node":    member.End.ID,
			"length":      member.Length,
			"angle":       member.Angle,
			"type":        bridgeModel.MemberTypes[member.ID],
			"area":        member.A,
			"inertia":     member.I,
			"elastic_modulus": member.E,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"bridge_id":    bridge.BridgeID,
		"name":         bridge.Name,
		"span_length":  bridge.SpanLength,
		"arch_rise":    bridge.ArchRise,
		"deck_width":   bridge.DeckWidth,
		"nodes":        nodes,
		"members":      members,
		"member_types": bridgeModel.MemberTypes,
	})
}

func saveMemberForces(analysisID int, forces []fea.MemberForces) {
	for _, f := range forces {
		combinedStress := f.AxialStress + f.BendingStress
		stressRatio := 0.0
		allowable := 8.5
		if allowable > 0 {
			stressRatio = combinedStress / allowable
		}

		database.DB.Exec(`
			INSERT INTO member_forces 
			(analysis_id, member_id, axial_force, shear_force, bending_moment,
				axial_stress, bending_stress, combined_stress, stress_ratio, is_overspeed)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, analysisID, f.MemberID, f.AxialForce, f.ShearForce, f.BendingMoment,
			f.AxialStress, f.BendingStress, combinedStress, stressRatio, stressRatio > 1.0)
	}
}

func saveNodeDisplacements(analysisID int, displacements []fea.NodeDisplacement) {
	for _, d := range displacements {
		database.DB.Exec(`
			INSERT INTO node_displacements
			(analysis_id, node_id, displacement_x, displacement_y, displacement_z, total_displacement)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, analysisID, d.NodeID, d.UX, d.UY, 0, d.TotalDisp)
	}
}

func checkStressAlerts(bridgeID int, forces []fea.MemberForces, bridgeModel *fea.BridgeModel) {
	if alert.GlobalAlertService == nil {
		return
	}

	for _, f := range forces {
		combinedStress := f.AxialStress + f.BendingStress
		allowable := 8.5
		stressRatio := combinedStress / allowable

		memberCode := "M" + strconv.Itoa(f.MemberID)

		alert.GlobalAlertService.CheckAndAlertMemberStress(
			bridgeID,
			f.MemberID,
			stressRatio,
			combinedStress,
			allowable,
			memberCode,
		)
	}
}
