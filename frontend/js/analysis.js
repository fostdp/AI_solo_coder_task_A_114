const API_BASE = 'http://localhost:8080/api/v1';

class BridgeAnalysis {
    constructor() {
        this.currentBridge = null;
        this.bridgeInfo = null;
        this.staticAnalysisResult = null;
        this.movingAnalysisResult = null;
        this.craftAnalysisResult = null;
        this.currentAnimationFrame = 0;
        this.animationInterval = null;
        this.isPlaying = false;
    }

    async loadBridgeInfo(bridgeId) {
        try {
            const response = await fetch(`${API_BASE}/bridges/${bridgeId}`);
            if (response.ok) {
                this.bridgeInfo = await response.json();
                return this.bridgeInfo;
            }
        } catch (e) {
            console.warn('Failed to load bridge info from API, using fallback data');
        }

        this.bridgeInfo = this.getFallbackBridgeInfo(bridgeId);
        return this.bridgeInfo;
    }

    getFallbackBridgeInfo(bridgeId) {
        const bridges = {
            1: { bridge_id: 1, name: '汴水虹桥', dynasty: '北宋', location: '河南开封', span_length: 25.6, arch_rise: 5.8, deck_width: 6.5, total_length: 32.0, construction_method: '叠梁拱' },
            2: { bridge_id: 2, name: '永安桥', dynasty: '南宋', location: '浙江温州', span_length: 21.3, arch_rise: 4.5, deck_width: 4.8, total_length: 28.5, construction_method: '贯木拱' },
            3: { bridge_id: 3, name: '龙津桥', dynasty: '明代', location: '福建寿宁', span_length: 28.5, arch_rise: 6.2, deck_width: 5.2, total_length: 35.0, construction_method: '木拱廊桥' },
            4: { bridge_id: 4, name: '广济桥', dynasty: '宋代', location: '广东潮州', span_length: 23.0, arch_rise: 5.0, deck_width: 5.0, total_length: 30.0, construction_method: '浮桥结合' },
            5: { bridge_id: 5, name: '万安桥', dynasty: '北宋', location: '福建泉州', span_length: 26.8, arch_rise: 5.5, deck_width: 5.8, total_length: 33.5, construction_method: '石木混合' },
            6: { bridge_id: 6, name: '飞虹桥', dynasty: '清代', location: '浙江庆元', span_length: 19.5, arch_rise: 4.2, deck_width: 4.5, total_length: 26.0, construction_method: '木拱廊桥' },
            7: { bridge_id: 7, name: '千乘桥', dynasty: '明代', location: '福建屏南', span_length: 27.3, arch_rise: 5.9, deck_width: 5.0, total_length: 34.0, construction_method: '贯木拱' },
            8: { bridge_id: 8, name: '安澜桥', dynasty: '宋代', location: '四川都江堰', span_length: 24.0, arch_rise: 5.2, deck_width: 4.8, total_length: 31.0, construction_method: '竹索木桥' },
            9: { bridge_id: 9, name: '枫桥', dynasty: '唐代', location: '江苏苏州', span_length: 18.5, arch_rise: 3.8, deck_width: 4.2, total_length: 24.0, construction_method: '单孔木拱' },
            10: { bridge_id: 10, name: '灞桥', dynasty: '唐代', location: '陕西西安', span_length: 22.0, arch_rise: 4.8, deck_width: 5.5, total_length: 29.0, construction_method: '多孔木梁' }
        };
        return bridges[bridgeId] || bridges[1];
    }

    async runStaticAnalysis(bridgeId, loadValue, loadPosition) {
        try {
            const response = await fetch(`${API_BASE}/analysis/static-load`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    bridge_id: bridgeId,
                    load_value: loadValue,
                    load_position: loadPosition
                })
            });

            if (response.ok) {
                this.staticAnalysisResult = await response.json();
                return this.staticAnalysisResult;
            }
        } catch (e) {
            console.warn('Static analysis API failed, using local simulation');
        }

        this.staticAnalysisResult = this.simulateStaticAnalysis(bridgeId, loadValue, loadPosition);
        return this.staticAnalysisResult;
    }

    simulateStaticAnalysis(bridgeId, loadValue, loadPosition) {
        const bridgeInfo = this.getFallbackBridgeInfo(bridgeId);
        const span = bridgeInfo.span_length;
        const numMembers = 40;
        const numNodes = 24;

        const memberForces = [];
        for (let i = 1; i <= numMembers; i++) {
            const position = (i / numMembers) * span;
            const distance = Math.abs(position - loadPosition);
            const influence = Math.max(0, 1 - distance / (span * 0.4));

            const axialForce = -loadValue * (0.3 + 0.7 * influence) * (i % 2 === 0 ? 1 : -1) * 10;
            const bendingMoment = loadValue * influence * 2;
            const shearForce = loadValue * influence * 0.5;

            const area = 0.12;
            const sectionModulus = 0.015;
            const axialStress = axialForce / area / 1000;
            const bendingStress = bendingMoment / sectionModulus / 1000;
            const combinedStress = axialStress + bendingStress;

            const allowableStress = 8.5;
            const stressRatio = Math.abs(combinedStress) / allowableStress;

            memberForces.push({
                member_id: i,
                member_type: i <= 20 ? 'arch_rib' : (i <= 32 ? 'vertical_post' : 'diagonal_brace'),
                axial_force: axialForce,
                shear_force: shearForce,
                bending_moment: bendingMoment,
                axial_stress: axialStress,
                bending_stress: bendingStress,
                combined_stress: combinedStress,
                stress_ratio: stressRatio,
                is_overspeed: stressRatio > 1.0
            });
        }

        const displacements = [];
        for (let i = 1; i <= numNodes; i++) {
            const position = ((i - 1) / (numNodes - 1)) * span;
            const distance = Math.abs(position - loadPosition);
            const influence = Math.max(0, 1 - distance / (span * 0.3));

            const dispX = (position - span / 2) / span * 5 * influence;
            const dispY = -20 * influence * (1 + Math.random() * 0.2);
            const totalDisp = Math.sqrt(dispX * dispX + dispY * dispY);

            displacements.push({
                node_id: i,
                displacement_x: dispX,
                displacement_y: dispY,
                total_displacement: totalDisp
            });
        }

        const maxStressRatio = Math.max(...memberForces.map(m => m.stress_ratio));
        const maxDisplacement = Math.max(...displacements.map(d => Math.abs(d.displacement_y)));

        const yingzaoComparison = memberForces.slice(0, 10).map((mf, i) => ({
            member_id: mf.member_id,
            member_type: mf.member_type,
            actual_stress: mf.combined_stress,
            allowable_stress: 8.5,
            stress_ratio: mf.stress_ratio,
            max_span_ratio: 12.0,
            actual_span_ratio: 8.0 + Math.random() * 3,
            section_modulus: 350 + Math.random() * 100,
            min_section_modulus: 300,
            compliant: mf.stress_ratio <= 1.0,
            spec_grade: i < 5 ? '一等' : '二等'
        }));

        return {
            analysis_id: Date.now(),
            bridge_id: bridgeId,
            analysis_type: 'static',
            member_forces: memberForces,
            displacements: displacements,
            max_stress_ratio: maxStressRatio,
            max_displacement: maxDisplacement,
            yingzao_comparison: yingzaoComparison
        };
    }

    async runMovingAnalysis(bridgeId, totalWeight, steps) {
        try {
            const response = await fetch(`${API_BASE}/analysis/moving-load`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    bridge_id: bridgeId,
                    total_weight: totalWeight,
                    steps: steps
                })
            });

            if (response.ok) {
                this.movingAnalysisResult = await response.json();
                return this.movingAnalysisResult;
            }
        } catch (e) {
            console.warn('Moving analysis API failed, using local simulation');
        }

        this.movingAnalysisResult = this.simulateMovingAnalysis(bridgeId, totalWeight, steps);
        return this.movingAnalysisResult;
    }

    simulateMovingAnalysis(bridgeId, totalWeight, steps) {
        const bridgeInfo = this.getFallbackBridgeInfo(bridgeId);
        const span = bridgeInfo.span_length;
        const results = [];

        for (let i = 0; i <= steps; i++) {
            const position = span * i / steps;
            const result = this.simulateStaticAnalysis(bridgeId, totalWeight, position);
            results.push({
                position: position,
                max_axial: Math.max(...result.member_forces.map(m => Math.abs(m.axial_force))),
                max_moment: Math.max(...result.member_forces.map(m => Math.abs(m.bending_moment))),
                member_forces: result.member_forces,
                displacements: result.displacements
            });
        }

        const maxStressRatio = Math.max(...results.map(r => 
            Math.max(...r.member_forces.map(m => m.stress_ratio))
        ));
        const maxDisplacement = Math.max(...results.map(r => 
            Math.max(...r.displacements.map(d => Math.abs(d.displacement_y)))
        ));

        return {
            analysis_id: Date.now(),
            bridge_id: bridgeId,
            analysis_type: 'moving',
            steps: results.length,
            results: results,
            max_stress_ratio: maxStressRatio,
            max_displacement: maxDisplacement
        };
    }

    async runCraftAnalysis(bridgeId, woodSpecies, craftLevel) {
        try {
            const response = await fetch(`${API_BASE}/craft/analyze`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    bridge_id: bridgeId,
                    wood_species: woodSpecies,
                    craftsmanship_rating: craftLevel
                })
            });

            if (response.ok) {
                this.craftAnalysisResult = await response.json();
                return this.craftAnalysisResult;
            }
        } catch (e) {
            console.warn('Craft analysis API failed, using local simulation');
        }

        this.craftAnalysisResult = this.simulateCraftAnalysis(bridgeId, woodSpecies, craftLevel);
        return this.craftAnalysisResult;
    }

    simulateCraftAnalysis(bridgeId, woodSpecies, craftLevel) {
        const bridgeInfo = this.getFallbackBridgeInfo(bridgeId);

        const sequences = {
            '叠梁拱': [
                '选址测量与基础施工',
                '砌筑桥台与桥墩',
                '搭建施工脚手架',
                '加工拱脚梁木构件',
                '铺设第一层拱架',
                '安装第二层叠梁',
                '逐节向上叠砌拱圈',
                '安装横木与拉结',
                '铺设桥面梁板',
                '安装栏杆与装饰',
                '竣工验收与荷载试验'
            ],
            '贯木拱': [
                '选址测量与地勘',
                '修筑石砌桥台',
                '准备贯木拱构件',
                '制作五边拱骨架',
                '穿插第一组贯木',
                '安装第二组贯木',
                '交错编织拱骨',
                '安装剪刀撑',
                '铺设桥面系统',
                '安装廊屋木架',
                '盖瓦与装饰',
                '完工验收'
            ],
            '木拱廊桥': [
                '风水堪舆与选址',
                '桥基砌筑与桥台',
                '备料与木材加工',
                '搭设木拱架',
                '安装三节拱骨',
                '安装五节拱骨',
                '拱架系统组装',
                '桥面梁架铺设',
                '廊屋柱网安装',
                '梁架斗拱安装',
                '屋面盖瓦',
                '油饰彩画',
                '落成祭祀'
            ]
        };

        const sequence = sequences[bridgeInfo.construction_method] || sequences['叠梁拱'];

        const woodGrades = {
            '杉木': { grade: '二等材', confidence: 0.85 },
            '松木': { grade: '二等材', confidence: 0.82 },
            '柏木': { grade: '一等材', confidence: 0.88 },
            '樟木': { grade: '一等材', confidence: 0.80 },
            '楠木': { grade: '一等材', confidence: 0.90 },
            '黄花梨': { grade: '特等材', confidence: 0.95 },
            '紫檀': { grade: '特等材', confidence: 0.92 }
        };

        const gradeInfo = woodGrades[woodSpecies] || { grade: '二等材', confidence: 0.75 };

        const joineryTypes = [
            { type: '燕尾榫', confidence: 0.85 },
            { type: '齐肩榫', confidence: 0.78 },
            { type: '榫卯结合', confidence: 0.90 }
        ];
        const joinery = joineryTypes[Math.floor(Math.random() * joineryTypes.length)];

        const confidence = (gradeInfo.confidence + joinery.confidence + craftLevel / 5) / 3;

        return {
            analysis_id: Date.now(),
            bridge_id: bridgeId,
            bridge_name: bridgeInfo.name,
            wood_species: woodSpecies,
            wood_grade: gradeInfo.grade,
            construction_sequence: sequence,
            joinery_type: joinery.type,
            confidence_score: Math.min(0.98, confidence),
            feature_importance: {
                density: 0.25,
                latewood_ratio: 0.20,
                knots_count: 0.18,
                hardness: 0.15,
                grain_density: 0.12,
                average_knot_size: 0.10
            },
            method_used: '决策树分类 + 规则引擎',
            wood_features: {
                grain_density: 3.5,
                grain_angle: 10,
                latewood_ratio: 0.4,
                knots_count: 4,
                average_knot_size: 1.0,
                density: 0.55,
                hardness: 4.0
            }
        };
    }

    async getBridgeStructure(bridgeId) {
        try {
            const response = await fetch(`${API_BASE}/analysis/structure/${bridgeId}`);
            if (response.ok) {
                return await response.json();
            }
        } catch (e) {
            console.warn('Structure API failed, using fallback');
        }

        return this.getFallbackBridgeInfo(bridgeId);
    }

    async getSensors(bridgeId) {
        try {
            const response = await fetch(`${API_BASE}/bridges/${bridgeId}/sensors`);
            if (response.ok) {
                const data = await response.json();
                return data.data || [];
            }
        } catch (e) {
            console.warn('Sensors API failed, using fallback');
        }

        return this.getFallbackSensors(bridgeId);
    }

    getFallbackSensors(bridgeId) {
        const sensorTypes = [
            { type: 'displacement', measurement: '位移', unit: 'mm', count: 6 },
            { type: 'strain', measurement: '应变', unit: 'μɛ', count: 8 },
            { type: 'temperature', measurement: '温度', unit: '°C', count: 4 },
            { type: 'humidity', measurement: '湿度', unit: '%RH', count: 3 },
            { type: 'vibration', measurement: '振动', unit: 'mm/s', count: 4 }
        ];

        const sensors = [];
        let idx = 1;
        sensorTypes.forEach(st => {
            for (let i = 0; i < st.count; i++) {
                sensors.push({
                    sensor_id: idx,
                    sensor_code: `S${bridgeId.toString().padStart(2, '0')}-${st.type.toUpperCase()}-${idx.toString().padStart(3, '0')}`,
                    sensor_type: st.type,
                    measurement_type: st.measurement,
                    unit: st.unit,
                    status: 'active',
                    current_value: (st.type === 'temperature' ? 22 : 
                                   st.type === 'humidity' ? 60 :
                                   st.type === 'strain' ? 150 :
                                   st.type === 'displacement' ? 5 : 0.5) + Math.random() * 5
                });
                idx++;
            }
        });

        return sensors;
    }

    async getAlerts(bridgeId) {
        try {
            const response = await fetch(`${API_BASE}/bridges/${bridgeId}/alerts?limit=20`);
            if (response.ok) {
                const data = await response.json();
                return data.data || [];
            }
        } catch (e) {
            console.warn('Alerts API failed, using fallback');
        }

        return this.getFallbackAlerts(bridgeId);
    }

    getFallbackAlerts(bridgeId) {
        const alerts = [
            {
                alert_id: 1,
                alert_type: 'stress_warning',
                alert_level: 'warning',
                alert_message: '拱肋构件应力接近容许值，请注意观察',
                measured_value: 7.2,
                threshold_value: 8.5,
                timestamp: new Date(Date.now() - 3600000).toISOString(),
                is_acknowledged: false
            },
            {
                alert_id: 2,
                alert_type: 'sensor_out_of_range',
                alert_level: 'warning',
                alert_message: '温度传感器读数偏高',
                measured_value: 38.5,
                threshold_value: 35,
                timestamp: new Date(Date.now() - 7200000).toISOString(),
                is_acknowledged: true
            }
        ];
        return alerts;
    }
}

const bridgeAnalysis = new BridgeAnalysis();
