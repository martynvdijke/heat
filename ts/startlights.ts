import './theme';
import { StartLightsEngine } from './startlights-core';
// Start Light System - F1-style 5-light countdown
// Thin wrapper: wires the shared StartLightsEngine to the standalone page DOM.

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

const engine = new StartLightsEngine({ setLightState, showMessage, showStatusBar });

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
                    engine.handleCommand(data);
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