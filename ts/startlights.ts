import './theme';
// Start Light System - F1-style 5-light countdown
interface StartLightsState {
    phase: 'idle' | 'counting' | 'green' | 'done';
    currentLight: number; // 0-5 (5 = all lit, ready for green)
}

let lightsState: StartLightsState = { phase: 'idle', currentLight: 0 };
let lightsTimer: ReturnType<typeof setTimeout> | null = null;
let audioCtx: AudioContext | null = null;

// Sequence timing (ms)
const LIGHT_INTERVAL = 1000;
const HOLD_ON_LIGHTS = 1000;
const GREEN_DURATION = 3000;

function getAudioContext(): AudioContext {
    if (!audioCtx) {
        audioCtx = new (window.AudioContext || (window as any).webkitAudioContext)();
    }
    return audioCtx;
}

function playBeep(frequency: number, duration: number, volume = 0.3): void {
    try {
        const ctx = getAudioContext();
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = 'square';
        osc.frequency.setValueAtTime(frequency, ctx.currentTime);
        gain.gain.setValueAtTime(volume, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + duration);
        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.start();
        osc.stop(ctx.currentTime + duration);
    } catch {
        // Audio not available
    }
}

function playHorn(): void {
    try {
        const ctx = getAudioContext();
        // Layered frequencies for a rich horn sound
        [220, 330, 440].forEach((freq, i) => {
            const osc = ctx.createOscillator();
            const gain = ctx.createGain();
            osc.type = 'sawtooth';
            osc.frequency.setValueAtTime(freq, ctx.currentTime + i * 0.05);
            gain.gain.setValueAtTime(0.15, ctx.currentTime + i * 0.05);
            gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 1.5 + i * 0.05);
            osc.connect(gain);
            gain.connect(ctx.destination);
            osc.start(ctx.currentTime + i * 0.05);
            osc.stop(ctx.currentTime + 1.5 + i * 0.05);
        });
    } catch {
        // Audio not available
    }
}

function renderStartLights(): void {
    const container = document.getElementById('start-lights');
    if (!container) return;
    container.innerHTML = '';
    for (let i = 0; i < 5; i++) {
        const light = document.createElement('div');
        light.className = 'start-light';
        light.id = `start-light-${i + 1}`;
        light.dataset.light = String(i + 1);

        const bulb = document.createElement('div');
        bulb.className = 'start-light-bulb';

        const label = document.createElement('div');
        label.className = 'start-light-label';
        label.textContent = String(i + 1);

        light.appendChild(bulb);
        light.appendChild(label);
        container.appendChild(light);
    }
}

function setLightState(lightNum: number, state: 'off' | 'red' | 'green'): void {
    const bulb = document.querySelector(`#start-light-${lightNum} .start-light-bulb`);
    if (!bulb) return;
    bulb.className = 'start-light-bulb';
    if (state === 'red') {
        bulb.classList.add('red');
    } else if (state === 'green') {
        bulb.classList.add('green');
    }
}

function showMessage(text: string, subtext = ''): void {
    const msgEl = document.getElementById('start-message');
    const subEl = document.getElementById('start-submessage');
    if (msgEl) msgEl.textContent = text;
    if (subEl) subEl.textContent = subtext;
}

function showStatusBar(text: string): void {
    const bar = document.getElementById('start-status-bar');
    if (bar) bar.textContent = text;
}

function resetAllLights(): void {
    for (let i = 1; i <= 5; i++) {
        setLightState(i, 'off');
    }
}

function runSequence(): void {
    if (lightsState.phase !== 'idle') return;

    lightsState = { phase: 'counting', currentLight: 0 };
    resetAllLights();
    showMessage('');
    showStatusBar('Sequence started');
    showMessage('', '5 lights');
    showStatusBar('START LIGHTS • SEQUENCE');

    let lightIndex = 0;

    function lightNext(): void {
        if (lightsState.phase !== 'counting') return;

        lightIndex++;
        if (lightIndex > 5) {
            // All lights on - hold then green
            lightsState.currentLight = 5;
            showMessage('', 'All lights on');
            playBeep(880, 0.3, 0.5);

            lightsTimer = setTimeout(() => {
                if (lightsState.phase !== 'counting') return;
                // ALL GREEN!
                lightsState.phase = 'green';
                for (let i = 1; i <= 5; i++) {
                    setLightState(i, 'green');
                }
                showMessage('GO! GO! GO!', '');
                showStatusBar('🟢 GREEN FLAG • RACE START!');
                playHorn();

                lightsTimer = setTimeout(() => {
                    lightsState.phase = 'done';
                    showMessage('Race Started', '');
                    showStatusBar('RACE IS ON');
                    lightsTimer = setTimeout(() => {
                        lightsState.phase = 'idle';
                        lightsState.currentLight = 0;
                        resetAllLights();
                        showMessage('Start Lights', 'Ready');
                        showStatusBar('START LIGHTS • READY');
                    }, 2000);
                }, GREEN_DURATION);
            }, HOLD_ON_LIGHTS);
            return;
        }

        lightsState.currentLight = lightIndex;
        setLightState(lightIndex, 'red');
        showMessage('', `${lightIndex}/5 lights`);
        playBeep(440 + lightIndex * 60, 0.2, 0.4);

        lightsTimer = setTimeout(lightNext, LIGHT_INTERVAL);
    }

    // Start the sequence
    lightsTimer = setTimeout(lightNext, LIGHT_INTERVAL);
}

function abortSequence(): void {
    if (lightsTimer) {
        clearTimeout(lightsTimer);
        lightsTimer = null;
    }
    lightsState = { phase: 'idle', currentLight: 0 };
    resetAllLights();
    showMessage('Aborted', '');
    showStatusBar('START LIGHTS • ABORTED');
}

function handleStartLightsCommand(cmd: any): void {
    if (cmd.state === 'sequence') {
        runSequence();
    } else if (cmd.state === 'abort') {
        abortSequence();
    } else if (cmd.state === 'reset') {
        abortSequence();
        showMessage('Start Lights', 'Ready');
        showStatusBar('START LIGHTS • READY');
    }
}

// Export for use in controller.ts
(window as any).triggerStartLights = function (): void {
    fetch('/api/flags', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'flag', flag: 'startlights', state: 'sequence' })
    });
};

(window as any).abortStartLights = function (): void {
    fetch('/api/flags', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'flag', flag: 'startlights', state: 'abort' })
    });
};

(window as any).resetStartLights = function (): void {
    fetch('/api/flags', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'flag', flag: 'startlights', state: 'reset' })
    });
};

// Initialize if on the start lights page
document.addEventListener('DOMContentLoaded', () => {
    const container = document.getElementById('start-lights');
    if (container) {
        renderStartLights();
        showMessage('Start Lights', 'Ready');
        showStatusBar('START LIGHTS • READY');

        // Connect to WebSocket for start light commands
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
        ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                if (data.type === 'flag' && data.flag === 'startlights') {
                    handleStartLightsCommand(data);
                }
            } catch {
                // ignore parse errors
            }
        };
        ws.onclose = () => {
            showStatusBar('DISCONNECTED • Reconnecting...');
            setTimeout(() => {
                window.location.reload();
            }, 5000);
        };
    }
});

// WebSocket integration for controller.ts (reuses existing WS connection)
// The controller.ts will need to handle startlights flag messages
// This is done via the existing WebSocket message handler in controller.ts
