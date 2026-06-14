class CraftPanel {
    constructor() {
        this.currentResult = null;
        this.woodSpeciesList = [];
        this.joineryTypesList = [];
    }

    init() {
        this.bindEvents();
        this.loadDictionaries();
    }

    bindEvents() {
        const craftLevel = document.getElementById('craftLevel');
        if (craftLevel) {
            craftLevel.addEventListener('input', (e) => {
                const stars = this.getStars(parseInt(e.target.value));
                const valueEl = document.getElementById('craftLevelValue');
                if (valueEl) valueEl.textContent = stars;
            });
        }
    }

    getStars(level) {
        const full = '★'.repeat(level);
        const empty = '☆'.repeat(5 - level);
        return full + empty;
    }

    async loadDictionaries() {
        try {
            const [speciesResp, joineryResp] = await Promise.all([
                fetch(`${window.API_BASE || '/api/v1'}/craft/wood-species`),
                fetch(`${window.API_BASE || '/api/v1'}/craft/joinery-types`)
            ]);

            if (speciesResp.ok) {
                const data = await speciesResp.json();
                this.woodSpeciesList = data.data || [];
            }

            if (joineryResp.ok) {
                const data = await joineryResp.json();
                this.joineryTypesList = data.data || [];
            }
        } catch (e) {
            console.warn('Failed to load craft dictionaries:', e);
        }
    }

    async runAnalysis(bridgeId) {
        const woodSpecies = document.getElementById('woodSpecies').value;
        const craftLevel = parseFloat(document.getElementById('craftLevel').value);

        try {
            const response = await fetch(`${window.API_BASE || '/api/v1'}/craft/analyze`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    bridge_id: bridgeId,
                    wood_species: woodSpecies,
                    craftsmanship_rating: craftLevel
                })
            });

            if (response.ok) {
                this.currentResult = await response.json();
            } else {
                throw new Error('API failed');
            }
        } catch (e) {
            console.warn('Craft analysis API failed, using local simulation');
            this.currentResult = this.simulateAnalysis(bridgeId, woodSpecies, craftLevel);
        }

        this.renderResult(this.currentResult);
        return this.currentResult;
    }

    simulateAnalysis(bridgeId, woodSpecies, craftLevel) {
        const woodGrades = {
            '杉木': { grade: '二等材', confidence: 0.72 },
            '松木': { grade: '二等材', confidence: 0.75 },
            '柏木': { grade: '一等材', confidence: 0.78 },
            '樟木': { grade: '一等材', confidence: 0.80 },
            '楠木': { grade: '一等材', confidence: 0.85 },
            '黄花梨': { grade: '特等材', confidence: 0.90 },
            '紫檀': { grade: '特等材', confidence: 0.92 }
        };

        const gradeInfo = woodGrades[woodSpecies] || { grade: '二等材', confidence: 0.75 };

        const joineryTypes = [
            { type: '燕尾榫', confidence: 0.85 },
            { type: '齐肩榫', confidence: 0.75 },
            { type: '榫卯结合', confidence: 0.90 },
            { type: '搭掌榫', confidence: 0.70 }
        ];
        const joinery = joineryTypes[Math.floor(Math.random() * joineryTypes.length)];

        const confidence = (gradeInfo.confidence + joinery.confidence + craftLevel / 5) / 3;

        const sequences = {
            '贯木拱': [
                '1. 桥台基础施工，定位放线',
                '2. 安装拱脚基石，找平灌浆',
                '3. 架设第一组五排拱肋木',
                '4. 安装横向系梁，初固榫卯',
                '5. 架设第二组三排拱肋木',
                '6. 安装斜撑杆件，形成稳定三角',
                '7. 调整各节点松紧度，校正轴线',
                '8. 铺设桥面龙骨，安装横梁',
                '9. 铺设桥面板，校正水平',
                '10. 安装栏杆及装饰构件',
                '11. 油漆防腐处理',
                '12. 荷载试验验收'
            ]
        };

        return {
            analysis_id: 0,
            bridge_id: bridgeId,
            wood_species: woodSpecies,
            wood_grade: gradeInfo.grade,
            construction_sequence: sequences['贯木拱'] || ['施工步骤'],
            joinery_type: joinery.type,
            confidence_score: confidence,
            feature_importance: {
                density: 0.25,
                latewood_ratio: 0.20,
                knots_count: 0.18,
                hardness: 0.15,
                grain_density: 0.12,
                average_knot_size: 0.10
            },
            method_used: '本地模拟(离线模式)',
            wood_features: {
                grain_density: 3.5,
                grain_angle: 8.0,
                latewood_ratio: 0.4
            }
        };
    }

    renderResult(result) {
        const tbody = document.getElementById('compareTableBody');
        if (!tbody || !result) return;

        tbody.innerHTML = '';

        const infoRows = [
            { label: '推断木材', value: result.wood_species },
            { label: '木材等级', value: result.wood_grade },
            { label: '榫卯类型', value: result.joinery_type },
            { label: '置信度', value: (result.confidence_score * 100).toFixed(1) + '%' },
            { label: '分析方法', value: result.method_used }
        ];

        infoRows.forEach(r => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td colspan="2"><strong>${r.label}</strong></td>
                <td colspan="5">${r.value}</td>
            `;
            tbody.appendChild(row);
        });

        if (result.construction_sequence && result.construction_sequence.length > 0) {
            this.renderSectionHeader(tbody, '施工顺序推断');

            result.construction_sequence.forEach((step, index) => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td colspan="2">步骤 ${index + 1}</td>
                    <td colspan="5">${step}</td>
                `;
                tbody.appendChild(row);
            });
        }

        if (result.feature_importance) {
            this.renderSectionHeader(tbody, '特征重要性');

            Object.entries(result.feature_importance).forEach(([key, value]) => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td colspan="2">${this.translateFeature(key)}</td>
                    <td colspan="5">
                        <div style="background:#e0e0e0;border-radius:3px;height:8px;width:100%;">
                            <div style="background:#3498db;border-radius:3px;height:100%;width:${Math.min(value * 100, 100)}%"></div>
                        </div>
                        <span style="font-size:11px;color:#666;">${(value * 100).toFixed(1)}%</span>
                    </td>
                `;
                tbody.appendChild(row);
            });
        }
    }

    renderSectionHeader(tbody, title) {
        const headerRow = document.createElement('tr');
        headerRow.innerHTML = `<td colspan="7" style="background:#f8f9fa;font-weight:600;border-top:2px solid #dee2e6;">${title}</td>`;
        tbody.appendChild(headerRow);
    }

    translateFeature(key) {
        const dict = {
            density: '木材密度',
            latewood_ratio: '晚材率',
            knots_count: '节疤数量',
            hardness: '硬度',
            grain_density: '纹理密度',
            average_knot_size: '平均节疤尺寸',
            grain_angle: '纹理角度',
            color_r: '颜色-R',
            color_g: '颜色-G',
            color_b: '颜色-B'
        };
        return dict[key] || key;
    }

    getHistory(bridgeId, limit = 10) {
        return fetch(`${window.API_BASE || '/api/v1'}/craft/history/${bridgeId}?limit=${limit}`)
            .then(r => r.ok ? r.json() : { total: 0, data: [] })
            .catch(() => ({ total: 0, data: [] }));
    }
}

window.craftPanel = new CraftPanel();
window.runCraftAnalysis = function() {
    return window.craftPanel.runAnalysis(window.currentBridgeId || 1);
};
