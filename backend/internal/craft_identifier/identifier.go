package craft_identifier

import (
	"log"
	"time"

	"ancient-bridge-system/internal/craft"
	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/messaging"
)

type CraftIdentifier struct {
	bus *messaging.MessageBus
}

func NewCraftIdentifier(bus *messaging.MessageBus) *CraftIdentifier {
	ci := &CraftIdentifier{bus: bus}
	go ci.watchRequests()
	return ci
}

func (ci *CraftIdentifier) watchRequests() {
	for {
		select {
		case msg := <-ci.bus.CraftAnalyzeReqChan:
			go ci.handleAnalyze(msg)
		}
	}
}

func (ci *CraftIdentifier) handleAnalyze(msg *messaging.Message) {
	req, ok := msg.Payload.(*messaging.CraftAnalyzeRequest)
	if !ok {
		log.Println("CraftIdentifier: Invalid CraftAnalyzeRequest payload")
		return
	}

	var woodFeatures *craft.WoodFeature
	var joineryFeature *craft.JoineryFeature

	if wf, ok := req.WoodFeatures.(*craft.WoodFeature); ok {
		woodFeatures = wf
	} else {
		woodFeatures = craft.GenerateTypicalWoodFeatures(req.WoodSpecies)
	}

	if jf, ok := req.JoineryFeature.(*craft.JoineryFeature); ok {
		joineryFeature = jf
	} else {
		joineryFeature = &craft.JoineryFeature{
			JointType:           "mortise_tenon",
			TenonLength:         0.15,
			TenonWidth:          0.08,
			TenonThickness:      0.04,
			MortiseDepth:        0.12,
			ShoulderAngle:       90.0,
			FitTolerance:        0.001,
			WoodSpecies:         req.WoodSpecies,
			CraftsmanshipRating: 0.85,
		}
	}

	result := craft.AnalyzeCraft(woodFeatures, joineryFeature, req.BridgeType)

	analysisID := ci.saveResult(req.BridgeID, result)

	resp := &messaging.CraftAnalyzeResponse{
		AnalysisID: analysisID,
		Result:     result,
	}

	respMsg := messaging.NewMessage(messaging.MsgTypeCraftAnalyzeResp, resp)
	respMsg.ReplyTo = msg.ReplyTo

	if msg.ReplyTo != nil {
		select {
		case msg.ReplyTo <- respMsg:
		default:
			log.Println("CraftIdentifier: Reply channel full for craft analyze")
		}
	}

	select {
	case ci.bus.CraftAnalyzeRespChan <- respMsg:
	default:
		log.Println("CraftIdentifier: CraftAnalyzeRespChan full")
	}
}

func (ci *CraftIdentifier) saveResult(bridgeID int, result *craft.CraftAnalysisResult) int {
	query := `
		INSERT INTO craft_analysis
		(bridge_id, wood_species, wood_grade, joinery_type, confidence_score, method_used, analysis_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING analysis_id
	`

	var id int
	err := database.DB.QueryRow(
		query,
		bridgeID,
		result.WoodSpecies,
		result.WoodGrade,
		result.JoineryType,
		result.ConfidenceScore,
		result.MethodUsed,
		time.Now(),
	).Scan(&id)

	if err != nil {
		log.Printf("CraftIdentifier: Failed to save craft analysis: %v", err)
		return 0
	}
	return id
}
