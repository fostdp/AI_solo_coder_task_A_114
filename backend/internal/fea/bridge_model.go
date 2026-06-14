package fea

import (
	"math"
)

type BridgeModel struct {
	Structure   *TrussStructure
	BridgeName  string
	SpanLength  float64
	ArchRise    float64
	DeckWidth   float64
	MemberTypes map[int]string
}

func GenerateArchBridge(bridgeID int, spanLength, archRise, deckWidth float64) *BridgeModel {
	structure := NewTrussStructure()
	memberTypes := make(map[int]string)

	numArchSegments := 10
	numDeckNodes := numArchSegments + 1
	numVerticalMembers := numArchSegments + 1

	archRadius := (spanLength*spanLength/4 + archRise*archRise) / (2 * archRise)
	archCenterY := archRise - archRadius
	archCenterX := spanLength / 2

	nodeID := 1

	for i := 0; i <= numArchSegments; i++ {
		angle := -math.Pi/2 + math.Pi*float64(i)/float64(numArchSegments)
		x := archCenterX + archRadius*math.Cos(angle)
		y := archCenterY + archRadius*math.Sin(angle)
		structure.AddNode(nodeID, x, y)
		nodeID++
	}

	for i := 0; i <= numArchSegments; i++ {
		x := float64(i) * spanLength / float64(numArchSegments)
		y := 0.0
		structure.AddNode(nodeID, x, y)
		nodeID++
	}

	structure.SetNodeConstraint(1, true, true, true)
	structure.SetNodeConstraint(numArchSegments+1, true, true, true)

	memberID := 1
	e := 10000.0
	aArch := 0.12
	iArch := 0.0015
	aDeck := 0.08
	iDeck := 0.0008
	aVertical := 0.06
	iVertical := 0.0004

	for i := 1; i <= numArchSegments; i++ {
		startID := i
		endID := i + 1
		member := structure.AddMember(memberID, startID, endID, e, aArch, iArch)
		if member != nil {
			memberTypes[memberID] = "arch_rib"
		}
		memberID++
	}

	deckStartID := numArchSegments + 2
	for i := 0; i < numArchSegments; i++ {
		startID := deckStartID + i
		endID := deckStartID + i + 1
		member := structure.AddMember(memberID, startID, endID, e, aDeck, iDeck)
		if member != nil {
			memberTypes[memberID] = "deck_beam"
		}
		memberID++
	}

	for i := 0; i <= numArchSegments; i++ {
		archNodeID := i + 1
		deckNodeID := deckStartID + i
		member := structure.AddMember(memberID, archNodeID, deckNodeID, e, aVertical, iVertical)
		if member != nil {
			memberTypes[memberID] = "vertical_post"
		}
		memberID++
	}

	numDiagonals := numArchSegments
	for i := 0; i < numDiagonals; i++ {
		topID := deckStartID + i
		botID := i + 2
		member := structure.AddMember(memberID, topID, botID, e, aVertical*0.7, iVertical*0.5)
		if member != nil {
			memberTypes[memberID] = "diagonal_brace"
		}
		memberID++
	}

	structure.AssembleStiffness()

	return &BridgeModel{
		Structure:   structure,
		BridgeName:  "bridge_" + string(rune('0'+bridgeID)),
		SpanLength:  spanLength,
		ArchRise:    archRise,
		DeckWidth:   deckWidth,
		MemberTypes: memberTypes,
	}
}

func (bm *BridgeModel) AnalyzeStaticLoad(loadValue float64, loadPosition float64) ([]MemberForces, []NodeDisplacement) {
	structure := bm.Structure

	deckNodes := make([]*Node, 0)
	for _, node := range structure.Nodes {
		if math.Abs(node.Y) < 0.01 {
			deckNodes = append(deckNodes, node)
		}
	}

	if len(deckNodes) < 2 {
		return nil, nil
	}

	loads := make([]NodeLoad, 0)
	for _, node := range deckNodes {
		dist := math.Abs(node.X - loadPosition)
		if dist < 2.0 {
			factor := 1.0 - dist/2.0
			loads = append(loads, NodeLoad{
				NodeID: node.ID,
				FY:     -loadValue * factor,
			})
		}
	}

	structure.ApplyLoads(loads)
	structure.Solve()

	memberForces := structure.CalculateMemberForces()
	displacements := structure.calculateDisplacements()

	return memberForces, displacements
}

func (bm *BridgeModel) AnalyzeMovingLoad(totalWeight float64, steps int) []MovingLoadResult {
	return bm.Structure.AnalyzeMovingLoad(totalWeight, steps)
}

type YingzaoComparison struct {
	MemberID           int
	MemberType         string
	ActualStress       float64
	AllowableStress    float64
	StressRatio        float64
	MaxSpanRatio       float64
	ActualSpanRatio    float64
	SectionModulus     float64
	MinSectionModulus  float64
	Compliant          bool
	GradeUsed          string
	SpecGrade          string
}

func (bm *BridgeModel) CompareWithYingzaoFashi(memberForces []MemberForces, specs []YingzaoSpec) []YingzaoComparison {
	var comparisons []YingzaoComparison

	for _, mf := range memberForces {
		memberType := bm.MemberTypes[mf.MemberID]
		member := bm.Structure.MemberMap[mf.MemberID]
		if member == nil {
			continue
		}

		sectionModulus := member.I / (member.A / 6.0)
		actualSpanRatio := member.Length / math.Sqrt(member.A)

		var bestSpec *YingzaoSpec
		for i := range specs {
			if specs[i].ComponentType == "梁栿" && memberType == "deck_beam" {
				if bestSpec == nil || specs[i].AllowableStress > bestSpec.AllowableStress {
					s := specs[i]
					bestSpec = &s
				}
			}
			if specs[i].ComponentType == "拱枋" && memberType == "arch_rib" {
				if bestSpec == nil || specs[i].AllowableStress > bestSpec.AllowableStress {
					s := specs[i]
					bestSpec = &s
				}
			}
		}

		if bestSpec == nil {
			defaultSpec := YingzaoSpec{
				ComponentType:     "default",
				AllowableStress:   8.5,
				MaxSpanRatio:      12.0,
				MinSectionModulus: 300.0,
			}
			bestSpec = &defaultSpec
		}

		stressRatio := math.Abs(mf.CombinedStress) / bestSpec.AllowableStress
		compliant := stressRatio <= 1.0 && actualSpanRatio <= bestSpec.MaxSpanRatio

		comparisons = append(comparisons, YingzaoComparison{
			MemberID:          mf.MemberID,
			MemberType:        memberType,
			ActualStress:      mf.CombinedStress,
			AllowableStress:   bestSpec.AllowableStress,
			StressRatio:       stressRatio,
			MaxSpanRatio:      bestSpec.MaxSpanRatio,
			ActualSpanRatio:   actualSpanRatio,
			SectionModulus:    sectionModulus,
			MinSectionModulus: bestSpec.MinSectionModulus,
			Compliant:         compliant,
			SpecGrade:         bestSpec.GradeLevel,
		})
	}

	return comparisons
}

type YingzaoSpec struct {
	ComponentType     string
	GradeLevel        string
	MaterialGrade     string
	MaxSpanRatio      float64
	MinSectionModulus float64
	AllowableStress   float64
	SafetyFactor      float64
}
