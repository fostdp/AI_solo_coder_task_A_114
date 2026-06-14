package messaging

import (
	"sync"
	"time"

	"ancient-bridge-system/internal/fea"
)

type MessageType string

const (
	MsgTypeSensorData       MessageType = "sensor_data"
	MsgTypeStaticLoadReq    MessageType = "static_load_req"
	MsgTypeMovingLoadReq    MessageType = "moving_load_req"
	MsgTypeStaticLoadResp   MessageType = "static_load_resp"
	MsgTypeMovingLoadResp   MessageType = "moving_load_resp"
	MsgTypeCraftAnalyzeReq  MessageType = "craft_analyze_req"
	MsgTypeCraftAnalyzeResp MessageType = "craft_analyze_resp"
	MsgTypeAlert            MessageType = "alert"
	MsgTypeStructureReq     MessageType = "structure_req"
	MsgTypeStructureResp    MessageType = "structure_resp"
)

type Message struct {
	ID        string
	Type      MessageType
	Timestamp time.Time
	Payload   interface{}
	ReplyTo   chan *Message
}

type MessageBus struct {
	SensorDataChan       chan *Message
	StaticLoadReqChan    chan *Message
	MovingLoadReqChan    chan *Message
	CraftAnalyzeReqChan  chan *Message
	AlertChan            chan *Message
	StructureReqChan     chan *Message

	StaticLoadRespChan   chan *Message
	MovingLoadRespChan   chan *Message
	CraftAnalyzeRespChan chan *Message
	StructureRespChan    chan *Message

	closeOnce sync.Once
	done      chan struct{}
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		SensorDataChan:       make(chan *Message, 1000),
		StaticLoadReqChan:    make(chan *Message, 100),
		MovingLoadReqChan:    make(chan *Message, 100),
		CraftAnalyzeReqChan:  make(chan *Message, 100),
		AlertChan:            make(chan *Message, 500),
		StructureReqChan:     make(chan *Message, 100),
		StaticLoadRespChan:   make(chan *Message, 100),
		MovingLoadRespChan:   make(chan *Message, 100),
		CraftAnalyzeRespChan: make(chan *Message, 100),
		StructureRespChan:    make(chan *Message, 100),
		done:                 make(chan struct{}),
	}
}

func (b *MessageBus) Close() {
	b.closeOnce.Do(func() {
		close(b.done)
	})
}

type SensorDataPayload struct {
	BridgeID    int
	SensorCode  string
	Value       float64
	Timestamp   time.Time
	QualityFlag int
	Unit        string
}

type StaticLoadRequest struct {
	BridgeID     int
	LoadValue    float64
	LoadPosition float64
	AnalysisName string
	LoadCase     string
	BridgeInfo   struct {
		SpanLength float64
		ArchRise   float64
		DeckWidth  float64
	}
}

type StaticLoadResponse struct {
	AnalysisID       int
	BridgeID         int
	MemberForces     []fea.MemberForces
	Displacements    []fea.NodeDisplacement
	MaxStressRatio   float64
	MaxDisplacement  float64
	ComparisonResult interface{}
	Error            string
}

type MovingLoadRequest struct {
	BridgeID    int
	TotalWeight float64
	Steps       int
	BridgeInfo  struct {
		SpanLength float64
		ArchRise   float64
		DeckWidth  float64
	}
}

type MovingLoadResponse struct {
	AnalysisID      int
	BridgeID        int
	Results         []fea.MovingLoadResult
	MaxStressRatio  float64
	MaxDisplacement float64
	Error           string
}

type CraftAnalyzeRequest struct {
	BridgeID       int
	WoodSpecies    string
	WoodFeatures   interface{}
	JoineryFeature interface{}
	BridgeType     string
}

type CraftAnalyzeResponse struct {
	AnalysisID int
	Result     interface{}
	Error      string
}

type AlertPayload struct {
	AlertID        int
	BridgeID       int
	MemberID       *int
	MemberCode     string
	AlertType      string
	AlertLevel     string
	AlertMessage   string
	MeasuredValue  float64
	ThresholdValue float64
	Unit           string
	Timestamp      time.Time
}

type StructureRequest struct {
	BridgeID   int
	BridgeInfo struct {
		SpanLength float64
		ArchRise   float64
		DeckWidth  float64
	}
}

type StructureResponse struct {
	BridgeID  int
	Nodes     []map[string]interface{}
	Members   []map[string]interface{}
	MemberTypes map[int]string
	Error     string
}

func NewMessage(msgType MessageType, payload interface{}) *Message {
	return &Message{
		ID:        generateID(),
		Type:      msgType,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}

func generateID() string {
	return time.Now().Format("20060102150405.000000")
}
