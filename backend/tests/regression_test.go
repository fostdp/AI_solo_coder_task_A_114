package tests

import (
	"math"
	"sync"
	"testing"
	"time"

	"ancient-bridge-system/internal/craft"
	"ancient-bridge-system/internal/fea"
	"ancient-bridge-system/internal/messaging"
)

func TestMessageBusCreation(t *testing.T) {
	bus := messaging.NewMessageBus()
	defer bus.Close()

	if bus == nil {
		t.Fatal("MessageBus should not be nil")
	}
	if bus.SensorDataChan == nil {
		t.Error("SensorDataChan should not be nil")
	}
	if bus.StaticLoadReqChan == nil {
		t.Error("StaticLoadReqChan should not be nil")
	}
	if bus.AlertChan == nil {
		t.Error("AlertChan should not be nil")
	}
}

func TestMessageCreation(t *testing.T) {
	payload := &messaging.SensorDataPayload{
		BridgeID:    1,
		SensorCode:  "DISP-001",
		Value:       1.5,
		QualityFlag: 0,
	}

	msg := messaging.NewMessage(messaging.MsgTypeSensorData, payload)

	if msg.Type != messaging.MsgTypeSensorData {
		t.Errorf("Expected MsgTypeSensorData, got %v", msg.Type)
	}
	if msg.ID == "" {
		t.Error("Message ID should not be empty")
	}

	p, ok := msg.Payload.(*messaging.SensorDataPayload)
	if !ok {
		t.Fatal("Payload should be SensorDataPayload")
	}
	if p.BridgeID != 1 {
		t.Errorf("Expected BridgeID 1, got %d", p.BridgeID)
	}
	if math.Abs(p.Value-1.5) > 1e-9 {
		t.Errorf("Expected Value 1.5, got %f", p.Value)
	}
}

func TestMessageBusChannelCommunication(t *testing.T) {
	bus := messaging.NewMessageBus()
	defer bus.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		case msg := <-bus.SensorDataChan:
			if msg.Type != messaging.MsgTypeSensorData {
				t.Errorf("Expected MsgTypeSensorData, got %v", msg.Type)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for SensorDataChan")
		}
	}()

	msg := messaging.NewMessage(messaging.MsgTypeSensorData, &messaging.SensorDataPayload{})
	bus.SensorDataChan <- msg

	wg.Wait()
}

func TestMessageBusStaticLoadReqResp(t *testing.T) {
	bus := messaging.NewMessageBus()
	defer bus.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	replyChan := make(chan *messaging.Message, 1)

	go func() {
		defer wg.Done()
		select {
		case reqMsg := <-bus.StaticLoadReqChan:
			req, ok := reqMsg.Payload.(*messaging.StaticLoadRequest)
			if !ok {
				t.Error("Payload should be StaticLoadRequest")
				return
			}
			if req.BridgeID != 42 {
				t.Errorf("Expected BridgeID 42, got %d", req.BridgeID)
			}

			respMsg := messaging.NewMessage(messaging.MsgTypeStaticLoadResp, &messaging.StaticLoadResponse{
				BridgeID:       req.BridgeID,
				MaxStressRatio: 0.85,
			})
			if reqMsg.ReplyTo != nil {
				reqMsg.ReplyTo <- respMsg
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for StaticLoadReqChan")
		}
	}()

	reqMsg := messaging.NewMessage(messaging.MsgTypeStaticLoadReq, &messaging.StaticLoadRequest{
		BridgeID:  42,
		LoadValue: 50.0,
	})
	reqMsg.ReplyTo = replyChan
	bus.StaticLoadReqChan <- reqMsg

	select {
	case respMsg := <-replyChan:
		resp, ok := respMsg.Payload.(*messaging.StaticLoadResponse)
		if !ok {
			t.Fatal("Response payload should be StaticLoadResponse")
		}
		if resp.BridgeID != 42 {
			t.Errorf("Expected response BridgeID 42, got %d", resp.BridgeID)
		}
		if math.Abs(resp.MaxStressRatio-0.85) > 1e-9 {
			t.Errorf("Expected MaxStressRatio 0.85, got %f", resp.MaxStressRatio)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for response")
	}

	wg.Wait()
}

func TestTrussStructureGeneration(t *testing.T) {
	ts := fea.GenerateArchBridge(25.0, 5.5, 4.0, 10000.0, 0.12, 0.0015)

	if len(ts.Nodes) == 0 {
		t.Fatal("No nodes generated")
	}
	if len(ts.Members) == 0 {
		t.Fatal("No members generated")
	}

	nodeZero := ts.Nodes[0]
	if !nodeZero.FixX || !nodeZero.FixY || !nodeZero.FixR {
		t.Error("First node should be fully constrained")
	}

	t.Logf("Generated %d nodes, %d members", len(ts.Nodes), len(ts.Members))
}

func TestTrussStiffnessMatrix(t *testing.T) {
	ts := fea.GenerateArchBridge(25.0, 5.5, 4.0, 10000.0, 0.12, 0.0015)

	err := ts.AssembleGlobalStiffness()
	if err != nil {
		t.Fatalf("Failed to assemble global stiffness: %v", err)
	}

	n := 3 * len(ts.Nodes)
	if len(ts.K) != n {
		t.Errorf("Expected K size %d, got %d", n, len(ts.K))
	}
}

func TestTrussStaticSolve(t *testing.T) {
	ts := fea.GenerateArchBridge(25.0, 5.5, 4.0, 10000.0, 0.12, 0.0015)

	midNode := len(ts.Nodes) / 2
	ts.Nodes[midNode].FY = -50.0

	err := ts.Solve()
	if err != nil {
		t.Fatalf("Failed to solve: %v", err)
	}

	forces := ts.CalculateMemberForces()
	if len(forces) != len(ts.Members) {
		t.Errorf("Expected %d force results, got %d", len(ts.Members), len(forces))
	}

	maxStress := 0.0
	for _, f := range forces {
		if math.Abs(f.AxialStress) > maxStress {
			maxStress = math.Abs(f.AxialStress)
		}
	}
	if maxStress == 0 {
		t.Error("Expected non-zero stresses")
	}

	t.Logf("Max axial stress: %.3f MPa", maxStress)
}

func TestTrussSemiRigidJointCorrection(t *testing.T) {
	ts := fea.GenerateArchBridge(25.0, 5.5, 4.0, 10000.0, 0.12, 0.0015)

	for i := range ts.Nodes {
		ts.SetNodeJointType(i, fea.JointMortiseTenon)
	}
	ts.EnableSemiRigidJoints(true)

	midNode := len(ts.Nodes) / 2
	ts.Nodes[midNode].FY = -50.0

	err := ts.Solve()
	if err != nil {
		t.Fatalf("Failed to solve with semi-rigid joints: %v", err)
	}

	forces := ts.CalculateMemberForces()
	maxStressRatio := 0.0
	for _, f := range forces {
		if f.StressRatio > maxStressRatio {
			maxStressRatio = f.StressRatio
		}
	}
	t.Logf("Max stress ratio with semi-rigid joints: %.3f", maxStressRatio)
}

func TestRandomForestTraining(t *testing.T) {
	speciesData, err := craft.BuildSpeciesTrainingData()
	if err != nil {
		t.Fatalf("Failed to build species training data: %v", err)
	}

	rf := craft.NewRandomForest(50, 3, 5, 0.7, craft.ProblemClassification)
	rf.Train(speciesData.Features, speciesData.Labels)

	if len(rf.Trees) != 50 {
		t.Errorf("Expected 50 trees, got %d", len(rf.Trees))
	}
	if rf.OOBError < 0 || rf.OOBError > 1 {
		t.Errorf("OOB error out of range: %f", rf.OOBError)
	}
	t.Logf("Species RF OOB error: %.4f", rf.OOBError)
}

func TestRandomForestPrediction(t *testing.T) {
	speciesData, _ := craft.BuildSpeciesTrainingData()

	rf := craft.NewRandomForest(50, 3, 5, 0.7, craft.ProblemClassification)
	rf.Train(speciesData.Features, speciesData.Labels)

	testSample := make([]float64, len(speciesData.Features[0]))
	copy(testSample, speciesData.Features[0])

	pred, conf, probs := rf.Predict(testSample)

	if pred == "" {
		t.Error("Prediction should not be empty")
	}
	if conf <= 0 || conf > 1 {
		t.Errorf("Confidence out of range: %f", conf)
	}
	if len(probs) == 0 {
		t.Error("Probabilities should not be empty")
	}
	t.Logf("Predicted: %s (confidence: %.2f%%)", pred, conf*100)
}

func TestAnalyzeCraftEndToEnd(t *testing.T) {
	woodFeatures := craft.GenerateTypicalWoodFeatures("杉木")
	joineryFeatures := &craft.JoineryFeature{
		JointType:           "mortise_tenon",
		CraftsmanshipRating: 0.8,
	}

	result := craft.AnalyzeCraft(woodFeatures, joineryFeatures, "贯木拱")

	if result.WoodSpecies == "" {
		t.Error("Wood species prediction should not be empty")
	}
	if result.WoodGrade == "" {
		t.Error("Wood grade prediction should not be empty")
	}
	if result.JoineryType == "" {
		t.Error("Joinery type should not be empty")
	}
	if result.ConfidenceScore <= 0 {
		t.Error("Confidence should be positive")
	}
	if len(result.ConstructionSequence) == 0 {
		t.Error("Construction sequence should not be empty")
	}
	if len(result.FeatureImportance) == 0 {
		t.Error("Feature importance should not be empty")
	}

	t.Logf("Species: %s, Grade: %s, Confidence: %.2f%%",
		result.WoodSpecies, result.WoodGrade, result.ConfidenceScore*100)
	t.Logf("Method: %s", result.MethodUsed)
	t.Logf("Sequence steps: %d", len(result.ConstructionSequence))
}

func TestFeatureImportance(t *testing.T) {
	gradeData, _ := craft.BuildGradeTrainingData()

	rf := craft.NewRandomForest(50, 3, 5, 0.7, craft.ProblemClassification)
	rf.Train(gradeData.Features, gradeData.Labels)

	if len(rf.Importance) == 0 {
		t.Error("Importance map should not be empty")
	}

	sum := 0.0
	for _, v := range rf.Importance {
		sum += v
	}
	if math.Abs(sum-1.0) > 0.05 {
		t.Errorf("Importance sum should be ~1.0, got %.4f", sum)
	}

	t.Logf("Feature importance sum: %.4f", sum)
	for k, v := range rf.Importance {
		t.Logf("  %s: %.4f", k, v)
	}
}

func TestMovingLoadSimulation(t *testing.T) {
	ts := fea.GenerateArchBridge(25.0, 5.5, 4.0, 10000.0, 0.12, 0.0015)

	steps := 10
	results := make([]fea.MovingLoadResult, 0, steps)
	maxRatio := 0.0

	for i := 0; i < steps; i++ {
		pos := float64(i) / float64(steps-1)
		posIdx := int(pos * float64(len(ts.Nodes)-1))
		if posIdx >= len(ts.Nodes) {
			posIdx = len(ts.Nodes) - 1
		}

		ts.ClearLoads()
		ts.Nodes[posIdx].FY = -80.0

		err := ts.Solve()
		if err != nil {
			t.Fatalf("Solve failed at step %d: %v", i, err)
		}

		forces := ts.CalculateMemberForces()
		for _, f := range forces {
			if f.StressRatio > maxRatio {
				maxRatio = f.StressRatio
			}
		}

		results = append(results, fea.MovingLoadResult{
			Position:     pos,
			MemberForces: forces,
		})
	}

	if len(results) != steps {
		t.Errorf("Expected %d steps, got %d", steps, len(results))
	}
	if maxRatio == 0 {
		t.Error("Expected non-zero max stress ratio")
	}

	t.Logf("Moving load steps: %d, max stress ratio: %.3f", len(results), maxRatio)
}

func TestConcurrentBusUsage(t *testing.T) {
	bus := messaging.NewMessageBus()
	defer bus.Close()

	var wg sync.WaitGroup
	total := 100

	wg.Add(total)
	for i := 0; i < total; i++ {
		go func(idx int) {
			defer wg.Done()
			msg := messaging.NewMessage(messaging.MsgTypeSensorData, &messaging.SensorDataPayload{
				BridgeID:   idx,
				SensorCode: "S-TEST",
				Value:      float64(idx),
			})
			select {
			case bus.SensorDataChan <- msg:
			case <-time.After(100 * time.Millisecond):
			}
		}(i)
	}

	received := 0
	go func() {
		for {
			select {
			case <-bus.SensorDataChan:
				received++
			case <-time.After(500 * time.Millisecond):
				return
			}
		}
	}()

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	t.Logf("Sent %d messages, received %d", total, received)
}
