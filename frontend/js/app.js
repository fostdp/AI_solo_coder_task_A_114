let bridgeView = null;
let currentView = 'model';
let currentBridgeId = 1;

document.addEventListener('DOMContentLoaded', function() {
    initApplication();
});

function initApplication() {
    bridgeView = new Bridge3DView('bridgeCanvas');
    
    loadBridge(1);
    
    setupEventListeners();
    
    setTimeout(() => {
        bridgeView.onResize();
    }, 100);
}

function setupEventListeners() {
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            switchView(item.dataset.view);
        });
    });
    
    document.getElementById('bridgeSelector').addEventListener('change', (e) => {
        const bridgeId = parseInt(e.target.value);
        loadBridge(bridgeId);
    });
    
    document.getElementById('showMembers').addEventListener('change', (e) => {
        bridgeView.toggleMembers(e.target.checked);
    });
    
    document.getElementById('showDeck').addEventListener('change', (e) => {
        bridgeView.toggleDeck(e.target.checked);
    });
    
    document.getElementById('showNodes').addEventListener('change', (e) => {
        bridgeView.toggleNodes(e.target.checked);
    });
    
    document.getElementById('showForces').addEventListener('change', (e) => {
        bridgeView.toggleForces(e.target.checked);
        document.getElementById('legendPanel').style.display = e.target.checked ? 'block' : 'none';
    });
    
    document.getElementById('showDeformation').addEventListener('change', (e) => {
        bridgeView.toggleDeformation(e.target.checked);
    });
    
    document.getElementById('deformationScale').addEventListener('input', (e) => {
        const value = parseInt(e.target.value);
        document.getElementById('deformationValue').textContent = value + 'x';
        bridgeView.setDeformationScale(value);
    });
    
    document.getElementById('loadPosition').addEventListener('input', (e) => {
        const value = parseInt(e.target.value);
        const bridgeInfo = bridgeAnalysis.getFallbackBridgeInfo(currentBridgeId);
        const position = bridgeInfo.span_length * value / 100;
        document.getElementById('positionValue').textContent = position.toFixed(1) + ' m';
    });
    
    if (window.craftPanel) {
        window.craftPanel.init();
    }
    
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;
            switchTab(tab);
        });
    });
    
    document.getElementById('playBtn').addEventListener('click', toggleAnimation);
}

function switchView(viewName) {
    currentView = viewName;
    
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.toggle('active', item.dataset.view === viewName);
    });
    
    document.getElementById('modelControls').style.display = viewName === 'model' ? 'block' : 'none';
    document.getElementById('analysisControls').style.display = viewName === 'analysis' ? 'block' : 'none';
    document.getElementById('movingControls').style.display = viewName === 'moving' ? 'block' : 'none';
    document.getElementById('craftControls').style.display = viewName === 'craft' ? 'block' : 'none';
    document.getElementById('sensorControls').style.display = viewName === 'sensors' ? 'block' : 'none';
    document.getElementById('alertControls').style.display = viewName === 'alerts' ? 'block' : 'none';
    
    document.getElementById('statsPanel').style.display = (viewName === 'analysis' || viewName === 'moving') ? 'block' : 'none';
    document.getElementById('dataPanel').style.display = (viewName === 'analysis' || viewName === 'moving' || viewName === 'craft') ? 'block' : 'none';
    
    if (viewName === 'sensors') {
        loadSensorData();
    }
    
    if (viewName === 'alerts') {
        loadAlertData();
    }
}

function switchTab(tabName) {
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.tab === tabName);
    });
    
    document.querySelectorAll('.tab-pane').forEach(pane => {
        pane.classList.remove('active');
    });
    
    document.getElementById('tab-' + tabName).classList.add('active');
}

async function loadBridge(bridgeId) {
    currentBridgeId = bridgeId;
    
    const bridgeInfo = await bridgeAnalysis.loadBridgeInfo(bridgeId);
    
    document.getElementById('bridgeName').textContent = bridgeInfo.name;
    document.getElementById('bridgeDynasty').textContent = bridgeInfo.dynasty;
    document.getElementById('bridgeSpan').textContent = bridgeInfo.span_length + ' m';
    document.getElementById('bridgeRise').textContent = bridgeInfo.arch_rise + ' m';
    document.getElementById('bridgeWidth').textContent = bridgeInfo.deck_width + ' m';
    
    bridgeView.loadBridge(bridgeInfo);
    
    clearAnalysisResults();
    
    if (currentView === 'sensors') {
        loadSensorData();
    }
    
    if (currentView === 'alerts') {
        loadAlertData();
    }
}

function clearAnalysisResults() {
    document.getElementById('maxStressRatio').textContent = '0.00';
    document.getElementById('maxDisplacement').textContent = '0.00 mm';
    document.getElementById('forcesTableBody').innerHTML = '';
    document.getElementById('displacementsTableBody').innerHTML = '';
    document.getElementById('compareTableBody').innerHTML = '';
    
    bridgeView.setForces([]);
    bridgeView.setDisplacements([]);
}

async function runStaticAnalysis() {
    const loadValue = parseFloat(document.getElementById('loadValue').value);
    const loadPosition = parseFloat(document.getElementById('loadPosition').value) / 100;
    
    const bridgeInfo = bridgeAnalysis.getFallbackBridgeInfo(currentBridgeId);
    const position = bridgeInfo.span_length * loadPosition;
    
    const result = await bridgeAnalysis.runStaticAnalysis(currentBridgeId, loadValue, position);
    
    updateForcesTable(result.member_forces);
    updateDisplacementsTable(result.displacements);
    updateCompareTable(result.yingzao_comparison);
    
    document.getElementById('maxStressRatio').textContent = result.max_stress_ratio.toFixed(3);
    document.getElementById('maxDisplacement').textContent = result.max_displacement.toFixed(2) + ' mm';
    
    bridgeView.setForces(result.member_forces);
    bridgeView.setDisplacements(result.displacements);
    
    if (document.getElementById('showForces').checked) {
        bridgeView.updateMemberColors();
    }
    
    if (document.getElementById('showDeformation').checked) {
        bridgeView.applyDeformation();
    }
    
    switchTab('forces');
}

async function runMovingAnalysis() {
    const totalWeight = parseFloat(document.getElementById('vehicleType').value);
    const steps = parseInt(document.getElementById('movingSteps').value);
    
    const result = await bridgeAnalysis.runMovingAnalysis(currentBridgeId, totalWeight, steps);
    
    document.getElementById('maxStressRatio').textContent = result.max_stress_ratio.toFixed(3);
    document.getElementById('maxDisplacement').textContent = result.max_displacement.toFixed(2) + ' mm';
    
    const midIndex = Math.floor(result.results.length / 2);
    const midResult = result.results[midIndex];
    updateForcesTable(midResult.member_forces);
    updateDisplacementsTable(midResult.displacements);
    
    bridgeView.setForces(midResult.member_forces);
    bridgeView.setDisplacements(midResult.displacements);
    
    bridgeView.createVehicle('ox');
    bridgeView.setVehiclePosition(0.5);
    
    bridgeView.currentAnimationFrame = midIndex;
    bridgeView.movingResults = result.results;
    
    switchTab('forces');
}

function toggleAnimation() {
    const btn = document.getElementById('playBtn');
    
    if (bridgeView.isPlaying) {
        stopAnimation();
        btn.textContent = '▶ 播放动画';
    } else {
        startAnimation();
        btn.textContent = '⏸ 暂停动画';
    }
}

function startAnimation() {
    if (!bridgeView.movingResults || bridgeView.movingResults.length === 0) {
        return;
    }
    
    bridgeView.isPlaying = true;
    bridgeView.currentAnimationFrame = 0;
    
    const speed = parseFloat(document.getElementById('animSpeed').value) || 1;
    const interval = 200 / speed;
    
    bridgeView.animationInterval = setInterval(() => {
        const result = bridgeView.movingResults[bridgeView.currentAnimationFrame];
        
        bridgeView.setForces(result.member_forces);
        bridgeView.setDisplacements(result.displacements);
        
        const position = result.position / bridgeAnalysis.getFallbackBridgeInfo(currentBridgeId).span_length;
        bridgeView.setVehiclePosition(position);
        
        if (document.getElementById('showDeformation').checked) {
            bridgeView.applyDeformation();
        }
        
        if (document.getElementById('showForces').checked) {
            bridgeView.updateMemberColors();
        }
        
        bridgeView.currentAnimationFrame++;
        if (bridgeView.currentAnimationFrame >= bridgeView.movingResults.length) {
            bridgeView.currentAnimationFrame = 0;
        }
    }, interval);
}

function stopAnimation() {
    bridgeView.isPlaying = false;
    if (bridgeView.animationInterval) {
        clearInterval(bridgeView.animationInterval);
        bridgeView.animationInterval = null;
    }
}

async function runCraftAnalysis() {
    if (window.craftPanel) {
        const result = await window.craftPanel.runAnalysis(currentBridgeId);
        switchTab('compare');
        return result;
    }
    return null;
}

function updateForcesTable(forces) {
    const tbody = document.getElementById('forcesTableBody');
    tbody.innerHTML = '';
    
    forces.slice(0, 20).forEach(f => {
        const stressClass = f.stress_ratio > 1 ? 'stress-danger' : 
                            f.stress_ratio > 0.8 ? 'stress-warning' : 'stress-safe';
        
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>M${f.member_id}</td>
            <td>${f.member_type || '-'}</td>
            <td>${f.axial_force.toFixed(2)}</td>
            <td>${f.shear_force.toFixed(2)}</td>
            <td>${f.bending_moment.toFixed(2)}</td>
            <td>${f.combined_stress.toFixed(3)}</td>
            <td class="${stressClass}">${f.stress_ratio.toFixed(3)}</td>
        `;
        tbody.appendChild(row);
    });
}

function updateDisplacementsTable(displacements) {
    const tbody = document.getElementById('displacementsTableBody');
    tbody.innerHTML = '';
    
    displacements.slice(0, 15).forEach(d => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>N${d.node_id}</td>
            <td>${d.displacement_x.toFixed(3)}</td>
            <td>${d.displacement_y.toFixed(3)}</td>
            <td>${d.total_displacement.toFixed(3)}</td>
        `;
        tbody.appendChild(row);
    });
}

function updateCompareTable(comparisons) {
    const tbody = document.getElementById('compareTableBody');
    tbody.innerHTML = '';
    
    if (!comparisons || comparisons.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;padding:20px;color:#999;">暂无对比数据</td></tr>';
        return;
    }
    
    comparisons.forEach(c => {
        const stressClass = c.stress_ratio > 1 ? 'stress-danger' : 
                            c.stress_ratio > 0.8 ? 'stress-warning' : 'stress-safe';
        const compliantClass = c.compliant ? 'compliant-yes' : 'compliant-no';
        const compliantText = c.compliant ? '合规 ✓' : '超限 ✗';
        
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>M${c.member_id}</td>
            <td>${c.member_type || '-'}</td>
            <td>${c.actual_stress.toFixed(2)} MPa</td>
            <td>${c.allowable_stress.toFixed(2)} MPa</td>
            <td class="${stressClass}">${c.stress_ratio.toFixed(3)}</td>
            <td>${c.actual_span_ratio ? c.actual_span_ratio.toFixed(2)}</td>
            <td class="${compliantClass}">${compliantText}</td>
        `;
        tbody.appendChild(row);
    });
}

async function loadSensorData() {
    const sensors = await bridgeAnalysis.getSensors(currentBridgeId);
    const list = document.getElementById('sensorList');
    list.innerHTML = '';
    
    sensors.slice(0, 15).forEach(sensor => {
        const item = document.createElement('div');
        item.className = 'sensor-item';
        item.innerHTML = `
            <span class="sensor-code">${sensor.sensor_code}</span>
            <span class="sensor-value">${sensor.current_value ? sensor.current_value.toFixed(2)} ${sensor.unit}</span>
            <div class="sensor-type">${sensor.measurement_type || sensor.sensor_type}</div>
        `;
        list.appendChild(item);
    });
}

async function loadAlertData() {
    const alerts = await bridgeAnalysis.getAlerts(currentBridgeId);
    const list = document.getElementById('alertList');
    list.innerHTML = '';
    
    if (alerts.length === 0) {
        list.innerHTML = '<div style="color:#999;text-align:center;padding:20px;">暂无告警</div>';
        return;
    }
    
    alerts.forEach(alert => {
        const item = document.createElement('div');
        item.className = `alert-item ${alert.alert_level}`;
        
        const time = new Date(alert.timestamp);
        const timeStr = time.toLocaleString('zh-CN');
        
        item.innerHTML = `
            <div class="alert-title">${alert.alert_message}</div>
            <div class="alert-time">${timeStr}</div>
            <div style="font-size:11px;color:#666;margin-top:4px;">
                测量值: ${alert.measured_value} / 阈值: ${alert.threshold_value}
            </div>
        `;
        list.appendChild(item);
    });
}

window.addEventListener('resize', function() {
    if (bridgeView) {
        bridgeView.onResize();
    }
});
