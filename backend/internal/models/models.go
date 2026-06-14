package models

import (
	"time"
)

type Bridge struct {
	BridgeID          int       `db:"bridge_id" json:"bridge_id"`
	Name              string    `db:"name" json:"name"`
	Alias             *string   `db:"alias" json:"alias"`
	Dynasty           string    `db:"dynasty" json:"dynasty"`
	Location          string    `db:"location" json:"location"`
	SpanLength        float64   `db:"span_length" json:"span_length"`
	ArchRise          float64   `db:"arch_rise" json:"arch_rise"`
	DeckWidth         float64   `db:"deck_width" json:"deck_width"`
	TotalLength       float64   `db:"total_length" json:"total_length"`
	MaterialType      string    `db:"material_type" json:"material_type"`
	ConstructionMethod string   `db:"construction_method" json:"construction_method"`
	HistoricalRecord  *string   `db:"historical_record" json:"historical_record"`
	DocumentationSource *string `db:"documentation_source" json:"documentation_source"`
	Status            string    `db:"status" json:"status"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

type BridgeMember struct {
	MemberID              int       `db:"member_id" json:"member_id"`
	BridgeID              int       `db:"bridge_id" json:"bridge_id"`
	MemberCode            string    `db:"member_code" json:"member_code"`
	MemberType            string    `db:"member_type" json:"member_type"`
	CrossSectionType      string    `db:"cross_section_type" json:"cross_section_type"`
	CrossSectionArea      float64   `db:"cross_section_area" json:"cross_section_area"`
	MomentOfInertia       float64   `db:"moment_of_inertia" json:"moment_of_inertia"`
	SectionWidth          float64   `db:"section_width" json:"section_width"`
	SectionHeight         float64   `db:"section_height" json:"section_height"`
	Length                float64   `db:"length" json:"length"`
	MaterialGrade         string    `db:"material_grade" json:"material_grade"`
	ElasticModulus        float64   `db:"elastic_modulus" json:"elastic_modulus"`
	PoissonsRatio         float64   `db:"poissons_ratio" json:"poissons_ratio"`
	AllowableTensileStress float64  `db:"allowable_tensile_stress" json:"allowable_tensile_stress"`
	AllowableCompressiveStress float64 `db:"allowable_compressive_stress" json:"allowable_compressive_stress"`
	AllowableShearStress  float64   `db:"allowable_shear_stress" json:"allowable_shear_stress"`
	StartNodeID           int       `db:"start_node_id" json:"start_node_id"`
	EndNodeID             int       `db:"end_node_id" json:"end_node_id"`
	PositionOrder         int       `db:"position_order" json:"position_order"`
	Description           *string   `db:"description" json:"description"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
}

type BridgeNode struct {
	NodeID       int       `db:"node_id" json:"node_id"`
	BridgeID     int       `db:"bridge_id" json:"bridge_id"`
	NodeCode     string    `db:"node_code" json:"node_code"`
	XCoordinate  float64   `db:"x_coordinate" json:"x_coordinate"`
	YCoordinate  float64   `db:"y_coordinate" json:"y_coordinate"`
	ZCoordinate  float64   `db:"z_coordinate" json:"z_coordinate"`
	ConstraintType string  `db:"constraint_type" json:"constraint_type"`
	NodeType     string    `db:"node_type" json:"node_type"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type Sensor struct {
	SensorID         int       `db:"sensor_id" json:"sensor_id"`
	BridgeID         int       `db:"bridge_id" json:"bridge_id"`
	MemberID         *int      `db:"member_id" json:"member_id"`
	SensorCode       string    `db:"sensor_code" json:"sensor_code"`
	SensorType       string    `db:"sensor_type" json:"sensor_type"`
	MeasurementType  string    `db:"measurement_type" json:"measurement_type"`
	Unit             string    `db:"unit" json:"unit"`
	InstallationLocation string  `db:"installation_location" json:"installation_location"`
	DTUDeviceID      string    `db:"dtu_device_id" json:"dtu_device_id"`
	SamplingInterval int       `db:"sampling_interval" json:"sampling_interval"`
	Status           string    `db:"status" json:"status"`
	CalibrationDate  *time.Time `db:"calibration_date" json:"calibration_date"`
	Accuracy         float64   `db:"accuracy" json:"accuracy"`
	RangeMin         float64   `db:"range_min" json:"range_min"`
	RangeMax         float64   `db:"range_max" json:"range_max"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type SensorData struct {
	SensorID    int       `db:"sensor_id" json:"sensor_id"`
	Timestamp   time.Time `db:"timestamp" json:"timestamp"`
	Value       float64   `db:"value" json:"value"`
	QualityFlag int       `db:"quality_flag" json:"quality_flag"`
	RawData     *string   `db:"raw_data" json:"raw_data"`
}

type EnvironmentalData struct {
	BridgeID      int       `db:"bridge_id" json:"bridge_id"`
	Timestamp     time.Time `db:"timestamp" json:"timestamp"`
	Temperature   float64   `db:"temperature" json:"temperature"`
	Humidity      float64   `db:"humidity" json:"humidity"`
	WindSpeed     float64   `db:"wind_speed" json:"wind_speed"`
	WindDirection float64   `db:"wind_direction" json:"wind_direction"`
	Rainfall      float64   `db:"rainfall" json:"rainfall"`
}

type AnalysisResult struct {
	AnalysisID   int       `db:"analysis_id" json:"analysis_id"`
	BridgeID     int       `db:"bridge_id" json:"bridge_id"`
	AnalysisType string    `db:"analysis_type" json:"analysis_type"`
	AnalysisName string    `db:"analysis_name" json:"analysis_name"`
	LoadCase     string    `db:"load_case" json:"load_case"`
	LoadValue    float64   `db:"load_value" json:"load_value"`
	LoadPosition float64   `db:"load_position" json:"load_position"`
	IsMovingLoad bool      `db:"is_moving_load" json:"is_moving_load"`
	AnalysisTime time.Time `db:"analysis_time" json:"analysis_time"`
	Parameters   *string   `db:"parameters" json:"parameters"`
	Status       string    `db:"status" json:"status"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type MemberForce struct {
	ForceID        int       `db:"force_id" json:"force_id"`
	AnalysisID     int       `db:"analysis_id" json:"analysis_id"`
	MemberID       int       `db:"member_id" json:"member_id"`
	AxialForce     float64   `db:"axial_force" json:"axial_force"`
	ShearForce     float64   `db:"shear_force" json:"shear_force"`
	BendingMoment  float64   `db:"bending_moment" json:"bending_moment"`
	AxialStress    float64   `db:"axial_stress" json:"axial_stress"`
	BendingStress  float64   `db:"bending_stress" json:"bending_stress"`
	CombinedStress float64   `db:"combined_stress" json:"combined_stress"`
	StressRatio    float64   `db:"stress_ratio" json:"stress_ratio"`
	IsOverspeed    bool      `db:"is_overspeed" json:"is_overspeed"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

type NodeDisplacement struct {
	DisplacementID    int       `db:"displacement_id" json:"displacement_id"`
	AnalysisID        int       `db:"analysis_id" json:"analysis_id"`
	NodeID            int       `db:"node_id" json:"node_id"`
	DisplacementX     float64   `db:"displacement_x" json:"displacement_x"`
	DisplacementY     float64   `db:"displacement_y" json:"displacement_y"`
	DisplacementZ     float64   `db:"displacement_z" json:"displacement_z"`
	TotalDisplacement float64   `db:"total_displacement" json:"total_displacement"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

type Alert struct {
	AlertID         int       `db:"alert_id" json:"alert_id"`
	BridgeID        int       `db:"bridge_id" json:"bridge_id"`
	MemberID        *int      `db:"member_id" json:"member_id"`
	SensorID        *int      `db:"sensor_id" json:"sensor_id"`
	AlertType       string    `db:"alert_type" json:"alert_type"`
	AlertLevel      string    `db:"alert_level" json:"alert_level"`
	AlertMessage    string    `db:"alert_message" json:"alert_message"`
	MeasuredValue   float64   `db:"measured_value" json:"measured_value"`
	ThresholdValue  float64   `db:"threshold_value" json:"threshold_value"`
	Timestamp       time.Time `db:"timestamp" json:"timestamp"`
	IsAcknowledged  bool      `db:"is_acknowledged" json:"is_acknowledged"`
	AcknowledgedAt  *time.Time `db:"acknowledged_at" json:"acknowledged_at"`
	AcknowledgedBy  *string   `db:"acknowledged_by" json:"acknowledged_by"`
	MQTTTopic       string    `db:"mqtt_topic" json:"mqtt_topic"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type CraftAnalysis struct {
	AnalysisID           int       `db:"analysis_id" json:"analysis_id"`
	BridgeID             int       `db:"bridge_id" json:"bridge_id"`
	AnalysisDate         time.Time `db:"analysis_date" json:"analysis_date"`
	WoodSpeciesPredicted string    `db:"wood_species_predicted" json:"wood_species_predicted"`
	WoodGradePredicted   string    `db:"wood_grade_predicted" json:"wood_grade_predicted"`
	ConstructionSequence []string  `db:"construction_sequence" json:"construction_sequence"`
	JoineryTypePredicted string    `db:"joinery_type_predicted" json:"joinery_type_predicted"`
	ConfidenceScore      float64   `db:"confidence_score" json:"confidence_score"`
	MethodUsed           string    `db:"method_used" json:"method_used"`
	FeatureImportance    *string   `db:"feature_importance" json:"feature_importance"`
	RawFeatures          *string   `db:"raw_features" json:"raw_features"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
}

type VehicleLoad struct {
	LoadID       int       `db:"load_id" json:"load_id"`
	VehicleType  string    `db:"vehicle_type" json:"vehicle_type"`
	AxleCount    int       `db:"axle_count" json:"axle_count"`
	TotalWeight  float64   `db:"total_weight" json:"total_weight"`
	AxleWeights  []float64 `db:"axle_weights" json:"axle_weights"`
	AxleSpacings []float64 `db:"axle_spacings" json:"axle_spacings"`
	IsStandard   bool      `db:"is_standard" json:"is_standard"`
	Description  *string   `db:"description" json:"description"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type YingzaoFashiSpec struct {
	SpecID            int       `db:"spec_id" json:"spec_id"`
	ComponentType     string    `db:"component_type" json:"component_type"`
	GradeLevel        string    `db:"grade_level" json:"grade_level"`
	MaterialGrade     string    `db:"material_grade" json:"material_grade"`
	MaxSpanRatio      float64   `db:"max_span_ratio" json:"max_span_ratio"`
	MinSectionModulus float64   `db:"min_section_modulus" json:"min_section_modulus"`
	AllowableStress   float64   `db:"allowable_stress" json:"allowable_stress"`
	SafetyFactor      float64   `db:"safety_factor" json:"safety_factor"`
	Dynasty           string    `db:"dynasty" json:"dynasty"`
	SourceChapter     string    `db:"source_chapter" json:"source_chapter"`
	Description       *string   `db:"description" json:"description"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

type WoodTextureFeature struct {
	FeatureID       int       `db:"feature_id" json:"feature_id"`
	BridgeID        int       `db:"bridge_id" json:"bridge_id"`
	MemberID        int       `db:"member_id" json:"member_id"`
	GrainDensity    float64   `db:"grain_density" json:"grain_density"`
	GrainAngle      float64   `db:"grain_angle" json:"grain_angle"`
	LatewoodRatio   float64   `db:"latewood_ratio" json:"latewood_ratio"`
	KnotsCount      int       `db:"knots_count" json:"knots_count"`
	AverageKnotSize float64   `db:"average_knot_size" json:"average_knot_size"`
	Density         float64   `db:"density" json:"density"`
	Hardness        float64   `db:"hardness" json:"hardness"`
	ColorValues     *string   `db:"color_values" json:"color_values"`
	ImageSource     string    `db:"image_source" json:"image_source"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type JoineryFeature struct {
	FeatureID           int       `db:"feature_id" json:"feature_id"`
	BridgeID            int       `db:"bridge_id" json:"bridge_id"`
	MemberID            int       `db:"member_id" json:"member_id"`
	JointType           string    `db:"joint_type" json:"joint_type"`
	TenonLength         float64   `db:"tenon_length" json:"tenon_length"`
	TenonWidth          float64   `db:"tenon_width" json:"tenon_width"`
	TenonThickness      float64   `db:"tenon_thickness" json:"tenon_thickness"`
	MortiseDepth        float64   `db:"mortise_depth" json:"mortise_depth"`
	ShoulderAngle       float64   `db:"shoulder_angle" json:"shoulder_angle"`
	FitTolerance        float64   `db:"fit_tolerance" json:"fit_tolerance"`
	WoodSpecies         string    `db:"wood_species" json:"wood_species"`
	CraftsmanshipRating float64   `db:"craftsmanship_rating" json:"craftsmanship_rating"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
}
