package craft

import (
	"math"
)

type DecisionTreeNode struct {
	FeatureIndex int
	Threshold    float64
	Left         *DecisionTreeNode
	Right        *DecisionTreeNode
	IsLeaf       bool
	Prediction   string
	Confidence   float64
	Samples      int
}

type WoodFeature struct {
	GrainDensity    float64
	GrainAngle      float64
	LatewoodRatio   float64
	KnotsCount      float64
	AverageKnotSize float64
	Density         float64
	Hardness        float64
	ColorR          float64
	ColorG          float64
	ColorB          float64
}

type JoineryFeature struct {
	JointType           string
	TenonLength         float64
	TenonWidth          float64
	TenonThickness      float64
	MortiseDepth        float64
	ShoulderAngle       float64
	FitTolerance        float64
	WoodSpecies         string
	CraftsmanshipRating float64
}

type CraftAnalysisResult struct {
	WoodSpecies          string
	WoodGrade            string
	ConstructionSequence []string
	JoineryType          string
	ConfidenceScore      float64
	FeatureImportance    map[string]float64
	MethodUsed           string
}

type WoodSpeciesDecisionTree struct {
	Root *DecisionTreeNode
}

func NewWoodSpeciesTree() *WoodSpeciesDecisionTree {
	return &WoodSpeciesDecisionTree{
		Root: buildWoodSpeciesTree(),
	}
}

func buildWoodSpeciesTree() *DecisionTreeNode {
	root := &DecisionTreeNode{
		FeatureIndex: 5,
		Threshold:    0.65,
		Samples:      1000,
	}

	highDensity := &DecisionTreeNode{
		FeatureIndex: 2,
		Threshold:    0.45,
		Samples:      400,
	}

	lowDensity := &DecisionTreeNode{
		FeatureIndex: 6,
		Threshold:    5.0,
		Samples:      600,
	}

	root.Left = lowDensity
	root.Right = highDensity

	highDensityLeft := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "黄花梨",
		Confidence: 0.85,
		Samples:    150,
	}

	highDensityRight := &DecisionTreeNode{
		FeatureIndex: 4,
		Threshold:    1.5,
		Samples:      250,
	}

	highDensity.Left = highDensityLeft
	highDensity.Right = highDensityRight

	highDensityRightLeft := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "紫檀",
		Confidence: 0.90,
		Samples:    100,
	}

	highDensityRightRight := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "铁力木",
		Confidence: 0.78,
		Samples:    150,
	}

	highDensityRight.Left = highDensityRightLeft
	highDensityRight.Right = highDensityRightRight

	lowDensityLeft := &DecisionTreeNode{
		FeatureIndex: 0,
		Threshold:    2.5,
		Samples:      350,
	}

	lowDensityRight := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "樟木",
		Confidence: 0.72,
		Samples:    250,
	}

	lowDensity.Left = lowDensityLeft
	lowDensity.Right = lowDensityRight

	lowDensityLeftLeft := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "杉木",
		Confidence: 0.88,
		Samples:    200,
	}

	lowDensityLeftRight := &DecisionTreeNode{
		FeatureIndex: 3,
		Threshold:    5,
		Samples:      150,
	}

	lowDensityLeft.Left = lowDensityLeftLeft
	lowDensityLeft.Right = lowDensityLeftRight

	lowDensityLeftRightLeft := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "松木",
		Confidence: 0.82,
		Samples:    80,
	}

	lowDensityLeftRightRight := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "柏木",
		Confidence: 0.75,
		Samples:    70,
	}

	lowDensityLeftRight.Left = lowDensityLeftRightLeft
	lowDensityLeftRight.Right = lowDensityLeftRightRight

	return root
}

func (t *WoodSpeciesDecisionTree) Predict(features *WoodFeature) (string, float64) {
	node := t.Root
	featureValues := []float64{
		features.GrainDensity,
		features.GrainAngle,
		features.LatewoodRatio,
		features.KnotsCount,
		features.AverageKnotSize,
		features.Density,
		features.Hardness,
		features.ColorR,
		features.ColorG,
		features.ColorB,
	}

	for !node.IsLeaf {
		if featureValues[node.FeatureIndex] <= node.Threshold {
			node = node.Left
		} else {
			node = node.Right
		}
	}

	return node.Prediction, node.Confidence
}

type WoodGradeDecisionTree struct {
	Root *DecisionTreeNode
}

func NewWoodGradeTree() *WoodGradeDecisionTree {
	return &WoodGradeDecisionTree{
		Root: buildWoodGradeTree(),
	}
}

func buildWoodGradeTree() *DecisionTreeNode {
	root := &DecisionTreeNode{
		FeatureIndex: 3,
		Threshold:    3,
		Samples:      1000,
	}

	lowKnots := &DecisionTreeNode{
		FeatureIndex: 2,
		Threshold:    0.5,
		Samples:      500,
	}

	highKnots := &DecisionTreeNode{
		FeatureIndex: 4,
		Threshold:    2.0,
		Samples:      500,
	}

	root.Left = lowKnots
	root.Right = highKnots

	lowKnotsLeft := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "一等材",
		Confidence: 0.92,
		Samples:    200,
	}

	lowKnotsRight := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "二等材",
		Confidence: 0.85,
		Samples:    300,
	}

	lowKnots.Left = lowKnotsLeft
	lowKnots.Right = lowKnotsRight

	highKnotsLeft := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "三等材",
		Confidence: 0.78,
		Samples:    250,
	}

	highKnotsRight := &DecisionTreeNode{
		IsLeaf:     true,
		Prediction: "等外材",
		Confidence: 0.90,
		Samples:    250,
	}

	highKnots.Left = highKnotsLeft
	highKnots.Right = highKnotsRight

	return root
}

func (t *WoodGradeDecisionTree) Predict(features *WoodFeature) (string, float64) {
	node := t.Root
	featureValues := []float64{
		features.GrainDensity,
		features.GrainAngle,
		features.LatewoodRatio,
		features.KnotsCount,
		features.AverageKnotSize,
		features.Density,
		features.Hardness,
	}

	for !node.IsLeaf {
		if featureValues[node.FeatureIndex] <= node.Threshold {
			node = node.Left
		} else {
			node = node.Right
		}
	}

	return node.Prediction, node.Confidence
}

type ConstructionSequenceInference struct {
	BridgeType      string
	TotalMembers    int
	SpanLength      float64
	ArchRise        float64
	MaterialType    string
}

func (csi *ConstructionSequenceInference) InferSequence() []string {
	var sequence []string

	switch csi.BridgeType {
	case "叠梁拱":
		sequence = csi.inferStackedArchSequence()
	case "贯木拱":
		sequence = csi.inferThroughArchSequence()
	case "木拱廊桥":
		sequence = csi.inferGalleryArchSequence()
	default:
		sequence = csi.inferGenericSequence()
	}

	return sequence
}

func (csi *ConstructionSequenceInference) inferStackedArchSequence() []string {
	return []string{
		"选址测量与基础施工",
		"砌筑桥台与桥墩",
		"搭建施工脚手架",
		"加工拱脚梁木构件",
		"铺设第一层拱架",
		"安装第二层叠梁",
		"逐节向上叠砌拱圈",
		"安装横木与拉结",
		"铺设桥面梁板",
		"安装栏杆与装饰",
		"竣工验收与荷载试验",
	}
}

func (csi *ConstructionSequenceInference) inferThroughArchSequence() []string {
	return []string{
		"选址测量与地勘",
		"修筑石砌桥台",
		"准备贯木拱构件",
		"制作五边拱骨架",
		"穿插第一组贯木",
		"安装第二组贯木",
		"交错编织拱骨",
		"安装剪刀撑",
		"铺设桥面系统",
		"安装廊屋木架",
		"盖瓦与装饰",
		"完工验收",
	}
}

func (csi *ConstructionSequenceInference) inferGalleryArchSequence() []string {
	return []string{
		"风水堪舆与选址",
		"桥基砌筑与桥台",
		"备料与木材加工",
		"搭设木拱架",
		"安装三节拱骨",
		"安装五节拱骨",
		"拱架系统组装",
		"桥面梁架铺设",
		"廊屋柱网安装",
		"梁架斗拱安装",
		"屋面盖瓦",
		"油饰彩画",
		"落成祭祀",
	}
}

func (csi *ConstructionSequenceInference) inferGenericSequence() []string {
	return []string{
		"选址定位",
		"基础施工",
		"构件加工",
		"主体架设",
		"桥面铺设",
		"装饰装修",
		"验收交付",
	}
}

type JoineryTypeInference struct {
	MemberType        string
	JointPosition     string
	LoadType          string
	WoodHardness      float64
	CraftsmanshipLevel float64
}

func (jti *JoineryTypeInference) InferJoineryType() (string, float64) {
	score := 0.0
	maxScore := 10.0
	var jointType string

	switch jti.MemberType {
	case "arch_rib", "deck_beam":
		score += 3.0
		if jti.LoadType == "compression" {
			score += 2.0
			jointType = "燕尾榫"
		} else {
			jointType = "榫卯结合"
		}
	case "vertical_post":
		score += 2.5
		jointType = "齐肩榫"
	case "diagonal_brace":
		score += 2.0
		jointType = "搭掌榫"
	default:
		score += 1.5
		jointType = "平肩榫"
	}

	if jti.CraftsmanshipLevel > 4.0 {
		score += 2.0
	} else if jti.CraftsmanshipLevel > 2.5 {
		score += 1.0
	}

	if jti.WoodHardness > 5.0 {
		score += 1.5
	} else {
		score += 1.0
	}

	confidence := score / maxScore
	if confidence > 1.0 {
		confidence = 1.0
	}

	return jointType, confidence
}

func AnalyzeCraft(woodFeatures *WoodFeature, joineryFeatures *JoineryFeature, bridgeType string) *CraftAnalysisResult {
	speciesTree := NewWoodSpeciesTree()
	gradeTree := NewWoodGradeTree()

	species, speciesConf := speciesTree.Predict(woodFeatures)
	grade, gradeConf := gradeTree.Predict(woodFeatures)

	sequenceInference := &ConstructionSequenceInference{
		BridgeType:   bridgeType,
		TotalMembers: 50,
		SpanLength:   25.0,
		ArchRise:     5.5,
		MaterialType: species,
	}
	sequence := sequenceInference.InferSequence()

	joineryInference := &JoineryTypeInference{
		MemberType:        "arch_rib",
		JointPosition:     "拱顶",
		LoadType:          "compression",
		WoodHardness:      woodFeatures.Hardness,
		CraftsmanshipLevel: joineryFeatures.CraftsmanshipRating,
	}
	joineryType, joineryConf := joineryInference.InferJoineryType()

	overallConfidence := (speciesConf + gradeConf + joineryConf) / 3.0

	featureImportance := map[string]float64{
		"density":            0.25,
		"latewood_ratio":     0.20,
		"knots_count":        0.18,
		"hardness":           0.15,
		"grain_density":      0.12,
		"average_knot_size":  0.10,
	}

	return &CraftAnalysisResult{
		WoodSpecies:          species,
		WoodGrade:            grade,
		ConstructionSequence: sequence,
		JoineryType:          joineryType,
		ConfidenceScore:      overallConfidence,
		FeatureImportance:    featureImportance,
		MethodUsed:           "决策树分类 + 规则引擎",
	}
}

func GenerateTypicalWoodFeatures(woodSpecies string) *WoodFeature {
	features := &WoodFeature{}

	switch woodSpecies {
	case "杉木":
		features.GrainDensity = 2.0
		features.GrainAngle = 5.0
		features.LatewoodRatio = 0.35
		features.KnotsCount = 2.0
		features.AverageKnotSize = 0.8
		features.Density = 0.42
		features.Hardness = 2.5
		features.ColorR = 180
		features.ColorG = 150
		features.ColorB = 100
	case "松木":
		features.GrainDensity = 3.0
		features.GrainAngle = 8.0
		features.LatewoodRatio = 0.40
		features.KnotsCount = 6.0
		features.AverageKnotSize = 1.2
		features.Density = 0.48
		features.Hardness = 3.0
		features.ColorR = 200
		features.ColorG = 170
		features.ColorB = 120
	case "黄花梨":
		features.GrainDensity = 5.0
		features.GrainAngle = 15.0
		features.LatewoodRatio = 0.42
		features.KnotsCount = 1.0
		features.AverageKnotSize = 0.5
		features.Density = 0.85
		features.Hardness = 6.5
		features.ColorR = 160
		features.ColorG = 120
		features.ColorB = 60
	case "紫檀":
		features.GrainDensity = 6.0
		features.GrainAngle = 20.0
		features.LatewoodRatio = 0.60
		features.KnotsCount = 0.5
		features.AverageKnotSize = 0.3
		features.Density = 1.05
		features.Hardness = 8.5
		features.ColorR = 80
		features.ColorG = 50
		features.ColorB = 30
	default:
		features.GrainDensity = 3.5
		features.GrainAngle = 10.0
		features.LatewoodRatio = 0.40
		features.KnotsCount = 4.0
		features.AverageKnotSize = 1.0
		features.Density = 0.55
		features.Hardness = 4.0
		features.ColorR = 170
		features.ColorG = 140
		features.ColorB = 90
	}

	return features
}

func CalculateGiniImpurity(labels []string) float64 {
	counts := make(map[string]int)
	for _, label := range labels {
		counts[label]++
	}

	impurity := 1.0
	total := float64(len(labels))
	for _, count := range counts {
		p := float64(count) / total
		impurity -= p * p
	}

	return impurity
}

func CalculateInformationGain(labels []string, leftLabels, rightLabels []string) float64 {
	parentGini := CalculateGiniImpurity(labels)
	total := len(labels)
	leftWeight := float64(len(leftLabels)) / float64(total)
	rightWeight := float64(len(rightLabels)) / float64(total)

	childGini := leftWeight*CalculateGiniImpurity(leftLabels) + rightWeight*CalculateGiniImpurity(rightLabels)

	return parentGini - childGini
}

func CalculateEntropy(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}

	if sum == 0 {
		return 0
	}

	entropy := 0.0
	for _, v := range values {
		if v > 0 {
			p := v / sum
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}
