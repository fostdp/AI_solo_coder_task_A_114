class Bridge3DView {
    constructor(canvasId) {
        this.canvas = document.getElementById(canvasId);
        this.container = this.canvas.parentElement;
        
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.controls = null;
        
        this.bridgeGroup = null;
        this.members = [];
        this.nodes = [];
        this.deck = null;
        
        this.memberForces = [];
        this.nodeDisplacements = [];
        this.showForces = false;
        this.showDeformation = false;
        this.deformationScale = 10;
        
        this.stressMin = -10;
        this.stressMax = 10;
        
        this.bridgeData = null;
        this.vehicle = null;
        this.vehiclePosition = 0;
        
        this.init();
    }
    
    init() {
        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0xe8f4f8);
        this.scene.fog = new THREE.Fog(0xe8f4f8, 50, 200);
        
        this.setupCamera();
        this.setupRenderer();
        this.setupLights();
        this.setupGround();
        this.setupControls();
        
        window.addEventListener('resize', () => this.onResize());
        
        this.animate();
    }
    
    setupCamera() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;
        
        this.camera = new THREE.PerspectiveCamera(
            50,
            width / height,
            0.1,
            1000
        );
        
        this.camera.position.set(30, 20, 30);
    }
    
    setupRenderer() {
        this.renderer = new THREE.WebGLRenderer({
            canvas: this.canvas,
            antialias: true
        });
        
        this.renderer.setSize(
            this.container.clientWidth,
            this.container.clientHeight
        );
        
        this.renderer.setPixelRatio(window.devicePixelRatio);
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    }
    
    setupLights() {
        const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
        this.scene.add(ambientLight);
        
        const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
        directionalLight.position.set(50, 80, 50);
        directionalLight.castShadow = true;
        directionalLight.shadow.mapSize.width = 2048;
        directionalLight.shadow.mapSize.height = 2048;
        directionalLight.shadow.camera.near = 0.5;
        directionalLight.shadow.camera.far = 500;
        directionalLight.shadow.camera.left = -50;
        directionalLight.shadow.camera.right = 50;
        directionalLight.shadow.camera.top = 50;
        directionalLight.shadow.camera.bottom = -50;
        this.scene.add(directionalLight);
        
        const fillLight = new THREE.DirectionalLight(0x88aaff, 0.3);
        fillLight.position.set(-30, 20, -30);
        this.scene.add(fillLight);
    }
    
    setupGround() {
        const groundGeometry = new THREE.PlaneGeometry(200, 200);
        const groundMaterial = new THREE.MeshStandardMaterial({
            color: 0xc8dcc8,
            roughness: 0.8,
            metalness: 0.1
        });
        
        const ground = new THREE.Mesh(groundGeometry, groundMaterial);
        ground.rotation.x = -Math.PI / 2;
        ground.position.y = -0.1;
        ground.receiveShadow = true;
        this.scene.add(ground);
        
        const gridHelper = new THREE.GridHelper(100, 50, 0x888888, 0xcccccc);
        gridHelper.position.y = 0.01;
        this.scene.add(gridHelper);
    }
    
    setupControls() {
        this.controls = new THREE.OrbitControls(
            this.camera,
            this.renderer.domElement
        );
        
        this.controls.enableDamping = true;
        this.controls.dampingFactor = 0.05;
        this.controls.minDistance = 5;
        this.controls.maxDistance = 100;
        this.controls.maxPolarAngle = Math.PI / 2.1;
    }
    
    loadBridge(bridgeData) {
        this.bridgeData = bridgeData;
        this.clearBridge();
        
        this.bridgeGroup = new THREE.Group();
        
        this.createArchMembers(bridgeData);
        this.createDeck(bridgeData);
        this.createNodes(bridgeData);
        this.createAbutments(bridgeData);
        
        this.bridgeGroup.position.x = -bridgeData.span_length / 2;
        
        this.scene.add(this.bridgeGroup);
        
        this.updateStatsDisplay();
    }
    
    clearBridge() {
        if (this.bridgeGroup) {
            this.scene.remove(this.bridgeGroup);
            this.bridgeGroup.traverse((child) => {
                if (child.geometry) child.geometry.dispose();
                if (child.material) {
                    if (Array.isArray(child.material)) {
                        child.material.forEach(m => m.dispose());
                    } else {
                        child.material.dispose();
                    }
                }
            });
            this.bridgeGroup = null;
        }
        this.members = [];
        this.nodes = [];
    }
    
    createArchMembers(bridgeData) {
        const span = bridgeData.span_length;
        const rise = bridgeData.arch_rise;
        const width = bridgeData.deck_width;
        
        const numSegments = 10;
        const radius = (span * span / 4 + rise * rise) / (2 * rise);
        const centerY = rise - radius;
        const centerX = span / 2;
        
        const woodColor = 0x8B4513;
        const woodMaterial = new THREE.MeshStandardMaterial({
            color: woodColor,
            roughness: 0.7,
            metalness: 0.1
        });
        
        const archRadius = 0.25;
        
        for (let side = -1; side <= 1; side += 2) {
            const zOffset = side * (width / 2 - 0.5);
            
            for (let i = 0; i < numSegments; i++) {
                const angle1 = -Math.PI / 2 + Math.PI * i / numSegments;
                const angle2 = -Math.PI / 2 + Math.PI * (i + 1) / numSegments;
                
                const x1 = centerX + radius * Math.cos(angle1);
                const y1 = centerY + radius * Math.sin(angle1);
                const x2 = centerX + radius * Math.cos(angle2);
                const y2 = centerY + radius * Math.sin(angle2);
                
                const member = this.createBeam(
                    x1, y1, zOffset,
                    x2, y2, zOffset,
                    archRadius,
                    woodMaterial,
                    'arch_rib'
                );
                
                member.userData.memberId = i + 1 + (side > 0 ? numSegments : 0);
                member.userData.memberType = 'arch_rib';
                
                this.members.push(member);
                this.bridgeGroup.add(member);
            }
        }
        
        const tieMaterial = new THREE.MeshStandardMaterial({
            color: 0xA0522D,
            roughness: 0.7,
            metalness: 0.1
        });
        
        const numTies = 5;
        for (let i = 1; i < numTies; i++) {
            const angle = -Math.PI / 2 + Math.PI * i / numTies;
            const x = centerX + radius * Math.cos(angle);
            const y = centerY + radius * Math.sin(angle);
            
            const tie = this.createBeam(
                x, y, -width / 2 + 0.5,
                x, y, width / 2 - 0.5,
                0.15,
                tieMaterial,
                'transverse_tie'
            );
            
            tie.userData.memberId = 2 * numSegments + i;
            tie.userData.memberType = 'transverse_tie';
            
            this.members.push(tie);
            this.bridgeGroup.add(tie);
        }
        
        const deckY = 0;
        const numPosts = numSegments + 1;
        const postMaterial = new THREE.MeshStandardMaterial({
            color: 0xCD853F,
            roughness: 0.7,
            metalness: 0.1
        });
        
        for (let i = 0; i <= numSegments; i++) {
            const angle = -Math.PI / 2 + Math.PI * i / numSegments;
            const archX = centerX + radius * Math.cos(angle);
            const archY = centerY + radius * Math.sin(angle);
            
            for (let side = -1; side <= 1; side += 2) {
                const zOffset = side * (width / 2 - 0.5);
                
                const post = this.createBeam(
                    archX, archY, zOffset,
                    archX, deckY, zOffset,
                    0.18,
                    postMaterial,
                    'vertical_post'
                );
                
                post.userData.memberId = 2 * numSegments + numTies + i * 2 + (side > 0 ? 1 : 0);
                post.userData.memberType = 'vertical_post';
                
                this.members.push(post);
                this.bridgeGroup.add(post);
            }
        }
        
        const numDiagonals = numSegments;
        const diagMaterial = new THREE.MeshStandardMaterial({
            color: 0xD2691E,
            roughness: 0.7,
            metalness: 0.1
        });
        
        for (let i = 0; i < numDiagonals; i++) {
            const angle1 = -Math.PI / 2 + Math.PI * i / numSegments;
            const angle2 = -Math.PI / 2 + Math.PI * (i + 1) / numSegments;
            
            const x1 = centerX + radius * Math.cos(angle1);
            const y1 = centerY + radius * Math.sin(angle1);
            const x2 = centerX + radius * Math.cos(angle2);
            const y2 = deckY;
            
            for (let side = -1; side <= 1; side += 2) {
                const zOffset = side * (width / 2 - 0.5);
                
                const diag = this.createBeam(
                    x1, y1, zOffset,
                    x2, y2, zOffset,
                    0.12,
                    diagMaterial,
                    'diagonal_brace'
                );
                
                diag.userData.memberId = 2 * numSegments + numTies + (numSegments + 1) * 2 + i * 2 + (side > 0 ? 1 : 0);
                diag.userData.memberType = 'diagonal_brace';
                
                this.members.push(diag);
                this.bridgeGroup.add(diag);
            }
        }
    }
    
    createBeam(x1, y1, z1, x2, y2, z2, radius, material, type) {
        const dx = x2 - x1;
        const dy = y2 - y1;
        const dz = z2 - z1;
        const length = Math.sqrt(dx * dx + dy * dy + dz * dz);
        
        const geometry = new THREE.CylinderGeometry(radius, radius, length, 8);
        geometry.translate(0, length / 2, 0);
        
        const beam = new THREE.Mesh(geometry, material.clone());
        beam.castShadow = true;
        beam.receiveShadow = true;
        
        beam.position.set(x1, y1, z1);
        
        const direction = new THREE.Vector3(dx, dy, dz).normalize();
        const up = new THREE.Vector3(0, 1, 0);
        const axis = new THREE.Vector3().crossVectors(up, direction).normalize();
        const angle = Math.acos(direction.dot(up));
        
        if (axis.length() > 0.001) {
            beam.setRotationFromAxisAngle(axis, angle);
        }
        
        return beam;
    }
    
    createDeck(bridgeData) {
        const span = bridgeData.span_length;
        const width = bridgeData.deck_width;
        const thickness = 0.3;
        const y = 0;
        
        const deckMaterial = new THREE.MeshStandardMaterial({
            color: 0x8B7355,
            roughness: 0.8,
            metalness: 0.05
        });
        
        const deckGeometry = new THREE.BoxGeometry(span, thickness, width);
        this.deck = new THREE.Mesh(deckGeometry, deckMaterial);
        this.deck.position.set(span / 2, y - thickness / 2, 0);
        this.deck.receiveShadow = true;
        this.deck.castShadow = true;
        
        this.bridgeGroup.add(this.deck);
        
        const plankMaterial = new THREE.MeshStandardMaterial({
            color: 0x9B8B7B,
            roughness: 0.8
        });
        
        const numPlanks = Math.floor(width / 0.3);
        const plankWidth = width / numPlanks;
        
        for (let i = 0; i < numPlanks; i++) {
            const plankGeo = new THREE.BoxGeometry(span, 0.05, plankWidth - 0.02);
            const plank = new THREE.Mesh(plankGeo, plankMaterial);
            plank.position.set(
                span / 2,
                y + 0.025,
                -width / 2 + plankWidth / 2 + i * plankWidth
            );
            plank.receiveShadow = true;
            this.bridgeGroup.add(plank);
        }
        
        const railMaterial = new THREE.MeshStandardMaterial({
            color: 0x654321,
            roughness: 0.7
        });
        
        const railHeight = 1.0;
        const railRadius = 0.06;
        
        for (let side = -1; side <= 1; side += 2) {
            const railZ = side * (width / 2 - 0.1);
            
            const topRail = this.createBeam(
                0, railHeight, railZ,
                span, railHeight, railZ,
                railRadius,
                railMaterial,
                'railing'
            );
            this.bridgeGroup.add(topRail);
            
            const midRail = this.createBeam(
                0, railHeight * 0.5, railZ,
                span, railHeight * 0.5, railZ,
                railRadius * 0.8,
                railMaterial,
                'railing'
            );
            this.bridgeGroup.add(midRail);
            
            const numPosts = 12;
            for (let i = 0; i <= numPosts; i++) {
                const x = (span / numPosts) * i;
                const post = this.createBeam(
                    x, 0, railZ,
                    x, railHeight, railZ,
                    railRadius * 0.7,
                    railMaterial,
                    'railing_post'
                );
                this.bridgeGroup.add(post);
            }
        }
    }
    
    createNodes(bridgeData) {
        const span = bridgeData.span_length;
        const rise = bridgeData.arch_rise;
        const width = bridgeData.deck_width;
        
        const numSegments = 10;
        const radius = (span * span / 4 + rise * rise) / (2 * rise);
        const centerY = rise - radius;
        const centerX = span / 2;
        
        const nodeMaterial = new THREE.MeshStandardMaterial({
            color: 0xff6600,
            emissive: 0x331100,
            roughness: 0.5,
            metalness: 0.5
        });
        
        let nodeId = 1;
        
        for (let side = -1; side <= 1; side += 2) {
            const zOffset = side * (width / 2 - 0.5);
            
            for (let i = 0; i <= numSegments; i++) {
                const angle = -Math.PI / 2 + Math.PI * i / numSegments;
                const x = centerX + radius * Math.cos(angle);
                const y = centerY + radius * Math.sin(angle);
                
                const nodeGeo = new THREE.SphereGeometry(0.15, 16, 16);
                const node = new THREE.Mesh(nodeGeo, nodeMaterial.clone());
                node.position.set(x, y, zOffset);
                node.userData.nodeId = nodeId++;
                node.userData.originalPosition = { x, y, z: zOffset };
                
                this.nodes.push(node);
                this.bridgeGroup.add(node);
            }
        }
        
        for (let side = -1; side <= 1; side += 2) {
            const zOffset = side * (width / 2 - 0.5);
            
            for (let i = 0; i <= numSegments; i++) {
                const x = (span / numSegments) * i;
                const y = 0;
                
                const nodeGeo = new THREE.SphereGeometry(0.12, 16, 16);
                const node = new THREE.Mesh(nodeGeo, nodeMaterial.clone());
                node.position.set(x, y, zOffset);
                node.userData.nodeId = nodeId++;
                node.userData.originalPosition = { x, y, z: zOffset };
                
                this.nodes.push(node);
                this.bridgeGroup.add(node);
            }
        }
    }
    
    createAbutments(bridgeData) {
        const width = bridgeData.deck_width;
        
        const abutmentMaterial = new THREE.MeshStandardMaterial({
            color: 0x808080,
            roughness: 0.9,
            metalness: 0.1
        });
        
        for (let side = -1; side <= 1; side += 2) {
            const x = side > 0 ? bridgeData.span_length : 0;
            
            const abutmentGeo = new THREE.BoxGeometry(2, 3, width + 2);
            const abutment = new THREE.Mesh(abutmentGeo, abutmentMaterial);
            abutment.position.set(x + side * 1, -1.5, 0);
            abutment.receiveShadow = true;
            abutment.castShadow = true;
            this.bridgeGroup.add(abutment);
        }
    }
    
    setForces(memberForces) {
        this.memberForces = memberForces;
        if (this.showForces) {
            this.updateMemberColors();
        }
    }
    
    setDisplacements(displacements) {
        this.nodeDisplacements = displacements;
    }
    
    updateMemberColors() {
        if (!this.memberForces || this.memberForces.length === 0) {
            return;
        }
        
        const forceMap = {};
        this.memberForces.forEach(f => {
            forceMap[f.member_id || f.MemberID] = f;
        });
        
        let minStress = Infinity;
        let maxStress = -Infinity;
        
        Object.values(forceMap).forEach(f => {
            const stress = f.axial_stress !== undefined ? f.axial_stress : (f.AxialStress || 0);
            if (stress < minStress) minStress = stress;
            if (stress > maxStress) maxStress = stress;
        });
        
        this.stressMin = minStress;
        this.stressMax = maxStress;
        
        const absMax = Math.max(Math.abs(minStress), Math.abs(maxStress));
        if (absMax === 0) return;
        
        this.members.forEach(member => {
            const memberId = member.userData.memberId;
            const force = forceMap[memberId];
            
            if (force) {
                const stress = force.axial_stress !== undefined ? force.axial_stress : (force.AxialStress || 0);
                const color = this.getStressColor(stress, -absMax, absMax);
                member.material.color.setHex(color);
                member.material.emissive = new THREE.Color(color).multiplyScalar(0.2);
            }
        });
        
        this.updateLegend(absMax);
    }
    
    getStressColor(stress, minStress, maxStress) {
        const range = maxStress - minStress;
        if (range === 0) return 0x8B4513;
        
        const t = (stress - minStress) / range;
        
        if (t < 0.25) {
            return this.interpolateColor(0x2980b9, 0x27ae60, t / 0.25);
        } else if (t < 0.5) {
            return this.interpolateColor(0x27ae60, 0xf1c40f, (t - 0.25) / 0.25);
        } else if (t < 0.75) {
            return this.interpolateColor(0xf1c40f, 0xe67e22, (t - 0.5) / 0.25);
        } else {
            return this.interpolateColor(0xe67e22, 0xe74c3c, (t - 0.75) / 0.25);
        }
    }
    
    interpolateColor(c1, c2, t) {
        const r1 = (c1 >> 16) & 0xff;
        const g1 = (c1 >> 8) & 0xff;
        const b1 = c1 & 0xff;
        
        const r2 = (c2 >> 16) & 0xff;
        const g2 = (c2 >> 8) & 0xff;
        const b2 = c2 & 0xff;
        
        const r = Math.round(r1 + (r2 - r1) * t);
        const g = Math.round(g1 + (g2 - g1) * t);
        const b = Math.round(b1 + (b2 - b1) * t);
        
        return (r << 16) | (g << 8) | b;
    }
    
    updateLegend(maxStress) {
        const legendPanel = document.getElementById('legendPanel');
        if (legendPanel) {
            document.getElementById('stressMin').textContent = (-maxStress).toFixed(1) + ' MPa';
            document.getElementById('stressMax').textContent = maxStress.toFixed(1) + ' MPa';
        }
    }
    
    applyDeformation() {
        if (!this.showDeformation || !this.nodeDisplacements) {
            this.resetDeformation();
            return;
        }
        
        const dispMap = {};
        this.nodeDisplacements.forEach(d => {
            dispMap[d.node_id || d.NodeID] = d;
        });
        
        this.nodes.forEach(node => {
            const nodeId = node.userData.nodeId;
            const disp = dispMap[nodeId];
            
            if (disp && node.userData.originalPosition) {
                const orig = node.userData.originalPosition;
                const dx = (disp.displacement_x !== undefined ? disp.displacement_x : (disp.UX || 0)) / 1000 * this.deformationScale;
                const dy = (disp.displacement_y !== undefined ? disp.displacement_y : (disp.UY || 0)) / 1000 * this.deformationScale;
                
                node.position.x = orig.x + dx;
                node.position.y = orig.y + dy;
            }
        });
    }
    
    resetDeformation() {
        this.nodes.forEach(node => {
            if (node.userData.originalPosition) {
                const orig = node.userData.originalPosition;
                node.position.x = orig.x;
                node.position.y = orig.y;
                node.position.z = orig.z;
            }
        });
    }
    
    toggleMembers(show) {
        this.members.forEach(m => {
            if (m.userData.memberType === 'arch_rib' || 
                m.userData.memberType === 'vertical_post' ||
                m.userData.memberType === 'diagonal_brace' ||
                m.userData.memberType === 'transverse_tie') {
                m.visible = show;
            }
        });
    }
    
    toggleDeck(show) {
        if (this.deck) {
            this.deck.visible = show;
        }
    }
    
    toggleNodes(show) {
        this.nodes.forEach(n => {
            n.visible = show;
        });
    }
    
    toggleForces(show) {
        this.showForces = show;
        if (show) {
            this.updateMemberColors();
        } else {
            this.resetMemberColors();
        }
    }
    
    toggleDeformation(show) {
        this.showDeformation = show;
        if (show) {
            this.applyDeformation();
        } else {
            this.resetDeformation();
        }
    }
    
    setDeformationScale(scale) {
        this.deformationScale = scale;
        if (this.showDeformation) {
            this.applyDeformation();
        }
    }
    
    resetMemberColors() {
        this.members.forEach(member => {
            let color = 0x8B4513;
            if (member.userData.memberType === 'vertical_post') {
                color = 0xCD853F;
            } else if (member.userData.memberType === 'diagonal_brace') {
                color = 0xD2691E;
            } else if (member.userData.memberType === 'transverse_tie') {
                color = 0xA0522D;
            }
            
            member.material.color.setHex(color);
            member.material.emissive.setHex(0x000000);
        });
    }
    
    updateStatsDisplay() {
        if (this.bridgeData) {
            document.getElementById('memberCount').textContent = this.members.length;
            document.getElementById('nodeCount').textContent = this.nodes.length;
        }
    }
    
    setTopView() {
        if (!this.bridgeData) return;
        const span = this.bridgeData.span_length;
        this.camera.position.set(span / 2, 50, 0);
        this.controls.target.set(span / 2, 0, 0);
    }
    
    setSideView() {
        if (!this.bridgeData) return;
        const span = this.bridgeData.span_length;
        this.camera.position.set(span / 2, span * 0.3, span);
        this.controls.target.set(span / 2, span * 0.15, 0);
    }
    
    setFrontView() {
        if (!this.bridgeData) return;
        const span = this.bridgeData.span_length;
        this.camera.position.set(-span, span * 0.3, 0);
        this.controls.target.set(span / 2, span * 0.15, 0);
    }
    
    setIsoView() {
        if (!this.bridgeData) return;
        const span = this.bridgeData.span_length;
        this.camera.position.set(span * 0.8, span * 0.6, span * 0.6);
        this.controls.target.set(span / 2, span * 0.1, 0);
    }
    
    createVehicle(vehicleType) {
        if (this.vehicle) {
            this.bridgeGroup.remove(this.vehicle);
        }
        
        const vehicleGroup = new THREE.Group();
        
        const bodyMaterial = new THREE.MeshStandardMaterial({
            color: 0x8B0000,
            roughness: 0.6,
            metalness: 0.3
        });
        
        const wheelMaterial = new THREE.MeshStandardMaterial({
            color: 0x333333,
            roughness: 0.9
        });
        
        const bodyLength = 4;
        const bodyWidth = 2;
        const bodyHeight = 1.5;
        
        const bodyGeo = new THREE.BoxGeometry(bodyLength, bodyHeight, bodyWidth);
        const body = new THREE.Mesh(bodyGeo, bodyMaterial);
        body.position.y = 1.2;
        body.castShadow = true;
        vehicleGroup.add(body);
        
        const wheelRadius = 0.5;
        const wheelWidth = 0.3;
        
        const wheelPositions = [
            { x: -1.2, z: -1 },
            { x: -1.2, z: 1 },
            { x: 1.2, z: -1 },
            { x: 1.2, z: 1 }
        ];
        
        wheelPositions.forEach(pos => {
            const wheelGeo = new THREE.CylinderGeometry(wheelRadius, wheelRadius, wheelWidth, 16);
            wheelGeo.rotateX(Math.PI / 2);
            const wheel = new THREE.Mesh(wheelGeo, wheelMaterial);
            wheel.position.set(pos.x, wheelRadius, pos.z);
            wheel.castShadow = true;
            vehicleGroup.add(wheel);
        });
        
        const roofMaterial = new THREE.MeshStandardMaterial({
            color: 0x654321,
            roughness: 0.7
        });
        
        const roofGeo = new THREE.ConeGeometry(1.5, 1.2, 4);
        const roof = new THREE.Mesh(roofGeo, roofMaterial);
        roof.position.y = 2.5;
        roof.rotation.y = Math.PI / 4;
        roof.castShadow = true;
        vehicleGroup.add(roof);
        
        this.vehicle = vehicleGroup;
        this.bridgeGroup.add(vehicleGroup);
        
        this.setVehiclePosition(0);
    }
    
    setVehiclePosition(position) {
        if (!this.vehicle || !this.bridgeData) return;
        
        this.vehiclePosition = position;
        const span = this.bridgeData.span_length;
        const x = position * span;
        
        const rise = this.bridgeData.arch_rise;
        const radius = (span * span / 4 + rise * rise) / (2 * rise);
        const centerY = rise - radius;
        const centerX = span / 2;
        
        const dx = x - centerX;
        const y = centerY + Math.sqrt(radius * radius - dx * dx);
        
        this.vehicle.position.set(x, y + 0.5, 0);
    }
    
    removeVehicle() {
        if (this.vehicle) {
            this.bridgeGroup.remove(this.vehicle);
            this.vehicle = null;
        }
    }
    
    onResize() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;
        
        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
        
        this.renderer.setSize(width, height);
    }
    
    animate() {
        requestAnimationFrame(() => this.animate());
        
        this.controls.update();
        this.renderer.render(this.scene, this.camera);
    }
}
