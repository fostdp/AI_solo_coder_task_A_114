-- ============================================================
-- 汴水虹桥构件与节点数据初始化
-- 基于杆系有限元模型的木拱桥结构
-- ============================================================

-- 为汴水虹桥（bridge_id = 1）创建节点
-- 拱肋节点（1-11）
INSERT INTO bridge_nodes (bridge_id, node_code, x_coordinate, y_coordinate, z_coordinate, constraint_type, node_type) VALUES
(1, 'A1', 0.00, 0.00, -3.0, 'fixed', 'arch_abutment'),
(1, 'A2', 2.56, 1.25, -3.0, 'free', 'arch_node'),
(1, 'A3', 5.12, 2.35, -3.0, 'free', 'arch_node'),
(1, 'A4', 7.68, 3.30, -3.0, 'free', 'arch_node'),
(1, 'A5', 10.24, 4.10, -3.0, 'free', 'arch_node'),
(1, 'A6', 12.80, 4.65, -3.0, 'free', 'arch_crown'),
(1, 'A7', 15.36, 4.10, -3.0, 'free', 'arch_node'),
(1, 'A8', 17.92, 3.30, -3.0, 'free', 'arch_node'),
(1, 'A9', 20.48, 2.35, -3.0, 'free', 'arch_node'),
(1, 'A10', 23.04, 1.25, -3.0, 'free', 'arch_node'),
(1, 'A11', 25.60, 0.00, -3.0, 'fixed', 'arch_abutment');

-- 右拱肋节点（12-22）
INSERT INTO bridge_nodes (bridge_id, node_code, x_coordinate, y_coordinate, z_coordinate, constraint_type, node_type) VALUES
(1, 'B1', 0.00, 0.00, 3.0, 'fixed', 'arch_abutment'),
(1, 'B2', 2.56, 1.25, 3.0, 'free', 'arch_node'),
(1, 'B3', 5.12, 2.35, 3.0, 'free', 'arch_node'),
(1, 'B4', 7.68, 3.30, 3.0, 'free', 'arch_node'),
(1, 'B5', 10.24, 4.10, 3.0, 'free', 'arch_node'),
(1, 'B6', 12.80, 4.65, 3.0, 'free', 'arch_crown'),
(1, 'B7', 15.36, 4.10, 3.0, 'free', 'arch_node'),
(1, 'B8', 17.92, 3.30, 3.0, 'free', 'arch_node'),
(1, 'B9', 20.48, 2.35, 3.0, 'free', 'arch_node'),
(1, 'B10', 23.04, 1.25, 3.0, 'free', 'arch_node'),
(1, 'B11', 25.60, 0.00, 3.0, 'fixed', 'arch_abutment');

-- 桥面节点（23-33）左侧
INSERT INTO bridge_nodes (bridge_id, node_code, x_coordinate, y_coordinate, z_coordinate, constraint_type, node_type) VALUES
(1, 'D1', 0.00, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D2', 2.56, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D3', 5.12, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D4', 7.68, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D5', 10.24, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D6', 12.80, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D7', 15.36, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D8', 17.92, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D9', 20.48, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D10', 23.04, 5.80, -3.0, 'free', 'deck_node'),
(1, 'D11', 25.60, 5.80, -3.0, 'free', 'deck_node');

-- 桥面节点（34-44）右侧
INSERT INTO bridge_nodes (bridge_id, node_code, x_coordinate, y_coordinate, z_coordinate, constraint_type, node_type) VALUES
(1, 'E1', 0.00, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E2', 2.56, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E3', 5.12, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E4', 7.68, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E5', 10.24, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E6', 12.80, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E7', 15.36, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E8', 17.92, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E9', 20.48, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E10', 23.04, 5.80, 3.0, 'free', 'deck_node'),
(1, 'E11', 25.60, 5.80, 3.0, 'free', 'deck_node');

-- ============================================================
-- 拱肋构件 - 左侧拱肋
-- ============================================================
INSERT INTO bridge_members (bridge_id, member_code, member_type, cross_section_type, 
    cross_section_area, moment_of_inertia, section_width, section_height, 
    length, material_grade, elastic_modulus, poissons_ratio,
    allowable_tensile_stress, allowable_compressive_stress, allowable_shear_stress,
    start_node_id, end_node_id, position_order, description)
VALUES 
(1, 'AL-01', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.85, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    1, 2, 1, '左拱肋第一段'),
(1, 'AL-02', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.78, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    2, 3, 2, '左拱肋第二段'),
(1, 'AL-03', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.72, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    3, 4, 3, '左拱肋第三段'),
(1, 'AL-04', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.68, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    4, 5, 4, '左拱肋第四段'),
(1, 'AL-05', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.65, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    5, 6, 5, '左拱肋第五段（拱顶左侧）'),
(1, 'AL-06', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.65, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    6, 7, 6, '右拱肋第五段（拱顶右侧）'),
(1, 'AL-07', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.68, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    7, 8, 7, '右拱肋第四段'),
(1, 'AL-08', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.72, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    8, 9, 8, '右拱肋第三段'),
(1, 'AL-09', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.78, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    9, 10, 9, '右拱肋第二段'),
(1, 'AL-10', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.85, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    10, 11, 10, '右拱肋第一段');

-- ============================================================
-- 拱肋构件 - 右侧拱肋
-- ============================================================
INSERT INTO bridge_members (bridge_id, member_code, member_type, cross_section_type, 
    cross_section_area, moment_of_inertia, section_width, section_height, 
    length, material_grade, elastic_modulus, poissons_ratio,
    allowable_tensile_stress, allowable_compressive_stress, allowable_shear_stress,
    start_node_id, end_node_id, position_order, description)
VALUES 
(1, 'AR-01', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.85, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    12, 13, 11, '右拱肋第一段'),
(1, 'AR-02', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.78, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    13, 14, 12, '右拱肋第二段'),
(1, 'AR-03', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.72, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    14, 15, 13, '右拱肋第三段'),
(1, 'AR-04', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.68, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    15, 16, 14, '右拱肋第四段'),
(1, 'AR-05', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.65, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    16, 17, 15, '右拱肋第五段（拱顶右侧）'),
(1, 'AR-06', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.65, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    17, 18, 16, '左拱肋第六段（拱顶左侧）'),
(1, 'AR-07', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.68, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    18, 19, 17, '左拱肋第七段'),
(1, 'AR-08', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.72, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    19, 20, 18, '左拱肋第八段'),
(1, 'AR-09', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.78, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    20, 21, 19, '左拱肋第九段'),
(1, 'AR-10', 'arch_rib', 'rectangle', 0.12, 0.0015, 0.30, 0.40, 
    2.85, '一等材', 10000000000, 0.3, 8.5, 12.0, 2.5, 
    21, 22, 20, '左拱肋第十段');

-- ============================================================
-- 立柱构件
-- ============================================================
INSERT INTO bridge_members (bridge_id, member_code, member_type, cross_section_type, 
    cross_section_area, moment_of_inertia, section_width, section_height, 
    length, material_grade, elastic_modulus, poissons_ratio,
    allowable_tensile_stress, allowable_compressive_stress, allowable_shear_stress,
    start_node_id, end_node_id, position_order, description)
SELECT 
    1,
    'VL-' || LPAD(ROW_NUMBER() OVER ()::text, 2, '0'),
    'vertical_post',
    'rectangle',
    0.09,
    0.0008,
    0.25,
    0.36,
    5.80 - y_coordinate,
    '二等材',
    9000000000,
    0.3,
    7.5,
    10.0,
    2.0,
    node_id,
    node_id + 22,
    30 + ROW_NUMBER() OVER (),
    '左侧立柱第' || ROW_NUMBER() OVER () || '根'
FROM bridge_nodes 
WHERE bridge_id = 1 AND node_code LIKE 'A%';

-- 右侧立柱
INSERT INTO bridge_members (bridge_id, member_code, member_type, cross_section_type, 
    cross_section_area, moment_of_inertia, section_width, section_height, 
    length, material_grade, elastic_modulus, poissons_ratio,
    allowable_tensile_stress, allowable_compressive_stress, allowable_shear_stress,
    start_node_id, end_node_id, position_order, description)
SELECT 
    1,
    'VR-' || LPAD(ROW_NUMBER() OVER ()::text, 2, '0'),
    'vertical_post',
    'rectangle',
    0.09,
    0.0008,
    0.25,
    0.36,
    5.80 - y_coordinate,
    '二等材',
    9000000000,
    0.3,
    7.5,
    10.0,
    2.0,
    node_id,
    node_id + 22,
    40 + ROW_NUMBER() OVER (),
    '右侧立柱第' || ROW_NUMBER() OVER () || '根'
FROM bridge_nodes 
WHERE bridge_id = 1 AND node_code LIKE 'B%';

-- ============================================================
-- 桥面梁构件
-- ============================================================
INSERT INTO bridge_members (bridge_id, member_code, member_type, cross_section_type, 
    cross_section_area, moment_of_inertia, section_width, section_height, 
    length, material_grade, elastic_modulus, poissons_ratio,
    allowable_tensile_stress, allowable_compressive_stress, allowable_shear_stress,
    start_node_id, end_node_id, position_order, description)
VALUES 
(1, 'DL-01', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    23, 24, 51, '左桥面梁第一段'),
(1, 'DL-02', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    24, 25, 52, '左桥面梁第二段'),
(1, 'DL-03', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    25, 26, 53, '左桥面梁第三段'),
(1, 'DL-04', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    26, 27, 54, '左桥面梁第四段'),
(1, 'DL-05', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    27, 28, 55, '左桥面梁第五段'),
(1, 'DL-06', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    28, 29, 56, '左桥面梁第六段'),
(1, 'DL-07', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    29, 30, 57, '左桥面梁第七段'),
(1, 'DL-08', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    30, 31, 58, '左桥面梁第八段'),
(1, 'DL-09', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    31, 32, 59, '左桥面梁第九段'),
(1, 'DL-10', 'deck_beam', 'rectangle', 0.10, 0.0012, 0.28, 0.36, 
    2.56, '一等材', 10000000000, 0.3, 8.0, 11.0, 2.2, 
    32, 33, 60, '左桥面梁第十段');

-- ============================================================
-- 传感器配置
-- ============================================================
INSERT INTO sensors (bridge_id, member_id, sensor_code, sensor_type, measurement_type, 
    unit, installation_location, dtu_device_id, sampling_interval, 
    accuracy, range_min, range_max)
VALUES 
(1, 1, 'S01-DISP-001', 'displacement', '位移', 'mm', '左拱脚', 'DTU-BIANSHUI-001', 3600, 0.01, -50, 50),
(1, 5, 'S01-DISP-002', 'displacement', '位移', 'mm', '左拱顶', 'DTU-BIANSHUI-001', 3600, 0.01, -50, 50),
(1, 10, 'S01-DISP-003', 'displacement', '位移', 'mm', '右拱脚', 'DTU-BIANSHUI-001', 3600, 0.01, -50, 50),
(1, 3, 'S01-DISP-004', 'displacement', '位移', 'mm', '左拱四分之一', 'DTU-BIANSHUI-001', 3600, 0.01, -50, 50),
(1, 7, 'S01-DISP-005', 'displacement', '位移', 'mm', '右拱四分之一', 'DTU-BIANSHUI-001', 3600, 0.01, -50, 50),
(1, 15, 'S01-DISP-006', 'displacement', '位移', 'mm', '桥面跨中', 'DTU-BIANSHUI-001', 3600, 0.01, -50, 50),

(1, 1, 'S01-STR-001', 'strain', '应变', 'μɛ', '左拱脚上下缘', 'DTU-BIANSHUI-001', 3600, 1, -2000, 2000),
(1, 5, 'S01-STR-002', 'strain', '应变', 'μɛ', '左拱顶上下缘', 'DTU-BIANSHUI-001', 3600, 1, -2000, 2000),
(1, 10, 'S01-STR-003', 'strain', '应变', 'μɛ', '右拱脚上下缘', 'DTU-BIANSHUI-001', 3600, 1, -2000, 2000),
(1, 3, 'S01-STR-004', 'strain', '应变', 'μɛ', '左拱四分之一', 'DTU-BIANSHUI-001', 3600, 1, -2000, 2000),
(1, 7, 'S01-STR-005', 'strain', '应变', 'μɛ', '右拱四分之一', 'DTU-BIANSHUI-001', 3600, 1, -2000, 2000),
(1, 15, 'S01-STR-006', 'strain', '应变', 'μɛ', '桥面跨中', 'DTU-BIANSHUI-001', 3600, 1, -2000, 2000),
(1, 12, 'S01-STR-007', 'strain', '应变', 'μɛ', '左立柱中部', 'DTU-BIANSHUI-001', 3600, 1, -2000, 2000),
(1, 18, 'S01-STR-008', 'strain', '应变', 'μɛ', '右立柱中部', 'DTU-BIANSHUI-001', 3600, 1, -2000, 2000),

(1, NULL, 'S01-TEMP-001', 'temperature', '温度', '°C', '拱顶环境', 'DTU-BIANSHUI-001', 3600, 0.1, -20, 60),
(1, NULL, 'S01-TEMP-002', 'temperature', '温度', '°C', '拱脚环境', 'DTU-BIANSHUI-001', 3600, 0.1, -20, 60),
(1, NULL, 'S01-TEMP-003', 'temperature', '温度', '°C', '桥面板温度', 'DTU-BIANSHUI-001', 3600, 0.1, -20, 60),
(1, NULL, 'S01-TEMP-004', 'temperature', '温度', '°C', '构件内部温度', 'DTU-BIANSHUI-001', 3600, 0.1, -20, 60),

(1, NULL, 'S01-HUM-001', 'humidity', '湿度', '%RH', '拱下空间', 'DTU-BIANSHUI-001', 3600, 1, 0, 100),
(1, NULL, 'S01-HUM-002', 'humidity', '湿度', '%RH', '桥面环境', 'DTU-BIANSHUI-001', 3600, 1, 0, 100),
(1, NULL, 'S01-HUM-003', 'humidity', '湿度', '%RH', '木材表面', 'DTU-BIANSHUI-001', 3600, 1, 0, 100),

(1, 1, 'S01-VIB-001', 'vibration', '振动速度', 'mm/s', '左拱脚', 'DTU-BIANSHUI-001', 3600, 0.01, 0, 50),
(1, 5, 'S01-VIB-002', 'vibration', '振动速度', 'mm/s', '拱顶', 'DTU-BIANSHUI-001', 3600, 0.01, 0, 50),
(1, 10, 'S01-VIB-003', 'vibration', '振动速度', 'mm/s', '右拱脚', 'DTU-BIANSHUI-001', 3600, 0.01, 0, 50),
(1, 15, 'S01-VIB-004', 'vibration', '振动速度', 'mm/s', '桥面跨中', 'DTU-BIANSHUI-001', 3600, 0.01, 0, 50),

(1, NULL, 'S01-TILT-001', 'tilt', '倾角', '°', '左桥台', 'DTU-BIANSHUI-001', 3600, 0.001, -5, 5),
(1, NULL, 'S01-TILT-002', 'tilt', '倾角', '°', '右桥台', 'DTU-BIANSHUI-001', 3600, 0.001, -5, 5);
