package craft

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type WoodFeature struct {
	GrainDensity    float64 `json:"grain_density"`
	GrainAngle      float64 `json:"grain_angle"`
	LatewoodRatio   float64 `json:"latewood_ratio"`
	KnotsCount      float64 `json:"knots_count"`
	AverageKnotSize float64 `json:"average_knot_size"`
	Density         float64 `json:"density"`
	Hardness        float64 `json:"hardness"`
	ColorR          int     `json:"color_r"`
	ColorG          int     `json:"color_g"`
	ColorB          int     `json:"color_b"`
}

type JoineryFeature struct {
	JointType           string  `json:"joint_type"`
	TenonLength         float64 `json:"tenon_length"`
	TenonWidth          float64 `json:"tenon_width"`
	TenonThickness      float64 `json:"tenon_thickness"`
	MortiseDepth        float64 `json:"mortise_depth"`
	ShoulderAngle       float64 `json:"shoulder_angle"`
	FitTolerance        float64 `json:"fit_tolerance"`
	WoodSpecies         string  `json:"wood_species"`
	CraftsmanshipRating float64 `json:"craftsmanship_rating"`
}

type CraftAnalysisResult struct {
	WoodSpecies          string              `json:"wood_species"`
	WoodGrade            string              `json:"wood_grade"`
	ConstructionSequence []string            `json:"construction_sequence"`
	JoineryType          string              `json:"joinery_type"`
	ConfidenceScore      float64             `json:"confidence_score"`
	FeatureImportance    map[string]float64  `json:"feature_importance"`
	MethodUsed           string              `json:"method_used"`
}

const (
	defaultMaxDepth     = 4
	defaultNumTrees     = 100
	defaultMinSamples   = 5
	defaultFeatureRatio = 0.7
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
	Depth        int
	ClassCounts  map[string]int
	OOBError     float64
}

type RandomForest struct {
	Trees         []*DecisionTreeNode
	NumTrees      int
	MaxDepth      int
	MinSamples    int
	FeatureRatio  float64
	FeatureNames  []string
	Importance    map[string]float64
	Labels        []string
	trainMutex    sync.Mutex
}

type DataSample struct {
	Features []float64
	Label    string
	Weight   float64
}

func NewRandomForest(numTrees, maxDepth int, featureNames []string) *RandomForest {
	if numTrees <= 0 {
		numTrees = defaultNumTrees
	}
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}

	rand.Seed(time.Now().UnixNano())

	return &RandomForest{
		Trees:         make([]*DecisionTreeNode, 0, numTrees),
		NumTrees:      numTrees,
		MaxDepth:      maxDepth,
		MinSamples:    defaultMinSamples,
		FeatureRatio:  defaultFeatureRatio,
		FeatureNames:  featureNames,
		Importance:    make(map[string]float64),
	}
}

func (rf *RandomForest) Train(samples []DataSample) {
	rf.Labels = extractUniqueLabels(samples)

	var wg sync.WaitGroup
	wg.Add(rf.NumTrees)

	treeChan := make(chan *DecisionTreeNode, rf.NumTrees)

	for t := 0; t < rf.NumTrees; t++ {
		go func(treeIdx int) {
			defer wg.Done()

			bootstrap := bootstrapSample(samples, len(samples))
			oobIdx := getOOBIndices(samples, bootstrap)

			numFeatures := int(math.Max(1, float64(len(bootstrap[0].Features))*rf.FeatureRatio))

			tree := rf.buildTree(bootstrap, 0, numFeatures)
			tree.OOBError = rf.calculateOOBError(tree, samples, oobIdx)

			treeChan <- tree
		}(t)
	}

	wg.Wait()
	close(treeChan)

	for tree := range treeChan {
		rf.Trees = append(rf.Trees, tree)
	}

	rf.calculateFeatureImportance(samples)
}

func (rf *RandomForest) buildTree(samples []DataSample, depth int, numFeatures int) *DecisionTreeNode {
	node := &DecisionTreeNode{
		Depth:       depth,
		Samples:     len(samples),
		ClassCounts: countClassLabels(samples),
	}

	if len(samples) == 0 {
		return nil
	}

	if depth >= rf.MaxDepth || len(samples) < rf.MinSamples || isPure(samples) {
		node.IsLeaf = true
		node.Prediction, node.Confidence = majorityVote(samples)
		return node
	}

	bestFeat, bestThresh, bestGain := rf.findBestSplit(samples, numFeatures)

	if bestGain < 1e-6 {
		node.IsLeaf = true
		node.Prediction, node.Confidence = majorityVote(samples)
		return node
	}

	leftSamples, rightSamples := splitDataset(samples, bestFeat, bestThresh)

	node.FeatureIndex = bestFeat
	node.Threshold = bestThresh
	node.Left = rf.buildTree(leftSamples, depth+1, numFeatures)
	node.Right = rf.buildTree(rightSamples, depth+1, numFeatures)

	return node
}

func (rf *RandomForest) findBestSplit(samples []DataSample, numFeatures int) (int, float64, float64) {
	numAllFeatures := len(samples[0].Features)
	featureIndices := randomFeatureIndices(numAllFeatures, numFeatures)

	bestGain := -1.0
	bestFeat := -1
	bestThresh := 0.0

	parentGini := weightedGini(samples)

	for _, fi := range featureIndices {
		thresholds := collectFeatureValues(samples, fi)
		sort.Float64s(thresholds)
		candidates := candidateThresholds(thresholds)

		for _, thresh := range candidates {
			left, right := splitDataset(samples, fi, thresh)

			if len(left) == 0 || len(right) == 0 {
				continue
			}

			childGini := weightedSplitGini(left, right)
			gain := parentGini - childGini

			if gain > bestGain {
				bestGain = gain
				bestFeat = fi
				bestThresh = thresh
			}
		}
	}

	return bestFeat, bestThresh, bestGain
}

func (rf *RandomForest) Predict(features []float64) (string, float64, map[string]float64) {
	votes := make(map[string]float64)
	totalWeight := 0.0

	for _, tree := range rf.Trees {
		pred, conf := predictTree(tree, features)
		weight := 1.0 / (1.0 + tree.OOBError)
		votes[pred] += conf * weight
		totalWeight += weight
	}

	for label := range votes {
		votes[label] /= totalWeight
	}

	bestLabel := ""
	bestProb := 0.0
	for label, prob := range votes {
		if prob > bestProb {
			bestProb = prob
			bestLabel = label
		}
	}

	return bestLabel, bestProb, votes
}

func predictTree(node *DecisionTreeNode, features []float64) (string, float64) {
	if node == nil {
		return "", 0.0
	}

	if node.IsLeaf {
		return node.Prediction, node.Confidence
	}

	if features[node.FeatureIndex] <= node.Threshold {
		return predictTree(node.Left, features)
	}
	return predictTree(node.Right, features)
}

func (rf *RandomForest) calculateOOBError(tree *DecisionTreeNode, samples []DataSample, oobIdx []int) float64 {
	if len(oobIdx) == 0 {
		return 0.5
	}

	correct := 0
	for _, idx := range oobIdx {
		s := samples[idx]
		pred, _ := predictTree(tree, s.Features)
		if pred == s.Label {
			correct++
		}
	}

	return 1.0 - float64(correct)/float64(len(oobIdx))
}

func (rf *RandomForest) calculateFeatureImportance(samples []DataSample) {
	importance := make([]float64, len(rf.FeatureNames))

	for _, tree := range rf.Trees {
		importance += treePermutationImportance(tree, samples, importance)
	}

	for i, fi := range importance {
		importance[i] = fi / float64(rf.NumTrees)
		if i < len(rf.FeatureNames) {
			rf.Importance[rf.FeatureNames[i]] = importance[i]
		}
	}

	normalizeMap(rf.Importance)
}

func treePermutationImportance(tree *DecisionTreeNode, samples []DataSample, baseImp []float64) []float64 {
	imp := make([]float64, len(baseImp))
	if len(samples) == 0 {
		return imp
	}

	baseCorrect := 0
	for _, s := range samples {
		pred, _ := predictTree(tree, s.Features)
		if pred == s.Label {
			baseCorrect++
		}
	}
	baseAcc := float64(baseCorrect) / float64(len(samples))

	for fi := 0; fi < len(imp); fi++ {
		permuted := make([]DataSample, len(samples))
		copy(permuted, samples)

		permuteFeature(permuted, fi)

		permCorrect := 0
		for _, s := range permuted {
			pred, _ := predictTree(tree, s.Features)
			if pred == s.Label {
				permCorrect++
			}
		}
		permAcc := float64(permCorrect) / float64(len(samples))
		imp[fi] = baseAcc - permAcc
	}

	return imp
}

func permuteFeature(samples []DataSample, featIdx int) {
	values := make([]float64, len(samples))
	for i, s := range samples {
		values[i] = s.Features[featIdx]
	}
	rand.Shuffle(len(values), func(i, j int) {
		values[i], values[j] = values[j], values[i]
	})
	for i := range samples {
		samples[i].Features[featIdx] = values[i]
	}
}

func extractUniqueLabels(samples []DataSample) []string {
	labelSet := make(map[string]bool)
	for _, s := range samples {
		labelSet[s.Label] = true
	}
	labels := make([]string, 0, len(labelSet))
	for l := range labelSet {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	return labels
}

func bootstrapSample(samples []DataSample, n int) []DataSample {
	result := make([]DataSample, n)
	for i := 0; i < n; i++ {
		idx := rand.Intn(len(samples))
		result[i] = samples[idx]
	}
	return result
}

func getOOBIndices(samples []DataSample, bootstrap []DataSample) []int {
	sampleMap := make(map[int]bool)
	for i := range bootstrap {
		idx := rand.Intn(len(samples))
		sampleMap[idx] = true
	}

	oob := make([]int, 0)
	for i := range samples {
		if !sampleMap[i] {
			oob = append(oob, i)
		}
	}
	return oob
}

func randomFeatureIndices(total, n int) []int {
	indices := make([]int, total)
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})
	if n > total {
		n = total
	}
	return indices[:n]
}

func isPure(samples []DataSample) bool {
	if len(samples) < 2 {
		return true
	}
	label := samples[0].Label
	for _, s := range samples[1:] {
		if s.Label != label {
			return false
		}
	}
	return true
}

func countClassLabels(samples []DataSample) map[string]int {
	counts := make(map[string]int)
	for _, s := range samples {
		counts[s.Label]++
	}
	return counts
}

func majorityVote(samples []DataSample) (string, float64) {
	counts := countClassLabels(samples)
	total := len(samples)
	if total == 0 {
		return "", 0.0
	}

	bestLabel := ""
	bestCount := 0
	for label, count := range counts {
		if count > bestCount {
			bestCount = count
			bestLabel = label
		}
	}
	return bestLabel, float64(bestCount) / float64(total)
}

func weightedGini(samples []DataSample) float64 {
	counts := make(map[string]float64)
	totalWeight := 0.0
	for _, s := range samples {
		w := s.Weight
		if w == 0 {
			w = 1.0
		}
		counts[s.Label] += w
		totalWeight += w
	}
	if totalWeight == 0 {
		return 0.0
	}

	gini := 1.0
	for _, w := range counts {
		p := w / totalWeight
		gini -= p * p
	}
	return gini
}

func weightedSplitGini(left, right []DataSample) float64 {
	nLeft := float64(len(left))
	nRight := float64(len(right))
	total := nLeft + nRight
	if total == 0 {
		return 0.0
	}
	return (nLeft/total)*weightedGini(left) + (nRight/total)*weightedGini(right)
}

func collectFeatureValues(samples []DataSample, featIdx int) []float64 {
	values := make([]float64, len(samples))
	for i, s := range samples {
		values[i] = s.Features[featIdx]
	}
	return values
}

func candidateThresholds(values []float64) []float64 {
	if len(values) < 2 {
		return values
	}
	thresholds := make([]float64, 0)
	seen := make(map[float64]bool)
	for i := 0; i < len(values)-1; i++ {
		if values[i] == values[i+1] {
			continue
		}
		mid := (values[i] + values[i+1]) / 2.0
		if !seen[mid] {
			thresholds = append(thresholds, mid)
			seen[mid] = true
		}
		if len(thresholds) > 20 {
			break
		}
	}
	if len(thresholds) == 0 {
		thresholds = append(thresholds, values[0])
	}
	return thresholds
}

func splitDataset(samples []DataSample, featIdx int, threshold float64) ([]DataSample, []DataSample) {
	left := make([]DataSample, 0)
	right := make([]DataSample, 0)
	for _, s := range samples {
		if s.Features[featIdx] <= threshold {
			left = append(left, s)
		} else {
			right = append(right, s)
		}
	}
	return left, right
}

func normalizeMap(m map[string]float64) {
	sum := 0.0
	for _, v := range m {
		sum += v
	}
	if sum == 0 {
		return
	}
	for k, v := range m {
		m[k] = v / sum
	}
}

var speciesFeatureNames = []string{
	"grain_density", "grain_angle", "latewood_ratio",
	"knots_count", "average_knot_size", "density",
	"hardness", "color_r", "color_g", "color_b",
}

var gradeFeatureNames = []string{
	"knots_count", "average_knot_size", "latewood_ratio",
	"density", "hardness", "grain_density",
	"grain_angle", "color_r", "color_g", "color_b",
}

func woodFeaturesToVector(wf *WoodFeature) []float64 {
	return []float64{
		wf.GrainDensity, wf.GrainAngle, wf.LatewoodRatio,
		wf.KnotsCount, wf.AverageKnotSize, wf.Density,
		wf.Hardness, float64(wf.ColorR), float64(wf.ColorG), float64(wf.ColorB),
	}
}

func addNoise(v float64, sigma float64) float64 {
	return v + (rand.Float64()-0.5)*2*sigma
}

func buildSpeciesTrainingData() []DataSample {
	samples := make([]DataSample, 0)
	speciesList := []struct {
		name   string
		base   *WoodFeature
		count  int
		sigma  float64
	}{
		{"杉木", GenerateTypicalWoodFeatures("杉木"), 80, 0.08},
		{"松木", GenerateTypicalWoodFeatures("松木"), 80, 0.10},
		{"楠木", {
			GrainDensity: 4.0, GrainAngle: 8.0, LatewoodRatio: 0.38,
			KnotsCount: 1.5, AverageKnotSize: 0.6, Density: 0.61,
			Hardness: 4.2, ColorR: 175, ColorG: 130, ColorB: 80,
		}, 60, 0.08},
		{"榆木", {
			GrainDensity: 4.5, GrainAngle: 12.0, LatewoodRatio: 0.45,
			KnotsCount: 3.0, AverageKnotSize: 1.0, Density: 0.68,
			Hardness: 5.2, ColorR: 190, ColorG: 145, ColorB: 95,
		}, 60, 0.10},
		{"黄花梨", GenerateTypicalWoodFeatures("黄花梨"), 40, 0.06},
		{"紫檀", GenerateTypicalWoodFeatures("紫檀"), 30, 0.05},
		{"柞木", {
			GrainDensity: 5.2, GrainAngle: 14.0, LatewoodRatio: 0.48,
			KnotsCount: 2.5, AverageKnotSize: 0.9, Density: 0.76,
			Hardness: 6.0, ColorR: 165, ColorG: 115, ColorB: 65,
		}, 40, 0.08},
	}

	for _, sp := range speciesList {
		for i := 0; i < sp.count; i++ {
			fv := woodFeaturesToVector(sp.base)
			for j := range fv {
				scale := 1.0
				if j >= 7 {
					scale = 15.0
				}
				fv[j] = addNoise(fv[j], fv[j]*sp.sigma+scale*0.01)
			}
			samples = append(samples, DataSample{
				Features: fv,
				Label:    sp.name,
				Weight:   1.0,
			})
		}
	}
	return samples
}

func buildGradeTrainingData() []DataSample {
	samples := make([]DataSample, 0)
	gradeDefs := []struct {
		grade  string
		knots  [2]float64
		density [2]float64
		latewood [2]float64
		hardness [2]float64
		count  int
	}{
		{"一等材", [2]float64{0, 2}, [2]float64{0.65, 1.1}, [2]float64{0.40, 0.65}, [2]float64{4.5, 9.0}, 80},
		{"二等材", [2]float64{1, 4}, [2]float64{0.50, 0.75}, [2]float64{0.35, 0.50}, [2]float64{3.0, 5.5}, 100},
		{"三等材", [2]float64{3, 7}, [2]float64{0.40, 0.58}, [2]float64{0.30, 0.43}, [2]float64{2.0, 4.0}, 100},
		{"等外材", [2]float64{5, 12}, [2]float64{0.30, 0.48}, [2]float64{0.20, 0.38}, [2]float64{1.0, 3.0}, 80},
	}

	randBase := GenerateTypicalWoodFeatures("默认")
	for _, gd := range gradeDefs {
		for i := 0; i < gd.count; i++ {
			wf := &WoodFeature{}
			*wf = *randBase
			wf.KnotsCount = gd.knots[0] + rand.Float64()*(gd.knots[1]-gd.knots[0])
			wf.AverageKnotSize = 0.3 + wf.KnotsCount*0.15 + rand.Float64()*0.3
			wf.Density = gd.density[0] + rand.Float64()*(gd.density[1]-gd.density[0])
			wf.LatewoodRatio = gd.latewood[0] + rand.Float64()*(gd.latewood[1]-gd.latewood[0])
			wf.Hardness = gd.hardness[0] + rand.Float64()*(gd.hardness[1]-gd.hardness[0])
			wf.GrainDensity = addNoise(wf.Density*7, 0.5)
			wf.GrainAngle = addNoise(10, 4)
			wf.ColorR = addNoise(170, 15)
			wf.ColorG = addNoise(140, 15)
			wf.ColorB = addNoise(90, 15)
			samples = append(samples, DataSample{
				Features: woodFeaturesToVector(wf),
				Label:    gd.grade,
				Weight:   1.0,
			})
		}
	}
	return samples
}

var (
	globalSpeciesRF *RandomForest
	globalGradeRF   *RandomForest
	initRFOnce      sync.Once
)

func initGlobalForests() {
	initRFOnce.Do(func() {
		speciesData := buildSpeciesTrainingData()
		globalSpeciesRF = NewRandomForest(80, 4, speciesFeatureNames)
		globalSpeciesRF.Train(speciesData)

		gradeData := buildGradeTrainingData()
		globalGradeRF = NewRandomForest(80, 4, gradeFeatureNames)
		globalGradeRF.Train(gradeData)

		logStr := "RandomForest initialized"
		_ = logStr
	})
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
	initGlobalForests()

	featureVec := woodFeaturesToVector(woodFeatures)

	species, speciesConf, _ := globalSpeciesRF.Predict(featureVec)
	grade, gradeConf, _ := globalGradeRF.Predict(featureVec)

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

	combinedImportance := make(map[string]float64)
	for k, v := range globalSpeciesRF.Importance {
		combinedImportance[k] += v * 0.5
	}
	for k, v := range globalGradeRF.Importance {
		combinedImportance[k] += v * 0.5
	}
	normalizeMap(combinedImportance)

	return &CraftAnalysisResult{
		WoodSpecies:          species,
		WoodGrade:            grade,
		ConstructionSequence: sequence,
		JoineryType:          joineryType,
		ConfidenceScore:      overallConfidence,
		FeatureImportance:    combinedImportance,
		MethodUsed:           "随机森林(Random Forest, max_depth=4) + 规则引擎",
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
