-- ============================================================
-- 古代木拱桥结构力学仿真与数字化复原系统
-- TimescaleDB 数据库初始化脚本
-- ============================================================

-- 创建数据库
CREATE DATABASE IF NOT EXISTS ancient_bridges;
\c ancient_bridges;

-- 启用 TimescaleDB 扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS postgis;

-- ============================================================
-- 桥梁基础信息表
-- ============================================================
CREATE TABLE IF NOT EXISTS bridges (
    bridge_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    alias VARCHAR(200),
    dynasty VARCHAR(50),
    location VARCHAR(200),
    span_length DECIMAL(10, 2),
    arch_rise DECIMAL(10, 2),
    deck_width DECIMAL(10, 2),
    total_length DECIMAL(10, 2),
    material_type VARCHAR(50),
    construction_method VARCHAR(100),
    historical_record TEXT,
    documentation_source TEXT,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 桥梁构件表（杆件系统）
-- ============================================================
CREATE TABLE IF NOT EXISTS bridge_members (
    member_id SERIAL PRIMARY KEY,
    bridge_id INTEGER REFERENCES bridges(bridge_id),
    member_code VARCHAR(50) NOT NULL,
    member_type VARCHAR(50),
    cross_section_type VARCHAR(50),
    cross_section_area DECIMAL(12, 4),
    moment_of_inertia DECIMAL(15, 6),
    section_width DECIMAL(10, 3),
    section_height DECIMAL(10, 3),
    length DECIMAL(12, 4),
    material_grade VARCHAR(50),
    elastic_modulus DECIMAL(15, 2),
    poissons_ratio DECIMAL(6, 4),
    allowable_tensile_stress DECIMAL(12, 2),
    allowable_compressive_stress DECIMAL(12, 2),
    allowable_shear_stress DECIMAL(12, 2),
    start_node_id INTEGER,
    end_node_id INTEGER,
    position_order INTEGER,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 节点坐标表
-- ============================================================
CREATE TABLE IF NOT EXISTS bridge_nodes (
    node_id SERIAL PRIMARY KEY,
    bridge_id INTEGER REFERENCES bridges(bridge_id),
    node_code VARCHAR(50) NOT NULL,
    x_coordinate DECIMAL(12, 4),
    y_coordinate DECIMAL(12, 4),
    z_coordinate DECIMAL(12, 4),
    constraint_type VARCHAR(30),
    node_type VARCHAR(30),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 传感器表
-- ============================================================
CREATE TABLE IF NOT EXISTS sensors (
    sensor_id SERIAL PRIMARY KEY,
    bridge_id INTEGER REFERENCES bridges(bridge_id),
    member_id INTEGER REFERENCES bridge_members(member_id),
    sensor_code VARCHAR(50) UNIQUE NOT NULL,
    sensor_type VARCHAR(50) NOT NULL,
    measurement_type VARCHAR(50),
    unit VARCHAR(20),
    installation_location VARCHAR(200),
    dtu_device_id VARCHAR(100),
    sampling_interval INTEGER DEFAULT 3600,
    status VARCHAR(20) DEFAULT 'active',
    calibration_date DATE,
    accuracy DECIMAL(10, 4),
    range_min DECIMAL(15, 4),
    range_max DECIMAL(15, 4),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 传感器数据表（时序表 - 使用 TimescaleDB 超表）
-- ============================================================
CREATE TABLE IF NOT EXISTS sensor_data (
    sensor_id INTEGER REFERENCES sensors(sensor_id),
    timestamp TIMESTAMPTZ NOT NULL,
    value DECIMAL(15, 6) NOT NULL,
    quality_flag SMALLINT DEFAULT 0,
    raw_data JSONB,
    PRIMARY KEY (sensor_id, timestamp)
);

-- 转换为 TimescaleDB 超表
SELECT create_hypertable('sensor_data', 'timestamp', if_not_exists => TRUE);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_sensor_data_bridge_time 
ON sensor_data (sensor_id, timestamp DESC);

-- ============================================================
-- 环境监测数据表
-- ============================================================
CREATE TABLE IF NOT EXISTS environmental_data (
    bridge_id INTEGER REFERENCES bridges(bridge_id),
    timestamp TIMESTAMPTZ NOT NULL,
    temperature DECIMAL(8, 2),
    humidity DECIMAL(8, 2),
    wind_speed DECIMAL(8, 2),
    wind_direction DECIMAL(6, 1),
    rainfall DECIMAL(8, 2),
    PRIMARY KEY (bridge_id, timestamp)
);

SELECT create_hypertable('environmental_data', 'timestamp', if_not_exists => TRUE);

-- ============================================================
-- 结构分析结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS analysis_results (
    analysis_id SERIAL PRIMARY KEY,
    bridge_id INTEGER REFERENCES bridges(bridge_id),
    analysis_type VARCHAR(50) NOT NULL,
    analysis_name VARCHAR(100),
    load_case VARCHAR(100),
    load_value DECIMAL(12, 4),
    load_position DECIMAL(10, 4),
    is_moving_load BOOLEAN DEFAULT FALSE,
    analysis_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    parameters JSONB,
    status VARCHAR(20) DEFAULT 'completed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 构件内力结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS member_forces (
    force_id SERIAL PRIMARY KEY,
    analysis_id INTEGER REFERENCES analysis_results(analysis_id),
    member_id INTEGER REFERENCES bridge_members(member_id),
    axial_force DECIMAL(15, 4),
    shear_force DECIMAL(15, 4),
    bending_moment DECIMAL(15, 4),
    axial_stress DECIMAL(12, 4),
    bending_stress DECIMAL(12, 4),
    combined_stress DECIMAL(12, 4),
    stress_ratio DECIMAL(8, 4),
    is_overspeed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 节点位移结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS node_displacements (
    displacement_id SERIAL PRIMARY KEY,
    analysis_id INTEGER REFERENCES analysis_results(analysis_id),
    node_id INTEGER REFERENCES bridge_nodes(node_id),
    displacement_x DECIMAL(12, 6),
    displacement_y DECIMAL(12, 6),
    displacement_z DECIMAL(12, 6),
    total_displacement DECIMAL(12, 6),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 告警记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS alerts (
    alert_id SERIAL PRIMARY KEY,
    bridge_id INTEGER REFERENCES bridges(bridge_id),
    member_id INTEGER REFERENCES bridge_members(member_id),
    sensor_id INTEGER REFERENCES sensors(sensor_id),
    alert_type VARCHAR(50) NOT NULL,
    alert_level VARCHAR(20) NOT NULL,
    alert_message TEXT,
    measured_value DECIMAL(15, 4),
    threshold_value DECIMAL(15, 4),
    timestamp TIMESTAMPTZ NOT NULL,
    is_acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMP,
    acknowledged_by VARCHAR(100),
    mqtt_topic VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_alerts_bridge_time 
ON alerts (bridge_id, timestamp DESC);

-- ============================================================
-- 工艺反演结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS craft_analysis (
    analysis_id SERIAL PRIMARY KEY,
    bridge_id INTEGER REFERENCES bridges(bridge_id),
    analysis_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    wood_species_predicted VARCHAR(100),
    wood_grade_predicted VARCHAR(50),
    construction_sequence TEXT[],
    joinery_type_predicted VARCHAR(100),
    confidence_score DECIMAL(6, 4),
    method_used VARCHAR(50),
    feature_importance JSONB,
    raw_features JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 木材纹理特征表
-- ============================================================
CREATE TABLE IF NOT EXISTS wood_texture_features (
    feature_id SERIAL PRIMARY KEY,
    bridge_id INTEGER REFERENCES bridges(bridge_id),
    member_id INTEGER REFERENCES bridge_members(member_id),
    grain_density DECIMAL(8, 2),
    grain_angle DECIMAL(6, 2),
    latewood_ratio DECIMAL(6, 4),
    knots_count INTEGER,
    average_knot_size DECIMAL(8, 2),
    density DECIMAL(10, 3),
    hardness DECIMAL(8, 2),
    color_values JSONB,
    image_source VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 榫卯节点特征表
-- ============================================================
CREATE TABLE IF NOT EXISTS joinery_features (
    feature_id SERIAL PRIMARY KEY,
    bridge_id INTEGER REFERENCES bridges(bridge_id),
    member_id INTEGER REFERENCES bridge_members(member_id),
    joint_type VARCHAR(100),
    tenon_length DECIMAL(8, 3),
    tenon_width DECIMAL(8, 3),
    tenon_thickness DECIMAL(8, 3),
    mortise_depth DECIMAL(8, 3),
    shoulder_angle DECIMAL(6, 2),
    fit_tolerance DECIMAL(8, 4),
    wood_species VARCHAR(100),
    craftsmanship_rating DECIMAL(4, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 车辆荷载表
-- ============================================================
CREATE TABLE IF NOT EXISTS vehicle_loads (
    load_id SERIAL PRIMARY KEY,
    vehicle_type VARCHAR(50),
    axle_count INTEGER,
    total_weight DECIMAL(10, 2),
    axle_weights DECIMAL(8, 2)[],
    axle_spacings DECIMAL(8, 2)[],
    is_standard BOOLEAN DEFAULT FALSE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 《营造法式》规范参数表
-- ============================================================
CREATE TABLE IF NOT EXISTS yingzao_fashi_specs (
    spec_id SERIAL PRIMARY KEY,
    component_type VARCHAR(100) NOT NULL,
    grade_level VARCHAR(20),
    material_grade VARCHAR(50),
    max_span_ratio DECIMAL(8, 4),
    min_section_modulus DECIMAL(12, 4),
    allowable_stress DECIMAL(10, 2),
    safety_factor DECIMAL(6, 3),
    dynasty VARCHAR(50) DEFAULT 'Song',
    source_chapter VARCHAR(100),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 插入初始数据 - 10座古代木拱桥
-- ============================================================
INSERT INTO bridges (name, alias, dynasty, location, span_length, arch_rise, deck_width, total_length, material_type, construction_method, historical_record, documentation_source) VALUES
('汴水虹桥', '汴京虹桥', '北宋', '河南开封', 25.6, 5.8, 6.5, 32.0, '木材', '叠梁拱', '《清明上河图》中记录的著名木拱桥', '清明上河图 宋 张择端'),
('永安桥', NULL, '南宋', '浙江温州', 21.3, 4.5, 4.8, 28.5, '木材', '贯木拱', '浙南廊桥代表作品', '中国古代桥梁史'),
('龙津桥', '廊桥', '明代', '福建寿宁', 28.5, 6.2, 5.2, 35.0, '木材', '木拱廊桥', '闽东北木拱廊桥典范', '福建古桥志'),
('广济桥', '湘子桥', '宋代', '广东潮州', 23.0, 5.0, 5.0, 30.0, '木材', '浮桥结合', '中国四大古桥之一', '潮州府志'),
('万安桥', '洛阳桥', '北宋', '福建泉州', 26.8, 5.5, 5.8, 33.5, '木材', '石木混合', '福建宋代木拱桥', '泉州府志'),
('飞虹桥', '木拱廊桥', '清代', '浙江庆元', 19.5, 4.2, 4.5, 26.0, '木材', '木拱廊桥', '浙南木拱廊桥代表', '庆元县志'),
('千乘桥', NULL, '明代', '福建屏南', 27.3, 5.9, 5.0, 34.0, '木材', '贯木拱', '闽北木拱桥精品', '屏南县志'),
('安澜桥', '珠浦桥', '宋代', '四川都江堰', 24.0, 5.2, 4.8, 31.0, '木材', '竹索木桥', '四川古代木桥代表', '都江堰志'),
('枫桥', '封桥', '唐代', '江苏苏州', 18.5, 3.8, 4.2, 24.0, '木材', '单孔木拱', '《枫桥夜泊》描绘之桥', '苏州府志'),
('灞桥', '销魂桥', '唐代', '陕西西安', 22.0, 4.8, 5.5, 29.0, '木材', '多孔木梁', '长安城东古桥', '唐两京城坊考');

-- ============================================================
-- 插入标准车辆荷载
-- ============================================================
INSERT INTO vehicle_loads (vehicle_type, axle_count, total_weight, axle_weights, axle_spacings, is_standard, description) VALUES
('宋代牛车', 2, 5.0, ARRAY[2.5, 2.5], ARRAY[2.5], true, '《清明上河图》中典型牛车'),
('宋代马车', 2, 3.5, ARRAY[1.8, 1.7], ARRAY[2.2], true, '宋代载客马车'),
('人力独轮车', 1, 0.8, ARRAY[0.8], ARRAY[]::decimal[], true, '宋代货运独轮车'),
('重型货车', 3, 12.0, ARRAY[4.0, 4.0, 4.0], ARRAY[3.0, 3.0], false, '模拟重型运输车辆'),
('行人荷载', 0, 0.5, ARRAY[]::decimal[], ARRAY[]::decimal[], true, '人均等效荷载');

-- ============================================================
-- 插入《营造法式》规范数据
-- ============================================================
INSERT INTO yingzao_fashi_specs (component_type, grade_level, material_grade, max_span_ratio, min_section_modulus, allowable_stress, safety_factor, source_chapter, description) VALUES
('梁栿', '一等', '一等材', 12.0, 580.0, 8.5, 3.0, '大木作制度一', '殿阁类建筑大梁规范'),
('梁栿', '二等', '二等材', 12.0, 430.0, 8.5, 3.0, '大木作制度一', '厅堂类建筑大梁规范'),
('梁栿', '三等', '三等材', 12.0, 300.0, 8.5, 3.0, '大木作制度一', '余屋类建筑大梁规范'),
('拱枋', '一等', '一等材', 15.0, 280.0, 10.0, 2.5, '大木作制度二', '斗拱拱件规范'),
('柱', '一等', '一等材', 10.0, NULL, 12.0, 3.5, '大木作制度三', '殿阁立柱规范');

-- ============================================================
-- 创建视图 - 桥梁应力状态总览
-- ============================================================
CREATE OR REPLACE VIEW bridge_stress_overview AS
SELECT 
    b.bridge_id,
    b.name,
    b.span_length,
    COUNT(DISTINCT m.member_id) AS total_members,
    COUNT(DISTINCT s.sensor_id) AS total_sensors,
    MAX(mf.stress_ratio) AS max_stress_ratio,
    SUM(CASE WHEN mf.is_overspeed THEN 1 ELSE 0 END) AS overspeed_count,
    MAX(a.analysis_time) AS last_analysis_time
FROM bridges b
LEFT JOIN bridge_members m ON b.bridge_id = m.bridge_id
LEFT JOIN sensors s ON b.bridge_id = s.bridge_id
LEFT JOIN analysis_results a ON b.bridge_id = a.bridge_id
LEFT JOIN member_forces mf ON a.analysis_id = mf.analysis_id
GROUP BY b.bridge_id, b.name, b.span_length;

-- ============================================================
-- 创建连续聚合视图 - 传感器小时统计
-- ============================================================
CREATE MATERIALIZED VIEW sensor_data_hourly
WITH (timescaledb.continuous) AS
SELECT 
    sensor_id,
    time_bucket('1 hour', timestamp) AS bucket,
    AVG(value) AS avg_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    COUNT(*) AS sample_count
FROM sensor_data
GROUP BY sensor_id, time_bucket('1 hour', timestamp)
WITH NO DATA;

SELECT add_continuous_aggregate_policy('sensor_data_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);
