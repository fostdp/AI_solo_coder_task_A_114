package fea

import (
	"math"
)

type JointType string

const (
	JointMortiseTenon  JointType = "mortise_tenon"
	JointDovetail      JointType = "dovetail"
	JointLap           JointType = "lap"
	JointRigid         JointType = "rigid"
	JointPin           JointType = "pin"
)

type SemiRigidJoint struct {
	Ktheta  float64
	Kt      float64
	Kaxial  float64
	Rtheta  float64
	Type    JointType
}

var defaultJointStiffness = map[JointType]SemiRigidJoint{
	JointMortiseTenon: {Ktheta: 0.65, Kt: 0.80, Kaxial: 0.90, Rtheta: 1.0, Type: JointMortiseTenon},
	JointDovetail:     {Ktheta: 0.75, Kt: 0.85, Kaxial: 0.70, Rtheta: 1.0, Type: JointDovetail},
	JointLap:          {Ktheta: 0.55, Kt: 0.60, Kaxial: 0.50, Rtheta: 1.0, Type: JointLap},
	JointRigid:        {Ktheta: 1.00, Kt: 1.00, Kaxial: 1.00, Rtheta: 1.0, Type: JointRigid},
	JointPin:          {Ktheta: 0.02, Kt: 0.95, Kaxial: 0.98, Rtheta: 1.0, Type: JointPin},
}

type Node struct {
	ID       int
	X, Y     float64
	UX, UY   float64
	RZ       float64
	FixedX   bool
	FixedY   bool
	FixedR   bool
	Joint    SemiRigidJoint
	Ktheta   float64
	KshearX  float64
	KshearY  float64
}

type Member struct {
	ID     int
	Start  *Node
	End    *Node
	E      float64
	A      float64
	I      float64
	Length float64
	Angle  float64
	JointStart JointType
	JointEnd   JointType
}

type TrussStructure struct {
	Nodes     []*Node
	Members   []*Member
	NodeMap   map[int]*Node
	MemberMap map[int]*Member
	K         [][]float64
	F         []float64
	U         []float64
	dofs      int
	jointStiffness map[JointType]SemiRigidJoint
	useSemiRigid   bool
}

func NewTrussStructure() *TrussStructure {
	return &TrussStructure{
		NodeMap:        make(map[int]*Node),
		MemberMap:      make(map[int]*Member),
		jointStiffness: defaultJointStiffness,
		useSemiRigid:   true,
	}
}

func (ts *TrussStructure) EnableSemiRigidJoints(enable bool) {
	ts.useSemiRigid = enable
}

func (ts *TrussStructure) SetCustomJointStiffness(jointType JointType, ktheta, kt, kaxial float64) {
	ts.jointStiffness[jointType] = SemiRigidJoint{
		Ktheta: ktheta,
		Kt:     kt,
		Kaxial: kaxial,
		Type:   jointType,
	}
}

func (ts *TrussStructure) AddNode(id int, x, y float64) *Node {
	node := &Node{
		ID:       id,
		X:        x,
		Y:        y,
		Joint:    ts.jointStiffness[JointMortiseTenon],
		Ktheta:   0,
		KshearX:  0,
		KshearY:  0,
	}
	ts.Nodes = append(ts.Nodes, node)
	ts.NodeMap[id] = node
	return node
}

func (ts *TrussStructure) SetNodeJointType(id int, jointType JointType) {
	if node, exists := ts.NodeMap[id]; exists {
		if js, ok := ts.jointStiffness[jointType]; ok {
			node.Joint = js
		}
	}
}

func (ts *TrussStructure) SetNodeCustomStiffness(id int, ktheta, kshearX, kshearY float64) {
	if node, exists := ts.NodeMap[id]; exists {
		node.Ktheta = ktheta
		node.KshearX = kshearX
		node.KshearY = kshearY
	}
}

func (ts *TrussStructure) SetNodeConstraint(id int, fx, fy, fr bool) {
	if node, exists := ts.NodeMap[id]; exists {
		node.FixedX = fx
		node.FixedY = fy
		node.FixedR = fr
	}
}

func (ts *TrussStructure) AddMember(id int, startID, endID int, e, a, i float64, jointTypes ...JointType) *Member {
	start := ts.NodeMap[startID]
	end := ts.NodeMap[endID]
	if start == nil || end == nil {
		return nil
	}

	dx := end.X - start.X
	dy := end.Y - start.Y
	length := math.Sqrt(dx*dx + dy*dy)
	angle := math.Atan2(dy, dx)

	jointStart := JointMortiseTenon
	jointEnd := JointMortiseTenon
	if len(jointTypes) >= 1 {
		jointStart = jointTypes[0]
	}
	if len(jointTypes) >= 2 {
		jointEnd = jointTypes[1]
	}

	member := &Member{
		ID:         id,
		Start:      start,
		End:        end,
		E:          e,
		A:          a,
		I:          i,
		Length:     length,
		Angle:      angle,
		JointStart: jointStart,
		JointEnd:   jointEnd,
	}

	ts.Members = append(ts.Members, member)
	ts.MemberMap[id] = member
	return member
}

func (m *Member) LocalStiffness() [6][6]float64 {
	ea := m.E * m.A
	ei := m.E * m.I
	l := m.Length
	l2 := l * l
	l3 := l * l * l

	return [6][6]float64{
		{ea / l, 0, 0, -ea / l, 0, 0},
		{0, 12 * ei / l3, 6 * ei / l2, 0, -12 * ei / l3, 6 * ei / l2},
		{0, 6 * ei / l2, 4 * ei / l, 0, -6 * ei / l2, 2 * ei / l},
		{-ea / l, 0, 0, ea / l, 0, 0},
		{0, -12 * ei / l3, -6 * ei / l2, 0, 12 * ei / l3, -6 * ei / l2},
		{0, 6 * ei / l2, 2 * ei / l, 0, -6 * ei / l2, 4 * ei / l},
	}
}

func (m *Member) LocalStiffnessSemiRigid() [6][6]float64 {
	k0 := m.LocalStiffness()
	if len(m.Start.Joint.Type) == 0 || len(m.End.Joint.Type) == 0 {
		return k0
	}

	js := m.Start.Joint
	je := m.End.Joint
	l := m.Length
	ei := m.E * m.I

	alphaS := ei / l
	alphaE := ei / l

	var k1, k2, k3, k4 float64
	if js.Ktheta > 0 && js.Ktheta < 1.0 {
		k1 = 4 * alphaS / (1 + 4*alphaS/js.Ktheta/ei)
	} else {
		k1 = k0[2][2]
	}

	if je.Ktheta > 0 && je.Ktheta < 1.0 {
		k4 = 4 * alphaE / (1 + 4*alphaE/je.Ktheta/ei)
	} else {
		k4 = k0[5][5]
	}

	shearFactor := (js.Kt + je.Kt) / 2.0
	axialFactor := (js.Kaxial + je.Kaxial) / 2.0

	var k [6][6]float64
	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			k[i][j] = k0[i][j]
		}
	}

	rowFactor := []float64{axialFactor, shearFactor, js.Ktheta, axialFactor, shearFactor, je.Ktheta}
	colFactor := []float64{axialFactor, shearFactor, js.Ktheta, axialFactor, shearFactor, je.Ktheta}
	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			avg := math.Sqrt(rowFactor[i] * colFactor[j])
			k[i][j] *= avg
		}
	}

	k[2][2] = k1
	k[5][5] = k4
	k[2][5] = k0[2][5] * math.Sqrt(js.Ktheta*je.Ktheta)
	k[5][2] = k[2][5]

	return k
}

func (m *Member) TransformationMatrix() [6][6]float64 {
	c := math.Cos(m.Angle)
	s := math.Sin(m.Angle)

	return [6][6]float64{
		{c, s, 0, 0, 0, 0},
		{-s, c, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0},
		{0, 0, 0, c, s, 0},
		{0, 0, 0, -s, c, 0},
		{0, 0, 0, 0, 0, 1},
	}
}

func (m *Member) GlobalStiffness() [6][6]float64 {
	var kLocal [6][6]float64
	if m.Start.Joint.Type != m.End.Joint.Type || m.Start.Joint.Type == "" {
		kLocal = m.LocalStiffnessSemiRigid()
	} else {
		kLocal = m.LocalStiffnessSemiRigid()
	}
	T := m.TransformationMatrix()
	var kGlobal [6][6]float64

	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			for k := 0; k < 6; k++ {
				for l := 0; l < 6; l++ {
					kGlobal[i][j] += T[k][i] * kLocal[k][l] * T[l][j]
				}
			}
		}
	}

	return kGlobal
}

func (ts *TrussStructure) AssembleStiffness() {
	ts.dofs = len(ts.Nodes) * 3

	ts.K = make([][]float64, ts.dofs)
	for i := range ts.K {
		ts.K[i] = make([]float64, ts.dofs)
	}

	for _, member := range ts.Members {
		kGlobal := member.GlobalStiffness()

		startIdx := (member.Start.ID - 1) * 3
		endIdx := (member.End.ID - 1) * 3

		indices := []int{
			startIdx, startIdx + 1, startIdx + 2,
			endIdx, endIdx + 1, endIdx + 2,
		}

		for i := 0; i < 6; i++ {
			for j := 0; j < 6; j++ {
				ts.K[indices[i]][indices[j]] += kGlobal[i][j]
			}
		}
	}

	if ts.useSemiRigid {
		ts.assembleNodeSprings()
	}
}

func (ts *TrussStructure) assembleNodeSprings() {
	for _, node := range ts.Nodes {
		if node.FixedR && node.FixedX && node.FixedY {
			continue
		}
		nodeIdx := (node.ID - 1) * 3

		connectedMembers := 0
		totalEIL := 0.0
		for _, member := range ts.Members {
			if member.Start.ID == node.ID || member.End.ID == node.ID {
				connectedMembers++
				if member.I > 0 {
					totalEIL += member.E * member.I / member.Length
				}
			}
		}
		if connectedMembers == 0 {
			continue
		}

		if !node.FixedX && node.KshearX > 0 {
			ts.K[nodeIdx][nodeIdx] += node.KshearX
		}
		if !node.FixedY && node.KshearY > 0 {
			ts.K[nodeIdx+1][nodeIdx+1] += node.KshearY
		}

		ktheta := node.Ktheta
		if ktheta == 0 && node.Joint.Ktheta > 0 {
			ktheta = node.Joint.Ktheta * totalEIL * 0.1
			if ktheta < 0 {
				ktheta = 0
			}
		}

		if !node.FixedR && ktheta > 0 {
			ts.K[nodeIdx+2][nodeIdx+2] += ktheta
		}
	}
}

func (ts *TrussStructure) ApplyLoads(loads []NodeLoad) {
	ts.F = make([]float64, ts.dofs)
	ts.U = make([]float64, ts.dofs)

	for _, load := range loads {
		nodeIdx := (load.NodeID - 1) * 3
		if nodeIdx < ts.dofs {
			ts.F[nodeIdx] = load.FX
			ts.F[nodeIdx+1] = load.FY
			ts.F[nodeIdx+2] = load.MZ
		}
	}
}

type NodeLoad struct {
	NodeID int
	FX     float64
	FY     float64
	MZ     float64
}

func (ts *TrussStructure) Solve() error {
	freeDofs := ts.getFreeDofs()
	nFree := len(freeDofs)

	Kff := make([][]float64, nFree)
	for i := range Kff {
		Kff[i] = make([]float64, nFree)
	}

	Ff := make([]float64, nFree)

	for i, fi := range freeDofs {
		Ff[i] = ts.F[fi]
		for j, fj := range freeDofs {
			Kff[i][j] = ts.K[fi][fj]
		}
	}

	Uf, err := gaussElimination(Kff, Ff)
	if err != nil {
		return err
	}

	for i, fi := range freeDofs {
		ts.U[fi] = Uf[i]
	}

	for i, node := range ts.Nodes {
		idx := i * 3
		node.UX = ts.U[idx]
		node.UY = ts.U[idx+1]
		node.RZ = ts.U[idx+2]
	}

	return nil
}

func (ts *TrussStructure) getFreeDofs() []int {
	var free []int
	for i, node := range ts.Nodes {
		idx := i * 3
		if !node.FixedX {
			free = append(free, idx)
		}
		if !node.FixedY {
			free = append(free, idx+1)
		}
		if !node.FixedR {
			free = append(free, idx+2)
		}
	}
	return free
}

func gaussElimination(A [][]float64, B []float64) ([]float64, error) {
	n := len(B)
	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, n+1)
		copy(aug[i][:n], A[i])
		aug[i][n] = B[i]
	}

	for col := 0; col < n; col++ {
		pivotRow := col
		for row := col + 1; row < n; row++ {
			if math.Abs(aug[row][col]) > math.Abs(aug[pivotRow][col]) {
				pivotRow = row
			}
		}
		aug[col], aug[pivotRow] = aug[pivotRow], aug[col]

		if math.Abs(aug[col][col]) < 1e-10 {
			return nil, nil
		}

		for row := col + 1; row < n; row++ {
			factor := aug[row][col] / aug[col][col]
			for j := col; j <= n; j++ {
				aug[row][j] -= factor * aug[col][j]
			}
		}
	}

	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		sum := 0.0
		for j := i + 1; j < n; j++ {
			sum += aug[i][j] * x[j]
		}
		x[i] = (aug[i][n] - sum) / aug[i][i]
	}

	return x, nil
}

type MemberForces struct {
	MemberID      int
	AxialForce    float64
	ShearForce    float64
	BendingMoment float64
	AxialStress   float64
	BendingStress float64
}

func (ts *TrussStructure) CalculateMemberForces() []MemberForces {
	var forces []MemberForces

	for _, member := range ts.Members {
		uLocal := ts.getMemberLocalDisplacements(member)
		fLocal := ts.calculateLocalForces(member, uLocal)

		axialForce := fLocal[0]
		shearForce := fLocal[1]
		bendingMoment := fLocal[2]

		axialStress := axialForce / member.A
		bendingStress := 0.0
		if member.I > 0 {
			bendingStress = bendingMoment * (member.Length / 12) / member.I
		}

		forces = append(forces, MemberForces{
			MemberID:      member.ID,
			AxialForce:    axialForce,
			ShearForce:    shearForce,
			BendingMoment: bendingMoment,
			AxialStress:   axialStress,
			BendingStress: bendingStress,
		})
	}

	return forces
}

func (ts *TrussStructure) getMemberLocalDisplacements(m *Member) [6]float64 {
	startIdx := (m.Start.ID - 1) * 3
	endIdx := (m.End.ID - 1) * 3

	uGlobal := [6]float64{
		ts.U[startIdx], ts.U[startIdx+1], ts.U[startIdx+2],
		ts.U[endIdx], ts.U[endIdx+1], ts.U[endIdx+2],
	}

	T := m.TransformationMatrix()
	var uLocal [6]float64
	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			uLocal[i] += T[i][j] * uGlobal[j]
		}
	}

	return uLocal
}

func (ts *TrussStructure) calculateLocalForces(m *Member, uLocal [6]float64) [6]float64 {
	kLocal := m.LocalStiffness()
	var fLocal [6]float64

	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			fLocal[i] += kLocal[i][j] * uLocal[j]
		}
	}

	return fLocal
}

type MovingLoadResult struct {
	Position     float64
	MaxAxial     float64
	MaxMoment    float64
	MemberForces []MemberForces
	Displacements []NodeDisplacement
}

type NodeDisplacement struct {
	NodeID    int
	UX        float64
	UY        float64
	TotalDisp float64
}

func (ts *TrussStructure) AnalyzeMovingLoad(loadValue float64, steps int) []MovingLoadResult {
	var results []MovingLoadResult

	totalLength := 0.0
	if len(ts.Nodes) > 0 {
		minX := ts.Nodes[0].X
		maxX := ts.Nodes[0].X
		for _, node := range ts.Nodes {
			if node.X < minX {
				minX = node.X
			}
			if node.X > maxX {
				maxX = node.X
			}
		}
		totalLength = maxX - minX
	}

	for i := 0; i <= steps; i++ {
		position := totalLength * float64(i) / float64(steps)

		loads := ts.calculateLoadDistribution(loadValue, position)
		ts.ApplyLoads(loads)
		ts.Solve()

		memberForces := ts.CalculateMemberForces()
		displacements := ts.calculateDisplacements()

		maxAxial := 0.0
		maxMoment := 0.0
		for _, mf := range memberForces {
			if math.Abs(mf.AxialForce) > maxAxial {
				maxAxial = math.Abs(mf.AxialForce)
			}
			if math.Abs(mf.BendingMoment) > maxMoment {
				maxMoment = math.Abs(mf.BendingMoment)
			}
		}

		results = append(results, MovingLoadResult{
			Position:      position,
			MaxAxial:      maxAxial,
			MaxMoment:     maxMoment,
			MemberForces:  memberForces,
			Displacements: displacements,
		})
	}

	return results
}

func (ts *TrussStructure) calculateLoadDistribution(loadValue, position float64) []NodeLoad {
	var loads []NodeLoad

	minX := ts.Nodes[0].X
	maxX := ts.Nodes[0].X
	for _, node := range ts.Nodes {
		if node.X < minX {
			minX = node.X
		}
		if node.X > maxX {
			maxX = node.X
		}
	}

	loadX := minX + position

	for _, node := range ts.Nodes {
		if node.Y > 0 {
			continue
		}
		dist := math.Abs(node.X - loadX)
		if dist < 3.0 {
			factor := 1.0 - dist/3.0
			loads = append(loads, NodeLoad{
				NodeID: node.ID,
				FY:     -loadValue * factor,
			})
		}
	}

	return loads
}

func (ts *TrussStructure) calculateDisplacements() []NodeDisplacement {
	var displacements []NodeDisplacement

	for _, node := range ts.Nodes {
		totalDisp := math.Sqrt(node.UX*node.UX + node.UY*node.UY)
		displacements = append(displacements, NodeDisplacement{
			NodeID:    node.ID,
			UX:        node.UX,
			UY:        node.UY,
			TotalDisp: totalDisp,
		})
	}

	return displacements
}
