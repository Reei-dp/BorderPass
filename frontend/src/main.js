import './style.css';
import { GetConfig, SaveConfig, GetLocalIP, StartService, StopService, GetTelemetry } from '../wailsjs/go/main/App';

const state = {
    screen: 1,
    role: 'server',
    position: 'left',
    address: ':65432',
    ip: '127.0.0.1',
    port: '65432',
    lanIP: '127.0.0.1',
    isRunning: false,
    telemetry: {
        isRunning: false,
        role: 'server',
        status: 'OFFLINE',
        remoteAddr: '',
        activeControl: 'HOST (LOCAL)',
        packetsSent: 0,
        packetsRecv: 0
    }
};

const appEl = document.querySelector('#app');

// Initialize state from Go backend
async function init() {
    try {
        const [cfg, ip] = await Promise.all([GetConfig(), GetLocalIP()]);
        if (ip) {
            state.lanIP = ip;
        }

        if (cfg) {
            state.role = cfg.role || 'server';
            state.position = cfg.position || 'left';
            state.address = cfg.address || ':65432';

            if (state.address.includes(':')) {
                const parts = state.address.split(':');
                state.port = parts[1] || '65432';
                if (state.role === 'client') {
                    state.ip = parts[0] || state.lanIP;
                }
            }
        }

        if (state.role === 'server') {
            state.ip = state.lanIP;
        } else if (!state.ip || state.ip === '0.0.0.0' || state.ip === '127.0.0.1') {
            state.ip = state.lanIP;
        }
    } catch (e) {
        console.error('Init error:', e);
    }
    render();
}

function render() {
    if (state.screen === 1) {
        renderScreen1();
    } else if (state.screen === 2) {
        renderScreen2();
    } else {
        if (state.role === 'server') {
            renderHostScreen();
        } else {
            renderClientScreen();
        }
    }
}

const LOGO_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 540 92" class="borderpass-logo">
  <defs>
    <filter id="neon-glow" x="-15%" y="-15%" width="130%" height="130%">
      <feGaussianBlur stdDeviation="2.5" result="blur"/>
      <feMerge>
        <feMergeNode in="blur"/>
        <feMergeNode in="SourceGraphic"/>
      </feMerge>
    </filter>
  </defs>
  <rect x="2" y="2" width="536" height="88" rx="5" fill="#121820" stroke="#00e5ff" stroke-width="1.5"/>
  <rect x="6" y="6" width="528" height="80" rx="3" fill="none" stroke="#00e5ff" stroke-width="1.2" stroke-opacity="0.8"/>
  <g fill="#00e5ff" filter="url(#neon-glow)">
<rect x="17.0" y="22.0" width="9.5" height="16.0"/>
<rect x="26.5" y="22.0" width="9.5" height="8.0"/>
<rect x="36.0" y="22.0" width="9.5" height="8.0"/>
<rect x="45.5" y="30.0" width="9.5" height="8.0"/>
<rect x="17.0" y="38.0" width="9.5" height="16.0"/>
<rect x="26.5" y="38.0" width="9.5" height="8.0"/>
<rect x="36.0" y="38.0" width="9.5" height="8.0"/>
<rect x="45.5" y="46.0" width="9.5" height="8.0"/>
<rect x="17.0" y="54.0" width="9.5" height="8.0"/>
<rect x="26.5" y="54.0" width="9.5" height="8.0"/>
<rect x="36.0" y="54.0" width="9.5" height="8.0"/>
<rect x="69.0" y="22.0" width="9.5" height="16.0"/>
<rect x="78.5" y="22.0" width="9.5" height="8.0"/>
<rect x="88.0" y="22.0" width="9.5" height="8.0"/>
<rect x="97.5" y="22.0" width="9.5" height="16.0"/>
<rect x="69.0" y="38.0" width="9.5" height="16.0"/>
<rect x="97.5" y="38.0" width="9.5" height="16.0"/>
<rect x="69.0" y="54.0" width="9.5" height="8.0"/>
<rect x="78.5" y="54.0" width="9.5" height="8.0"/>
<rect x="88.0" y="54.0" width="9.5" height="8.0"/>
<rect x="97.5" y="54.0" width="9.5" height="8.0"/>
<rect x="121.0" y="22.0" width="9.5" height="16.0"/>
<rect x="130.5" y="22.0" width="9.5" height="8.0"/>
<rect x="140.0" y="22.0" width="9.5" height="8.0"/>
<rect x="149.5" y="22.0" width="9.5" height="16.0"/>
<rect x="121.0" y="38.0" width="9.5" height="16.0"/>
<rect x="130.5" y="46.0" width="9.5" height="8.0"/>
<rect x="140.0" y="46.0" width="9.5" height="8.0"/>
<rect x="149.5" y="38.0" width="9.5" height="8.0"/>
<rect x="121.0" y="54.0" width="9.5" height="8.0"/>
<rect x="149.5" y="54.0" width="9.5" height="8.0"/>
<rect x="173.0" y="22.0" width="9.5" height="16.0"/>
<rect x="182.5" y="22.0" width="9.5" height="8.0"/>
<rect x="192.0" y="22.0" width="9.5" height="8.0"/>
<rect x="201.5" y="30.0" width="9.5" height="8.0"/>
<rect x="173.0" y="38.0" width="9.5" height="16.0"/>
<rect x="201.5" y="38.0" width="9.5" height="16.0"/>
<rect x="173.0" y="54.0" width="9.5" height="8.0"/>
<rect x="182.5" y="54.0" width="9.5" height="8.0"/>
<rect x="192.0" y="54.0" width="9.5" height="8.0"/>
<rect x="225.0" y="22.0" width="9.5" height="16.0"/>
<rect x="234.5" y="22.0" width="9.5" height="8.0"/>
<rect x="244.0" y="22.0" width="9.5" height="8.0"/>
<rect x="253.5" y="22.0" width="9.5" height="8.0"/>
<rect x="225.0" y="38.0" width="9.5" height="16.0"/>
<rect x="234.5" y="38.0" width="9.5" height="8.0"/>
<rect x="244.0" y="38.0" width="9.5" height="8.0"/>
<rect x="225.0" y="54.0" width="9.5" height="8.0"/>
<rect x="234.5" y="54.0" width="9.5" height="8.0"/>
<rect x="244.0" y="54.0" width="9.5" height="8.0"/>
<rect x="253.5" y="54.0" width="9.5" height="8.0"/>
<rect x="277.0" y="22.0" width="9.5" height="16.0"/>
<rect x="286.5" y="22.0" width="9.5" height="8.0"/>
<rect x="296.0" y="22.0" width="9.5" height="8.0"/>
<rect x="305.5" y="22.0" width="9.5" height="16.0"/>
<rect x="277.0" y="38.0" width="9.5" height="16.0"/>
<rect x="286.5" y="46.0" width="9.5" height="8.0"/>
<rect x="296.0" y="46.0" width="9.5" height="8.0"/>
<rect x="305.5" y="38.0" width="9.5" height="8.0"/>
<rect x="277.0" y="54.0" width="9.5" height="8.0"/>
<rect x="305.5" y="54.0" width="9.5" height="8.0"/>
<rect x="329.0" y="22.0" width="9.5" height="16.0"/>
<rect x="338.5" y="22.0" width="9.5" height="8.0"/>
<rect x="348.0" y="22.0" width="9.5" height="8.0"/>
<rect x="357.5" y="22.0" width="9.5" height="16.0"/>
<rect x="329.0" y="38.0" width="9.5" height="16.0"/>
<rect x="338.5" y="46.0" width="9.5" height="8.0"/>
<rect x="348.0" y="46.0" width="9.5" height="8.0"/>
<rect x="357.5" y="38.0" width="9.5" height="16.0"/>
<rect x="329.0" y="54.0" width="9.5" height="8.0"/>
<rect x="381.0" y="22.0" width="9.5" height="16.0"/>
<rect x="390.5" y="22.0" width="9.5" height="8.0"/>
<rect x="400.0" y="22.0" width="9.5" height="8.0"/>
<rect x="409.5" y="22.0" width="9.5" height="16.0"/>
<rect x="381.0" y="38.0" width="9.5" height="16.0"/>
<rect x="390.5" y="46.0" width="9.5" height="8.0"/>
<rect x="400.0" y="46.0" width="9.5" height="8.0"/>
<rect x="409.5" y="38.0" width="9.5" height="16.0"/>
<rect x="381.0" y="54.0" width="9.5" height="8.0"/>
<rect x="409.5" y="54.0" width="9.5" height="8.0"/>
<rect x="433.0" y="22.0" width="9.5" height="16.0"/>
<rect x="442.5" y="22.0" width="9.5" height="8.0"/>
<rect x="452.0" y="22.0" width="9.5" height="8.0"/>
<rect x="461.5" y="22.0" width="9.5" height="8.0"/>
<rect x="433.0" y="38.0" width="9.5" height="8.0"/>
<rect x="442.5" y="38.0" width="9.5" height="8.0"/>
<rect x="452.0" y="38.0" width="9.5" height="8.0"/>
<rect x="461.5" y="38.0" width="9.5" height="16.0"/>
<rect x="433.0" y="54.0" width="9.5" height="8.0"/>
<rect x="442.5" y="54.0" width="9.5" height="8.0"/>
<rect x="452.0" y="54.0" width="9.5" height="8.0"/>
<rect x="485.0" y="22.0" width="9.5" height="16.0"/>
<rect x="494.5" y="22.0" width="9.5" height="8.0"/>
<rect x="504.0" y="22.0" width="9.5" height="8.0"/>
<rect x="513.5" y="22.0" width="9.5" height="8.0"/>
<rect x="485.0" y="38.0" width="9.5" height="8.0"/>
<rect x="494.5" y="38.0" width="9.5" height="8.0"/>
<rect x="504.0" y="38.0" width="9.5" height="8.0"/>
<rect x="513.5" y="38.0" width="9.5" height="16.0"/>
<rect x="485.0" y="54.0" width="9.5" height="8.0"/>
<rect x="494.5" y="54.0" width="9.5" height="8.0"/>
<rect x="504.0" y="54.0" width="9.5" height="8.0"/>
  </g>
</svg>`;

// -----------------------------------------------------------------------------
// Screen 1: Splash / Vector Title + Full-width Continue
// -----------------------------------------------------------------------------
function renderScreen1() {
    appEl.innerHTML = `
        <div class="splash-container">
            ${LOGO_SVG}
            <button class="ascii-btn" id="continueBtn">[ CONTINUE ]</button>
        </div>
    `;

    document.getElementById('continueBtn').onclick = () => {
        state.screen = 2;
        render();
    };
}

// -----------------------------------------------------------------------------
// Screen 2: Choose Host or Second PC
// -----------------------------------------------------------------------------
function renderScreen2() {
    appEl.innerHTML = `
        <div class="role-container">
            <div class="role-card" id="chooseHost">
                <div class="role-title">HOST PC</div>
                <div class="role-desc">Shares Mouse & Keyboard<br>Primary Workstation</div>
                <div class="role-action-badge">[ CHOOSE HOST ]</div>
            </div>
            <div class="role-card" id="chooseSecond">
                <div class="role-title">SECOND PC</div>
                <div class="role-desc">Receives Remote Control<br>Secondary Computer</div>
                <div class="role-action-badge">[ CHOOSE SECOND PC ]</div>
            </div>
        </div>
    `;

    document.getElementById('chooseHost').onclick = async () => {
        state.role = 'server';
        state.ip = state.lanIP;
        state.port = '65432';
        state.address = `:${state.port}`;
        await SaveConfig(state.role, state.position, state.address);

        // Auto-start server for Host immediately!
        try {
            await StartService(state.role, state.address, state.position);
            state.isRunning = true;
        } catch (e) {
            console.error('Auto-start host error:', e);
        }

        state.screen = 3;
        render();
    };

    document.getElementById('chooseSecond').onclick = async () => {
        state.role = 'client';
        if (state.ip === '0.0.0.0' || state.ip === '127.0.0.1' || !state.ip) {
            state.ip = state.lanIP;
        }
        state.port = '65432';
        state.address = `${state.ip}:${state.port}`;
        await SaveConfig(state.role, state.position, state.address);
        state.screen = 3;
        render();
    };
}

// -----------------------------------------------------------------------------
// Screen 3 - HOST MODE: Monitors Arrangement + Network + Live Status Bar
// -----------------------------------------------------------------------------
function renderHostScreen() {
    appEl.innerHTML = `
        <div class="main-layout">
            <div class="top-bar">
                <button class="back-btn" id="backBtn">[ ◄ BACK ]</button>
                <div class="role-badge">HOST PC (Shares Input)</div>
            </div>

            <!-- Top Box: Monitors Arrangement Canvas -->
            <div class="panel">
                <div class="panel-header">MONITORS ARRANGEMENT</div>
                <div class="arrangement-canvas" id="canvas">
                    <div class="monitor-box host" id="hostBox">
                        <div class="monitor-name">HOST PC</div>
                        <div class="monitor-status">${state.lanIP}</div>
                    </div>
                    <div class="monitor-box draggable" id="secondBox">
                        <div class="monitor-name">SECOND PC</div>
                        <div class="monitor-status" id="secondStatus">
                            <span class="status-dot" id="secondDot"></span>
                            <span id="secondText">Waiting...</span>
                        </div>
                    </div>
                </div>
                <div class="transition-indicator" id="transitionText">
                    ${getTransitionText()}
                </div>
            </div>

            <!-- Bottom Grid: Left Network, Right Status -->
            <div class="bottom-grid">
                <!-- Bottom-Left Box: Network -->
                <div class="panel">
                    <div class="panel-header">NETWORK</div>
                    <div class="form-group">
                        <span class="form-label">IP:</span>
                        <input class="form-input" id="ipInput" type="text" value="${state.lanIP}" readonly />
                    </div>
                    <div class="form-group">
                        <span class="form-label">Port:</span>
                        <input class="form-input" id="portInput" type="text" value="${state.port}" />
                    </div>
                    <div class="helper-text">
                        Enter this IP and Port on your Second PC to connect.
                    </div>
                </div>

                <!-- Bottom-Right Box: Client Status -->
                <div class="panel">
                    <div class="panel-header">CLIENT STATUS</div>
                    <div class="status-list">
                        <div class="status-item">
                            <span class="status-key">Server</span>
                            <span class="status-val" id="serverState">${state.isRunning ? '● Listening' : '○ Stopped'}</span>
                        </div>
                        <div class="status-item">
                            <span class="status-key">Client</span>
                            <span class="status-val" id="clientState">○ Waiting for connection...</span>
                        </div>
                        <div class="status-item">
                            <span class="status-key">Control</span>
                            <span class="status-val" id="telemetryControl">HOST (LOCAL)</span>
                        </div>
                        <div class="status-item">
                            <span class="status-key">Packets</span>
                            <span class="status-val" id="telemetryPackets">0 sent</span>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Host Status Bar (No redundant Start button) -->
            <div class="host-status-bar">
                <div class="host-status-left">
                    <span class="status-dot ${state.isRunning ? 'active' : ''}" id="hostBarDot"></span>
                    <span id="hostBarText">${state.isRunning ? `HOST ACTIVE — LISTENING ON ${state.lanIP}:${state.port}` : 'HOST STOPPED'}</span>
                </div>
                <button class="restart-btn" id="restartBtn">↻ RESTART</button>
            </div>
        </div>
    `;

    document.getElementById('backBtn').onclick = async () => {
        if (state.isRunning) {
            await StopService();
            state.isRunning = false;
        }
        state.screen = 2;
        render();
    };

    const portInput = document.getElementById('portInput');
    portInput.oninput = () => {
        state.port = portInput.value.trim();
        updateAddress();
    };

    const restartBtn = document.getElementById('restartBtn');
    if (restartBtn) {
        restartBtn.onclick = async () => {
            restartBtn.innerText = '↻ RESTARTING...';
            restartBtn.disabled = true;
            try {
                updateAddress();
                await StartService(state.role, state.address, state.position);
                state.isRunning = true;
            } catch (e) {
                console.error('Restart error:', e);
            } finally {
                setTimeout(() => {
                    if (restartBtn) {
                        restartBtn.innerText = '↻ RESTART';
                        restartBtn.disabled = false;
                    }
                }, 500);
            }
        };
    }

    initCanvasArrangement();
}

// -----------------------------------------------------------------------------
// Screen 3 - SECOND PC (CLIENT) MODE: Only IP & Port Inputs + Connect Button!
// -----------------------------------------------------------------------------
function renderClientScreen() {
    appEl.innerHTML = `
        <div class="main-layout client-mode">
            <div class="top-bar">
                <button class="back-btn" id="backBtn">[ ◄ BACK ]</button>
                <div class="role-badge">SECOND PC (Remote Input)</div>
            </div>

            <!-- Connect Panel: ONLY IP and Port -->
            <div class="panel">
                <div class="panel-header">CONNECT TO HOST PC</div>
                <div class="form-group">
                    <span class="form-label">Host IP:</span>
                    <input class="form-input" id="ipInput" type="text" value="${state.ip}" placeholder="e.g. 192.168.1.17" autofocus />
                </div>
                <div class="form-group">
                    <span class="form-label">Port:</span>
                    <input class="form-input" id="portInput" type="text" value="${state.port}" placeholder="65432" />
                </div>
                <div class="helper-text">
                    Enter the LAN IP and Port displayed on your Host PC.
                </div>
            </div>

            <!-- Client Connection Status -->
            <div class="panel">
                <div class="panel-header">CONNECTION STATUS</div>
                <div class="status-list">
                    <div class="status-item">
                        <span class="status-key">Status</span>
                        <span class="status-val" id="clientState">${state.isRunning ? '◌ Connecting...' : '○ Disconnected'}</span>
                    </div>
                    <div class="status-item">
                        <span class="status-key">Host Target</span>
                        <span class="status-val" id="clientTarget">${state.ip}:${state.port}</span>
                    </div>
                    <div class="status-item">
                        <span class="status-key">Control Focus</span>
                        <span class="status-val" id="telemetryControl">HOST (LOCAL)</span>
                    </div>
                    <div class="status-item">
                        <span class="status-key">Packets Received</span>
                        <span class="status-val" id="telemetryPackets">0 recv</span>
                    </div>
                </div>
            </div>

            <!-- Connect / Disconnect Button -->
            <button class="start-btn ${state.isRunning ? 'running' : ''}" id="startBtn">
                ${state.isRunning ? '[ ⏹ DISCONNECT ]' : '[ ▶ CONNECT TO HOST ]'}
            </button>
        </div>
    `;

    document.getElementById('backBtn').onclick = async () => {
        if (state.isRunning) {
            await StopService();
            state.isRunning = false;
        }
        state.screen = 2;
        render();
    };

    const ipInput = document.getElementById('ipInput');
    const portInput = document.getElementById('portInput');

    ipInput.oninput = () => {
        state.ip = ipInput.value.trim();
        updateAddress();
        const targetEl = document.getElementById('clientTarget');
        if (targetEl) targetEl.innerText = `${state.ip}:${state.port}`;
    };

    portInput.oninput = () => {
        state.port = portInput.value.trim();
        updateAddress();
        const targetEl = document.getElementById('clientTarget');
        if (targetEl) targetEl.innerText = `${state.ip}:${state.port}`;
    };

    document.getElementById('startBtn').onclick = async () => {
        if (state.isRunning) {
            await StopService();
            state.isRunning = false;
        } else {
            updateAddress();
            await StartService(state.role, state.address, state.position);
            state.isRunning = true;
        }
        renderClientScreen();
    };
}

function updateAddress() {
    if (state.role === 'server') {
        state.address = `:${state.port}`;
    } else {
        state.address = `${state.ip}:${state.port}`;
    }
    SaveConfig(state.role, state.position, state.address);
}

function getTransitionText() {
    let arrow = '◄';
    if (state.position === 'right') arrow = '►';
    if (state.position === 'top') arrow = '▲';
    if (state.position === 'bottom') arrow = '▼';
    return `${arrow} SECOND PC: ${state.position.toUpperCase()} ${arrow}`;
}

// -----------------------------------------------------------------------------
// Interactive Monitor Canvas for Host Mode: Drag and Snap around Host
// -----------------------------------------------------------------------------
function initCanvasArrangement() {
    const canvas = document.getElementById('canvas');
    if (!canvas) return;

    const hostBox = document.getElementById('hostBox');
    const secondBox = document.getElementById('secondBox');
    if (!hostBox || !secondBox) return;

    function updatePositions(smooth = false) {
        const cWidth = canvas.clientWidth || 700;
        const cHeight = canvas.clientHeight || 190;
        const boxW = 160;
        const boxH = 64;
        const gap = 20;

        let hX = 0, hY = 0, sX = 0, sY = 0;

        switch (state.position) {
        case 'left':
            // Second PC on the LEFT of Host
            const totalWLeft = boxW * 2 + gap;
            const startXLeft = (cWidth - totalWLeft) / 2;
            const centerYLeft = (cHeight - boxH) / 2;
            sX = startXLeft;
            sY = centerYLeft;
            hX = startXLeft + boxW + gap;
            hY = centerYLeft;
            break;

        case 'right':
            // Second PC on the RIGHT of Host
            const totalWRight = boxW * 2 + gap;
            const startXRight = (cWidth - totalWRight) / 2;
            const centerYRight = (cHeight - boxH) / 2;
            hX = startXRight;
            hY = centerYRight;
            sX = startXRight + boxW + gap;
            sY = centerYRight;
            break;

        case 'top':
            // Second PC on TOP of Host
            const totalHTop = boxH * 2 + gap;
            const centerXTop = (cWidth - boxW) / 2;
            const startYTop = (cHeight - totalHTop) / 2;
            sX = centerXTop;
            sY = startYTop;
            hX = centerXTop;
            hY = startYTop + boxH + gap;
            break;

        case 'bottom':
            // Second PC on BOTTOM of Host
            const totalHBottom = boxH * 2 + gap;
            const centerXBottom = (cWidth - boxW) / 2;
            const startYBottom = (cHeight - totalHBottom) / 2;
            hX = centerXBottom;
            hY = startYBottom;
            sX = centerXBottom;
            sY = startYBottom + boxH + gap;
            break;
        }

        if (smooth) {
            hostBox.style.transition = 'left 0.25s cubic-bezier(0.4, 0, 0.2, 1), top 0.25s cubic-bezier(0.4, 0, 0.2, 1)';
            secondBox.style.transition = 'left 0.25s cubic-bezier(0.4, 0, 0.2, 1), top 0.25s cubic-bezier(0.4, 0, 0.2, 1)';
            setTimeout(() => {
                hostBox.style.transition = '';
                secondBox.style.transition = '';
            }, 260);
        }

        hostBox.style.left = `${Math.round(hX)}px`;
        hostBox.style.top = `${Math.round(hY)}px`;
        secondBox.style.left = `${Math.round(sX)}px`;
        secondBox.style.top = `${Math.round(sY)}px`;
    }

    // Position immediately
    updatePositions(false);

    // Keep monitors centered when window is resized or floated
    if (window.ResizeObserver) {
        const ro = new ResizeObserver(() => {
            updatePositions(false);
        });
        ro.observe(canvas);
    }

    // Make SECOND PC draggable
    makeDraggable(secondBox, canvas, hostBox, (newPos) => {
        state.position = newPos;
        SaveConfig(state.role, state.position, state.address);
        const transText = document.getElementById('transitionText');
        if (transText) transText.innerText = getTransitionText();
        updatePositions(true);
    });
}

function makeDraggable(boxEl, canvasEl, hostBoxEl, onSnap) {
    let isDragging = false;
    let startX = 0;
    let startY = 0;
    let initLeft = 0;
    let initTop = 0;

    boxEl.onpointerdown = (e) => {
        isDragging = true;
        boxEl.classList.add('dragging');
        startX = e.clientX;
        startY = e.clientY;
        initLeft = parseFloat(boxEl.style.left) || 0;
        initTop = parseFloat(boxEl.style.top) || 0;
        boxEl.setPointerCapture(e.pointerId);
    };

    boxEl.onpointermove = (e) => {
        if (!isDragging) return;
        const dx = e.clientX - startX;
        const dy = e.clientY - startY;
        boxEl.style.left = `${initLeft + dx}px`;
        boxEl.style.top = `${initTop + dy}px`;
    };

    boxEl.onpointerup = (e) => {
        if (!isDragging) return;
        isDragging = false;
        boxEl.classList.remove('dragging');
        boxEl.releasePointerCapture(e.pointerId);

        const currentLeft = parseFloat(boxEl.style.left) || 0;
        const currentTop = parseFloat(boxEl.style.top) || 0;
        const hostLeft = parseFloat(hostBoxEl.style.left) || 0;
        const hostTop = parseFloat(hostBoxEl.style.top) || 0;

        const diffX = currentLeft - hostLeft;
        const diffY = currentTop - hostTop;

        let targetSnap = 'left';
        if (Math.abs(diffX) > Math.abs(diffY)) {
            targetSnap = diffX >= 0 ? 'right' : 'left';
        } else {
            targetSnap = diffY >= 0 ? 'bottom' : 'top';
        }

        if (onSnap) onSnap(targetSnap);
    };
}

// -----------------------------------------------------------------------------
// Telemetry Polling Loop (250ms)
// -----------------------------------------------------------------------------
setInterval(async () => {
    if (state.screen !== 3) return;
    try {
        const tele = await GetTelemetry();
        if (!tele) return;

        state.isRunning = tele.isRunning;
        const isConnected = tele.status === 'CONNECTED';

        if (state.role === 'server') {
            const secondText = document.getElementById('secondText');
            const secondDot = document.getElementById('secondDot');
            const serverState = document.getElementById('serverState');
            const clientState = document.getElementById('clientState');
            const telemetryControl = document.getElementById('telemetryControl');
            const telemetryPackets = document.getElementById('telemetryPackets');
            const hostBarDot = document.getElementById('hostBarDot');
            const hostBarText = document.getElementById('hostBarText');

            if (serverState) {
                serverState.innerText = tele.isRunning ? '● Listening' : '○ Stopped';
                serverState.style.color = tele.isRunning ? '#39d353' : '#8b949e';
            }

            if (isConnected) {
                const clientIp = tele.remoteAddr || 'Connected';
                if (secondText) secondText.innerText = clientIp;
                if (secondDot) secondDot.className = 'status-dot active';
                if (clientState) {
                    clientState.innerText = `● ${clientIp}`;
                    clientState.style.color = '#39d353';
                }
            } else {
                if (secondText) secondText.innerText = 'Waiting...';
                if (secondDot) secondDot.className = 'status-dot';
                if (clientState) {
                    clientState.innerText = '○ Waiting for connection...';
                    clientState.style.color = '#8b949e';
                }
            }

            if (telemetryControl) telemetryControl.innerText = tele.activeControl || 'HOST (LOCAL)';
            if (telemetryPackets) telemetryPackets.innerText = `${tele.packetsSent} sent / ${tele.packetsRecv} recv`;

            if (hostBarDot) {
                hostBarDot.className = tele.isRunning ? 'status-dot active' : 'status-dot';
            }
            if (hostBarText) {
                hostBarText.innerText = tele.isRunning ? `HOST ACTIVE — LISTENING ON ${state.lanIP}:${state.port}` : 'HOST STOPPED';
            }
        } else {
            // Client mode
            const clientState = document.getElementById('clientState');
            const telemetryControl = document.getElementById('telemetryControl');
            const telemetryPackets = document.getElementById('telemetryPackets');
            const startBtn = document.getElementById('startBtn');

            if (clientState) {
                if (isConnected) {
                    clientState.innerText = `● Connected (${tele.remoteAddr || state.address})`;
                    clientState.style.color = '#39d353';
                } else if (tele.isRunning) {
                    clientState.innerText = `◌ Connecting to ${state.address || (state.ip + ':' + state.port)}...`;
                    clientState.style.color = '#e3b341';
                } else {
                    clientState.innerText = '○ Disconnected';
                    clientState.style.color = '#8b949e';
                }
            }

            if (telemetryControl) telemetryControl.innerText = tele.activeControl || 'HOST (LOCAL)';
            if (telemetryPackets) telemetryPackets.innerText = `${tele.packetsRecv} recv`;

            if (startBtn) {
                if (tele.isRunning && !startBtn.classList.contains('running')) {
                    startBtn.classList.add('running');
                    startBtn.innerText = '[ ⏹ DISCONNECT ]';
                } else if (!tele.isRunning && startBtn.classList.contains('running')) {
                    startBtn.classList.remove('running');
                    startBtn.innerText = '[ ▶ CONNECT TO HOST ]';
                }
            }
        }
    } catch (e) {
        // Ignored
    }
}, 250);

// Start app
init();
