package structural_simulator

import (
	"log"
	"math"
	"sync"

	"ancient-bridge-system/internal/database"
	"ancient-bridge-system/internal/fea"
	"ancient-bridge-system/internal/messaging"
)

type StructuralSimulator struct {
	bus      *messaging.MessageBus
	modelMap map[int]*fea.TrussStructure
	mapMu    sync.RWMutex
}

func NewStructuralSimulator(bus *messaging.MessageBus) *StructuralSimulator {
	sim := &StructuralSimulator{
		bus:      bus,
		modelMap: make(map[int]*fea.TrussStructure),
	}
	go sim.watchRequests()
	return sim
}

func (s *StructuralSimulator) watchRequests() {
	for {
		select {
		case msg := <-s.bus.StaticLoadReqChan:
			go s.handleStaticLoad(msg)

		case msg := <-s.bus.MovingLoadReqChan:
			go s.handleMovingLoad(msg)

		case msg := <-s.bus.StructureReqChan:
			go s.handleStructureReq(msg)
		}
	}
}

func (s *StructuralSimulator) getOrCreateModel(bridgeID int, span, rise, width float64) *fea.TrussStructure {
	s.mapMu.RLock()
	m, ok := s.modelMap[bridgeID]
	s.mapMu.RUnlock()
	if ok {
		return m
	}

	ts := fea.GenerateArchBridge(span, rise, width, 10000.0, 0.12, 0.0015)

	s.mapMu.Lock()
	s.modelMap[bridgeID] = ts
	s.mapMu.Unlock()
	return ts
}

func (s *StructuralSimulator) handleStaticLoad(msg *messaging.Message) {
	req, ok := msg.Payload.(*messaging.StaticLoadRequest)
	if !ok {
		log.Println("StructuralSimulator: Invalid StaticLoadRequest payload")
		return
	}

	ts := s.getOrCreateModel(req.BridgeID, req.BridgeInfo.SpanLength, req.BridgeInfo.ArchRise, req.BridgeInfo.DeckWidth)

	posIdx := int(req.LoadPosition * float64(len(ts.Nodes)-1))
	if posIdx < 0 {
		posIdx = 0
	}
	if posIdx >= len(ts.Nodes) {
		posIdx = len(ts.Nodes) - 1
	}

	ts.ClearLoads()
	ts.Nodes[posIdx].FY = -req.LoadValue

	err := ts.Solve()
	resp := &messaging.StaticLoadResponse{
		BridgeID: req.BridgeID,
	}

	if err != nil {
		resp.Error = err.Error()
	} else {
		memberForces := ts.CalculateMemberForces()
		displacements := s.buildDisplacements(ts)

		maxStressRatio := 0.0
		for _, mf := range memberForces {
			if mf.StressRatio > maxStressRatio {
				maxStressRatio = mf.StressRatio
			}
		}

		maxDisp := 0.0
		for _, n := range ts.Nodes {
			disp := math.Sqrt(n.UX*n.UX + n.UY*n.UY)
			if disp > maxDisp {
				maxDisp = disp
			}
		}

		allowable := 8.5
		comparison := ts.CompareWithYingzaoFashi(allowable, req.LoadCase)

		analysisID := s.saveAnalysis(req.BridgeID, req.AnalysisName, req.LoadCase, req.LoadValue, maxStressRatio, maxDisp)

		go s.saveMemberForcesAsync(analysisID, memberForces, displacements, req.BridgeID, maxStressRatio, allowable, ts)

		resp.AnalysisID = analysisID
		resp.MemberForces = memberForces
		resp.Displacements = displacements
		resp.MaxStressRatio = maxStressRatio
		resp.MaxDisplacement = maxDisp
		resp.ComparisonResult = comparison
	}

	respMsg := messaging.NewMessage(messaging.MsgTypeStaticLoadResp, resp)
	respMsg.ReplyTo = msg.ReplyTo

	if msg.ReplyTo != nil {
		select {
		case msg.ReplyTo <- respMsg:
		default:
			log.Println("StructuralSimulator: Reply channel full for static load")
		}
	}

	select {
	case s.bus.StaticLoadRespChan <- respMsg:
	default:
		log.Println("StructuralSimulator: StaticLoadRespChan full")
	}
}

func (s *StructuralSimulator) handleMovingLoad(msg *messaging.Message) {
	req, ok := msg.Payload.(*messaging.MovingLoadRequest)
	if !ok {
		log.Println("StructuralSimulator: Invalid MovingLoadRequest payload")
		return
	}

	ts := s.getOrCreateModel(req.BridgeID, req.BridgeInfo.SpanLength, req.BridgeInfo.ArchRise, req.BridgeInfo.DeckWidth)

	results := make([]fea.MovingLoadResult, 0)
	maxStressRatio := 0.0
	maxDisp := 0.0
	steps := req.Steps
	if steps <= 0 {
		steps = 20
	}

	for i := 0; i < steps; i++ {
		pos := float64(i) / float64(steps-1)
		posIdx := int(pos * float64(len(ts.Nodes)-1))
		if posIdx >= len(ts.Nodes) {
			posIdx = len(ts.Nodes) - 1
		}

		ts.ClearLoads()
		ts.Nodes[posIdx].FY = -req.TotalWeight

		err := ts.Solve()
		if err != nil {
			continue
		}

		memberForces := ts.CalculateMemberForces()
		for j := range memberForces {
			if memberForces[j].StressRatio > maxStressRatio {
				maxStressRatio = memberForces[j].StressRatio
			}
		}

		for _, n := range ts.Nodes {
			disp := math.Sqrt(n.UX*n.UX + n.UY*n.UY)
			if disp > maxDisp {
				maxDisp = disp
			}
		}

		results = append(results, fea.MovingLoadResult{
			Position:     pos,
			MemberForces: memberForces,
		})
	}

	allowable := 8.5
	analysisID := s.saveAnalysis(req.BridgeID, "移动荷载分析", "moving", req.TotalWeight, maxStressRatio, maxDisp)

	resp := &messaging.MovingLoadResponse{
		AnalysisID:      analysisID,
		BridgeID:        req.BridgeID,
		Results:         results,
		MaxStressRatio:  maxStressRatio,
		MaxDisplacement: maxDisp,
	}

	go func() {
		for _, r := range results {
			for _, mf := range r.MemberForces {
				s.checkAndSendAlert(analysisID, req.BridgeID, mf, allowable)
			}
		}
	}()

	respMsg := messaging.NewMessage(messaging.MsgTypeMovingLoadResp, resp)
	respMsg.ReplyTo = msg.ReplyTo

	if msg.ReplyTo != nil {
		select {
		case msg.ReplyTo <- respMsg:
		default:
			log.Println("StructuralSimulator: Reply channel full for moving load")
		}
	}

	select {
	case s.bus.MovingLoadRespChan <- respMsg:
	default:
		log.Println("StructuralSimulator: MovingLoadRespChan full")
	}
}

func (s *StructuralSimulator) handleStructureReq(msg *messaging.Message) {
	req, ok := msg.Payload.(*messaging.StructureRequest)
	if !ok {
		log.Println("StructuralSimulator: Invalid StructureRequest payload")
		return
	}

	ts := s.getOrCreateModel(req.BridgeID, req.BridgeInfo.SpanLength, req.BridgeInfo.ArchRise, req.BridgeInfo.DeckWidth)

	nodes := make([]map[string]interface{}, len(ts.Nodes))
	for i, n := range ts.Nodes {
		nodes[i] = map[string]interface{}{
			"id":    n.ID,
			"x":     n.X,
			"y":     n.Y,
			"ux":    n.UX,
			"uy":    n.UY,
			"rz":    n.RZ,
			"fix_x": n.FixX,
			"fix_y": n.FixY,
			"fix_r": n.FixR,
		}
	}

	members := make([]map[string]interface{}, len(ts.Members))
	memberTypes := make(map[int]string)
	for i, m := range ts.Members {
		members[i] = map[string]interface{}{
			"id":            m.ID,
			"start_node_id": m.StartNodeID,
			"end_node_id":   m.EndNodeID,
			"area":          m.A,
			"inertia":       m.I,
			"length":        m.L,
			"angle":         m.Theta,
			"member_type":   m.MemberType,
			"member_code":   m.MemberCode,
		}
		memberTypes[m.ID] = m.MemberType
	}

	resp := &messaging.StructureResponse{
		BridgeID:    req.BridgeID,
		Nodes:       nodes,
		Members:     members,
		MemberTypes: memberTypes,
	}

	respMsg := messaging.NewMessage(messaging.MsgTypeStructureResp, resp)
	respMsg.ReplyTo = msg.ReplyTo

	if msg.ReplyTo != nil {
		select {
		case msg.ReplyTo <- respMsg:
		default:
			log.Println("StructuralSimulator: Reply channel full for structure")
		}
	}

	select {
	case s.bus.StructureRespChan <- respMsg:
	default:
		log.Println("StructuralSimulator: StructureRespChan full")
	}
}

func (s *StructuralSimulator) buildDisplacements(ts *fea.TrussStructure) []fea.NodeDisplacement {
	disps := make([]fea.NodeDisplacement, len(ts.Nodes))
	for i, n := range ts.Nodes {
		disps[i] = fea.NodeDisplacement{
			NodeID: n.ID,
			UX:     n.UX,
			UY:     n.UY,
			RZ:     n.RZ,
		}
	}
	return disps
}

func (s *StructuralSimulator) saveAnalysis(bridgeID int, name, loadCase string, loadValue, maxStressRatio, maxDisp float64) int {
	query := `
		INSERT INTO analysis_results
		(bridge_id, analysis_name, analysis_type, load_case, load_value, max_stress_ratio, max_displacement, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'completed')
		RETURNING analysis_id
	`

	var id int
	err := database.DB.QueryRow(query, bridgeID, name, "static", loadCase, loadValue, maxStressRatio, maxDisp).Scan(&id)
	if err != nil {
		log.Printf("StructuralSimulator: Failed to save analysis: %v", err)
		return 0
	}
	return id
}

func (s *StructuralSimulator) saveMemberForcesAsync(analysisID int, memberForces []fea.MemberForces, disps []fea.NodeDisplacement, bridgeID int, maxStressRatio, allowable float64, ts *fea.TrussStructure) {
	for _, mf := range memberForces {
		query := `
			INSERT INTO member_forces
			(analysis_id, member_id, axial_force, shear_force, bending_moment, axial_stress, bending_stress, stress_ratio)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, _ = database.DB.Exec(query, analysisID, mf.MemberID, mf.AxialForce, mf.ShearForce, mf.BendingMoment, mf.AxialStress, mf.BendingStress, mf.StressRatio)

		s.checkAndSendAlert(analysisID, bridgeID, mf, allowable)
	}

	for _, d := range disps {
		query := `
			INSERT INTO node_displacements
			(analysis_id, node_id, ux, uy, rz)
			VALUES ($1, $2, $3, $4, $5)
		`
		_, _ = database.DB.Exec(query, analysisID, d.NodeID, d.UX, d.UY, d.RZ)
	}
}

func (s *StructuralSimulator) checkAndSendAlert(analysisID, bridgeID int, mf fea.MemberForces, allowable float64) {
	var memberCode string
	for _, m := range s.getOrCreateModel(bridgeID, 25, 5.5, 4).Members {
		if m.ID == mf.MemberID {
			memberCode = m.MemberCode
			break
		}
	}

	measured := mf.AxialStress
	if math.Abs(mf.BendingStress) > math.Abs(measured) {
		measured = mf.BendingStress
	}

	if mf.StressRatio >= 0.8 {
		alertLevel := "warning"
		alertType := "stress_warning"
		alertMsg := ""
		if mf.StressRatio >= 1.0 {
			alertLevel = "danger"
			alertType = "stress_exceeded"
			alertMsg = "构件应力超限"
		} else {
			alertMsg = "构件应力接近容许值"
		}

		alert := &messaging.AlertPayload{
			AlertID:        0,
			BridgeID:       bridgeID,
			MemberID:       &mf.MemberID,
			MemberCode:     memberCode,
			AlertType:      alertType,
			AlertLevel:     alertLevel,
			AlertMessage:   alertMsg,
			MeasuredValue:  measured,
			ThresholdValue: allowable,
			Unit:           "MPa",
		}

		msg := messaging.NewMessage(messaging.MsgTypeAlert, alert)
		select {
		case s.bus.AlertChan <- msg:
		default:
			log.Printf("StructuralSimulator: AlertChan full, dropping alert for member %d", mf.MemberID)
		}
	}
}
