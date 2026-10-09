package stream

const RemoteHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>Cast Desktop Remote</title>
    <style>
        :root {
            --bg: #0b0f19;
            --card-bg: rgba(26, 34, 52, 0.75);
            --card-border: rgba(255, 255, 255, 0.08);
            --primary: #06b6d4;
            --primary-hover: #0891b2;
            --accent-green: #10b981;
            --accent-amber: #f59e0b;
            --accent-red: #ef4444;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
        body {
            background: radial-gradient(circle at top, #1e293b, var(--bg));
            color: var(--text-main);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
            padding: 1.5rem 1rem 3rem 1rem;
        }
        .container {
            width: 100%;
            max-width: 480px;
            display: flex;
            flex-direction: column;
            gap: 1.25rem;
        }
        header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0.5rem 0.25rem;
        }
        .brand {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }
        .brand-icon {
            width: 36px;
            height: 36px;
            background: linear-gradient(135deg, #06b6d4, #3b82f6);
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 4px 12px rgba(6, 182, 212, 0.35);
        }
        .brand-title {
            font-size: 1.15rem;
            font-weight: 700;
            letter-spacing: -0.02em;
        }
        .live-badge {
            display: inline-flex;
            align-items: center;
            gap: 0.4rem;
            padding: 0.35rem 0.75rem;
            border-radius: 9999px;
            background: rgba(16, 185, 129, 0.15);
            border: 1px solid rgba(16, 185, 129, 0.3);
            color: #34d399;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        .live-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: #10b981;
            box-shadow: 0 0 8px #10b981;
            animation: pulse 1.5s infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.4; transform: scale(0.85); }
        }
        .card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 16px;
            padding: 1.25rem;
            backdrop-filter: blur(12px);
            -webkit-backdrop-filter: blur(12px);
            box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
        }
        .card-header {
            font-size: 0.8rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.06em;
            color: var(--text-muted);
            margin-bottom: 0.75rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .device-info {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }
        .device-name {
            font-size: 1.35rem;
            font-weight: 700;
            color: #fff;
        }
        .tags-row {
            display: flex;
            flex-wrap: wrap;
            gap: 0.5rem;
            margin-top: 0.25rem;
        }
        .tag {
            font-size: 0.72rem;
            padding: 0.2rem 0.55rem;
            border-radius: 6px;
            background: rgba(255, 255, 255, 0.06);
            color: var(--text-muted);
            font-weight: 500;
        }
        .tag-active {
            background: rgba(6, 182, 212, 0.15);
            color: var(--primary);
            border: 1px solid rgba(6, 182, 212, 0.3);
        }
        .volume-display {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            margin-bottom: 0.75rem;
        }
        .volume-val {
            font-size: 2.25rem;
            font-weight: 800;
            letter-spacing: -0.03em;
            color: #fff;
        }
        .slider-container {
            margin: 1rem 0;
        }
        input[type="range"] {
            -webkit-appearance: none;
            width: 100%;
            height: 10px;
            border-radius: 5px;
            background: #334155;
            outline: none;
        }
        input[type="range"]::-webkit-slider-thumb {
            -webkit-appearance: none;
            width: 26px;
            height: 26px;
            border-radius: 50%;
            background: var(--primary);
            box-shadow: 0 0 12px rgba(6, 182, 212, 0.6);
            cursor: pointer;
            transition: transform 0.1s;
        }
        input[type="range"]::-webkit-slider-thumb:active {
            transform: scale(1.2);
        }
        .btn-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 0.5rem;
            margin-top: 0.75rem;
        }
        button {
            border: none;
            cursor: pointer;
            padding: 0.75rem 0.5rem;
            border-radius: 10px;
            font-weight: 600;
            font-size: 0.85rem;
            transition: all 0.15s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 0.4rem;
        }
        .btn-secondary {
            background: rgba(255, 255, 255, 0.08);
            color: var(--text-main);
        }
        .btn-secondary:hover {
            background: rgba(255, 255, 255, 0.15);
        }
        .btn-secondary:active {
            transform: scale(0.96);
        }
        .btn-control {
            padding: 1rem;
            font-size: 1rem;
            border-radius: 12px;
        }
        .btn-primary {
            background: linear-gradient(135deg, #06b6d4, #0284c7);
            color: #fff;
            box-shadow: 0 4px 14px rgba(6, 182, 212, 0.4);
        }
        .btn-primary:active { transform: scale(0.97); }
        .btn-danger {
            background: linear-gradient(135deg, #ef4444, #dc2626);
            color: #fff;
            box-shadow: 0 4px 14px rgba(239, 68, 68, 0.35);
        }
        .btn-danger:active { transform: scale(0.97); }
        .btn-muted {
            background: #e11d48 !important;
            color: #fff !important;
        }
        .playback-controls {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 0.75rem;
        }
        .select-custom {
            width: 100%;
            background: #1e293b;
            color: #fff;
            border: 1px solid rgba(255, 255, 255, 0.12);
            padding: 0.75rem;
            border-radius: 10px;
            font-size: 0.95rem;
            outline: none;
            margin-bottom: 0.75rem;
        }
        .qr-section {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 0.75rem;
            padding: 1rem 0;
        }
        .qr-img {
            width: 160px;
            height: 160px;
            border-radius: 12px;
            background: #fff;
            padding: 8px;
            box-shadow: 0 4px 16px rgba(0,0,0,0.4);
        }
        .toast {
            position: fixed;
            bottom: 1.5rem;
            background: #1e293b;
            border: 1px solid var(--primary);
            color: #fff;
            padding: 0.7rem 1.25rem;
            border-radius: 9999px;
            box-shadow: 0 8px 24px rgba(0,0,0,0.5);
            font-size: 0.85rem;
            font-weight: 500;
            opacity: 0;
            transform: translateY(20px);
            transition: all 0.25s ease;
            pointer-events: none;
            z-index: 1000;
        }
        .toast.show {
            opacity: 1;
            transform: translateY(0);
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="brand">
                <div class="brand-icon">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                        <line x1="1" y1="1" x2="23" y2="23"></line>
                        <path d="M21 21H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h3m3-3h6l2 3h4a2 2 0 0 1 2 2v9.34m-7.72-2.06a4 4 0 1 1-5.56-5.56"></path>
                    </svg>
                </div>
                <div>
                    <div class="brand-title">Cast Desktop</div>
                </div>
            </div>
            <div id="liveBadge" class="live-badge">
                <div class="live-dot" id="liveDot"></div>
                <span id="liveText">LIVE</span>
            </div>
        </header>

        <!-- Device & Stream Info Card -->
        <div class="card">
            <div class="card-header">
                <span>Active Target</span>
                <span id="uptime">00:00:00</span>
            </div>
            <div class="device-info">
                <div class="device-name" id="deviceName">Searching...</div>
                <div class="tags-row">
                    <span class="tag tag-active" id="tagPreset">Balanced (1.0s)</span>
                    <span class="tag" id="tagHW">AMD VA-API GPU</span>
                    <span class="tag" id="tagRes">1080p @ 30fps</span>
                </div>
            </div>
        </div>

        <!-- Volume Control Card -->
        <div class="card">
            <div class="card-header">
                <span>TV Volume</span>
                <span id="muteStatus">Active</span>
            </div>
            <div class="volume-display">
                <div class="volume-val"><span id="volNum">--</span>%</div>
                <button class="btn-secondary" id="btnMute" onclick="toggleMute()" style="padding: 0.5rem 0.9rem;">
                    <span id="muteIcon">🔊</span> <span id="muteLabel">Mute</span>
                </button>
            </div>
            <div class="slider-container">
                <input type="range" id="volSlider" min="0" max="100" value="50" oninput="onSliderInput(this.value)" onchange="sendVolume(this.value)">
            </div>
            <div class="btn-grid">
                <button class="btn-secondary" onclick="sendVolume(15)">15%</button>
                <button class="btn-secondary" onclick="sendVolume(30)">30%</button>
                <button class="btn-secondary" onclick="sendVolume(50)">50%</button>
                <button class="btn-secondary" onclick="sendVolume(75)">75%</button>
            </div>
        </div>

        <!-- Playback Controls Card -->
        <div class="card">
            <div class="card-header">
                <span>Playback & Power</span>
            </div>
            <div class="playback-controls">
                <button class="btn-control btn-secondary" id="btnPause" onclick="togglePlayPause()">
                    <span id="pauseIcon">⏸️</span> <span id="pauseLabel">Pause</span>
                </button>
                <button class="btn-control btn-danger" onclick="stopStream()">
                    ⏹️ Stop Cast
                </button>
            </div>
        </div>

        <!-- Device Switcher Card -->
        <div class="card">
            <div class="card-header">
                <span>Switch Output Device</span>
            </div>
            <select class="select-custom" id="deviceSelect">
                <option value="">Scanning network...</option>
            </select>
            <button class="btn-secondary" style="width: 100%;" onclick="switchDevice()">
                🔄 Switch to Selected Device
            </button>
        </div>

        <!-- Share Remote QR Card -->
        <div class="card">
            <div class="card-header">
                <span>Mobile Remote QR</span>
                <span>WiFi Share</span>
            </div>
            <div class="qr-section">
                <img class="qr-img" src="/qrcode" alt="Remote QR Code">
                <p style="font-size: 0.78rem; color: var(--text-muted); text-align: center;">
                    Scan with any phone camera on this WiFi to control TV volume and playback without installing any app.
                </p>
            </div>
        </div>
    </div>

    <div class="toast" id="toast"></div>

    <script>
        let isMuted = false;
        let isPaused = false;
        let lastSliderUpdate = 0;

        function showToast(msg) {
            const t = document.getElementById('toast');
            t.textContent = msg;
            t.classList.add('show');
            setTimeout(() => t.classList.remove('show'), 2000);
        }

        async function fetchStatus() {
            try {
                const res = await fetch('/api/status');
                if (!res.ok) return;
                const d = await res.json();
                
                document.getElementById('deviceName').textContent = d.device || 'Active Chromecast';
                document.getElementById('tagPreset').textContent = (d.preset || 'Balanced').toUpperCase();
                document.getElementById('tagHW').textContent = d.hw_accel ? 'AMD VA-API GPU' : 'Software x264';
                if (d.audio_only) {
                    document.getElementById('tagRes').textContent = 'Audio-Only FLAC';
                } else {
                    document.getElementById('tagRes').textContent = (d.resolution || '1080p') + ' @ ' + (d.fps || 30) + 'fps';
                }

                // Volume
                if (Date.now() - lastSliderUpdate > 3000) {
                    const vol = Math.round(d.volume || 0);
                    document.getElementById('volNum').textContent = vol;
                    document.getElementById('volSlider').value = vol;
                }

                // Mute state
                isMuted = !!d.muted;
                const btnMute = document.getElementById('btnMute');
                const muteIcon = document.getElementById('muteIcon');
                const muteLabel = document.getElementById('muteLabel');
                const muteStatus = document.getElementById('muteStatus');
                if (isMuted) {
                    btnMute.classList.add('btn-muted');
                    muteIcon.textContent = '🔇';
                    muteLabel.textContent = 'Unmute';
                    muteStatus.textContent = 'MUTED';
                    muteStatus.style.color = '#ef4444';
                } else {
                    btnMute.classList.remove('btn-muted');
                    muteIcon.textContent = '🔊';
                    muteLabel.textContent = 'Mute';
                    muteStatus.textContent = 'Active';
                    muteStatus.style.color = 'var(--accent-green)';
                }

                // Play/Pause state
                isPaused = !!d.paused;
                const pauseIcon = document.getElementById('pauseIcon');
                const pauseLabel = document.getElementById('pauseLabel');
                const liveBadge = document.getElementById('liveBadge');
                const liveDot = document.getElementById('liveDot');
                const liveText = document.getElementById('liveText');
                if (isPaused) {
                    pauseIcon.textContent = '▶️';
                    pauseLabel.textContent = 'Resume';
                    liveText.textContent = 'PAUSED';
                    liveBadge.style.background = 'rgba(245, 158, 11, 0.15)';
                    liveBadge.style.borderColor = 'rgba(245, 158, 11, 0.3)';
                    liveBadge.style.color = '#f59e0b';
                    liveDot.style.background = '#f59e0b';
                    liveDot.style.boxShadow = '0 0 8px #f59e0b';
                } else {
                    pauseIcon.textContent = '⏸️';
                    pauseLabel.textContent = 'Pause';
                    liveText.textContent = 'LIVE';
                    liveBadge.style.background = 'rgba(16, 185, 129, 0.15)';
                    liveBadge.style.borderColor = 'rgba(16, 185, 129, 0.3)';
                    liveBadge.style.color = '#34d399';
                    liveDot.style.background = '#10b981';
                    liveDot.style.boxShadow = '0 0 8px #10b981';
                }

                if (d.uptime) {
                    document.getElementById('uptime').textContent = d.uptime;
                }
            } catch (e) {
                console.error(e);
            }
        }

        async function fetchDevices() {
            try {
                const res = await fetch('/api/devices');
                if (!res.ok) return;
                const list = await res.json();
                const sel = document.getElementById('deviceSelect');
                sel.innerHTML = '';
                if (!list || list.length === 0) {
                    sel.innerHTML = '<option value="">No other devices discovered</option>';
                    return;
                }
                list.forEach(name => {
                    const opt = document.createElement('option');
                    opt.value = name;
                    opt.textContent = name;
                    sel.appendChild(opt);
                });
            } catch (e) {
                console.error(e);
            }
        }

        function onSliderInput(val) {
            document.getElementById('volNum').textContent = val;
            lastSliderUpdate = Date.now();
        }

        async function sendVolume(val) {
            lastSliderUpdate = Date.now();
            document.getElementById('volNum').textContent = val;
            document.getElementById('volSlider').value = val;
            try {
                const res = await fetch('/api/volume', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({volume: parseInt(val, 10)})
                });
                if (res.ok) {
                    showToast('TV Volume set to ' + val + '%');
                }
            } catch (e) {
                showToast('Failed to set volume');
            }
        }

        async function toggleMute() {
            try {
                const res = await fetch('/api/mute', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({muted: !isMuted})
                });
                if (res.ok) {
                    isMuted = !isMuted;
                    showToast(isMuted ? 'TV Muted' : 'TV Unmuted');
                    fetchStatus();
                }
            } catch (e) {
                showToast('Failed to toggle mute');
            }
        }

        async function togglePlayPause() {
            try {
                const endpoint = isPaused ? '/api/play' : '/api/pause';
                const res = await fetch(endpoint, { method: 'POST' });
                if (res.ok) {
                    isPaused = !isPaused;
                    showToast(isPaused ? 'Streaming Paused' : 'Streaming Resumed');
                    fetchStatus();
                }
            } catch (e) {
                showToast('Failed to toggle playback');
            }
        }

        async function stopStream() {
            if (!confirm('Are you sure you want to stop casting to TV?')) return;
            try {
                await fetch('/api/stop', { method: 'POST' });
                showToast('Casting stopped');
                document.body.innerHTML = '<div style="display:flex;justify-content:center;align-items:center;height:100vh;flex-direction:column;gap:1rem;color:#94a3b8;"><h2>Casting Stopped</h2><p>You can close this tab.</p></div>';
            } catch (e) {
                showToast('Failed to stop casting');
            }
        }

        async function switchDevice() {
            const sel = document.getElementById('deviceSelect');
            const target = sel.value;
            if (!target) return;
            try {
                const res = await fetch('/api/switch', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({device: target})
                });
                if (res.ok) {
                    showToast('Switching to ' + target + '...');
                    setTimeout(fetchStatus, 1500);
                }
            } catch (e) {
                showToast('Failed to switch device');
            }
        }

        fetchStatus();
        fetchDevices();
        setInterval(fetchStatus, 2000);
        setInterval(fetchDevices, 15000);
    </script>
</body>
</html>`
