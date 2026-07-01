import './theme';
import { showToast } from './toast';

interface ControllerRacer {
    id: number;
    name: string;
    profile_picture: string;
    car_color: string;
    car_name: string;
    points: number;
    rank: number;
    position: number;
}

let raceState = 'stopped';
let raceTimer: ReturnType<typeof setInterval> | null = null;
let raceSeconds = 0;
let controllerRacers: ControllerRacer[] = [];
let currentLap = 0;

function startRaceTimer(): void {
    raceTimer = setInterval(() => {
        raceSeconds++;
        const mins = Math.floor(raceSeconds / 60);
        const secs = raceSeconds % 60;
        document.getElementById('race-timer')!.textContent =
            `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }, 1000);
}

async function loadControllerData(): Promise<void> {
    const [racersRes, tracksRes] = await Promise.all([
        fetch('/api/racers'),
        fetch('/api/tracks')
    ]);
    controllerRacers = await racersRes.json();
    const tracks = await tracksRes.json();
    renderStandings();
    populateDriverSelect();
    const trackSelect = document.getElementById('track-select') as HTMLSelectElement;
    tracks.forEach((t: any) => {
        const opt = document.createElement('option');
        opt.value = t.id;
        opt.textContent = `${t.country} - ${t.name}`;
        trackSelect.appendChild(opt);
    });
}

function renderStandings(): void {
    const sorted = [...controllerRacers].sort((a, b) => a.position - b.position);
    document.getElementById('standings-list')!.innerHTML = sorted.map((r, i) => `
        <div class="driver-row ${i === 0 ? 'active' : ''}">
            <div class="position-btn btn ${r.position === 1 ? 'btn-warning' : r.position <= 3 ? 'btn-secondary' : 'btn-outline-light'} me-2">
                ${r.position}
            </div>
            <img src="${r.profile_picture}" class="rounded-circle me-2" width="32" height="32" onerror="this.style.display='none'">
            <div class="flex-grow-1">
                <div class="fw-bold small">${r.name}</div>
                <small class="opacity-50">${r.car_name}</small>
            </div>
            <button class="btn btn-sm btn-outline-info ms-1" onclick="triggerBlueFlag(${r.id}, '${r.name}')" title="Blue Flag">
                <i class="fa-solid fa-flag"></i>
            </button>
            <button class="btn btn-sm btn-outline-secondary ms-1" onclick="triggerBlackWhiteFlag(${r.id}, '${r.name}')" title="Black & White Flag">
                <i class="fa-solid fa-flag-checkered"></i>
            </button>
            <button class="btn btn-sm btn-outline-secondary ms-1" onclick="moveUp(${r.id})">
                <i class="fa-solid fa-chevron-up"></i>
            </button>
            <button class="btn btn-sm btn-outline-secondary ms-1" onclick="moveDown(${r.id})">
                <i class="fa-solid fa-chevron-down"></i>
            </button>
        </div>
    `).join('');
}

function startRace(): void {
    if (raceState === 'stopped') {
        raceState = 'racing';
        raceSeconds = 0;
        currentLap = 1;
        updateStatus();
        startRaceTimer();
    }
}

function pauseRace(): void {
    if (raceState === 'racing') {
        raceState = 'paused';
        clearInterval(raceTimer!);
        updateStatus();
    } else if (raceState === 'paused') {
        raceState = 'racing';
        startRaceTimer();
        updateStatus();
    }
}

function stopRace(): void {
    raceState = 'stopped';
    clearInterval(raceTimer!);
    raceTimer = null;
    currentLap = 0;
    updateStatus();
}

function updateStatus(): void {
    const dot = document.getElementById('status-dot')!;
    const status = document.getElementById('race-status')!;
    const state = document.getElementById('race-state')!;
    dot.className = 'status-indicator';
    switch (raceState) {
        case 'racing':
            dot.classList.add('racing');
            status.textContent = 'RACING';
            state.textContent = 'GREEN FLAG';
            break;
        case 'paused':
            dot.classList.add('ready');
            status.textContent = 'PAUSED';
            state.textContent = 'YELLOW FLAG';
            break;
        default:
            dot.classList.add('stopped');
            status.textContent = 'STOPPED';
            state.textContent = 'READY';
    }
    document.getElementById('current-lap')!.textContent = String(currentLap);
}

function shuffleGrid(): void {
    const shuffled = [...controllerRacers].sort(() => Math.random() - 0.5);
    shuffled.forEach((r, i) => {
        r.rank = i + 1;
        r.position = 100 - i;
    });
    fetch('/api/racers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(controllerRacers)
    }).then(() => renderStandings());
}

function moveUp(id: number): void {
    const racer = controllerRacers.find(r => r.id === id);
    if (racer && racer.position > 1) {
        const targetPos = racer.position - 1;
        const other = controllerRacers.find(o => o.id !== id && o.position === targetPos);
        if (other) {
            other.position = racer.position;
            racer.position = targetPos;
        }
        savePositions();
    }
}

function moveDown(id: number): void {
    const racer = controllerRacers.find(r => r.id === id);
    if (racer) {
        const targetPos = racer.position + 1;
        const other = controllerRacers.find(o => o.id !== id && o.position === targetPos);
        if (other) {
            other.position = racer.position;
            racer.position = targetPos;
        }
        savePositions();
    }
}

function savePositions(): void {
    fetch('/api/racers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(controllerRacers)
    }).then(() => renderStandings());
}

let safetyActive = false;
let redFlagActive = false;

function toggleSafetyCar(btn: HTMLButtonElement): void {
    safetyActive = !safetyActive;
    btn.classList.toggle('active-flag', safetyActive);
    btn.innerHTML = safetyActive
        ? '<i class="fa-solid fa-car-side me-2"></i>Safety ON'
        : '<i class="fa-solid fa-car-side me-2"></i>Safety OFF';
    broadcastMessage({ type: 'flag', flag: safetyActive ? 'safety' : 'clear', state: safetyActive ? 'on' : 'off' });
}

function toggleRedFlag(btn: HTMLButtonElement): void {
    redFlagActive = !redFlagActive;
    btn.classList.toggle('active-flag', redFlagActive);
    btn.innerHTML = redFlagActive
        ? '<i class="fa-solid fa-circle-exclamation me-2"></i>Red Flag ON'
        : '<i class="fa-solid fa-circle-exclamation me-2"></i>Red Flag OFF';
    broadcastMessage({ type: 'flag', flag: redFlagActive ? 'red' : 'clear', state: redFlagActive ? 'on' : 'off' });
}

function triggerYellowFlag(): void {
    broadcastMessage({ type: 'flag', flag: 'yellow', state: 'on' });
    setTimeout(() => broadcastMessage({ type: 'flag', flag: 'yellow', state: 'off' }), 5000);
}

function triggerChequeredFlag(): void {
    broadcastMessage({ type: 'flag', flag: 'chequered', state: 'on' });
    setTimeout(() => broadcastMessage({ type: 'flag', flag: 'chequered', state: 'off' }), 8000);
}

function populateDriverSelect(): void {
    const select = document.getElementById('flag-driver-select') as HTMLSelectElement;
    if (!select) return;
    select.innerHTML = '<option value="">Select driver...</option>' +
        controllerRacers.map(r => `<option value="${r.id}">${r.name}</option>`).join('');
}

function sendBlueFlag(): void {
    const select = document.getElementById('flag-driver-select') as HTMLSelectElement;
    const id = parseInt(select.value);
    if (!id) { showToast('Select a driver first', 'warning'); return; }
    const name = select.options[select.selectedIndex]?.text || '';
    broadcastMessage({ type: 'flag', flag: 'blue', racer_id: id, racer_name: name });
}

function sendBlackWhiteFlag(): void {
    const select = document.getElementById('flag-driver-select') as HTMLSelectElement;
    const id = parseInt(select.value);
    if (!id) { showToast('Select a driver first', 'warning'); return; }
    const name = select.options[select.selectedIndex]?.text || '';
    broadcastMessage({ type: 'flag', flag: 'blackwhite', racer_id: id, racer_name: name });
}

function triggerBlueFlag(id: number, name: string): void {
    broadcastMessage({ type: 'flag', flag: 'blue', racer_id: id, racer_name: name });
}

function triggerBlackWhiteFlag(id: number, name: string): void {
    broadcastMessage({ type: 'flag', flag: 'blackwhite', racer_id: id, racer_name: name });
}

function saveRaceSettings(): void {
    const track = (document.getElementById('track-select') as HTMLSelectElement).value;
    const laps = parseInt((document.getElementById('race-laps') as HTMLInputElement).value) || 53;
    const name = (document.getElementById('race-name') as HTMLInputElement).value;
    fetch('/api/race-info', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            track_id: track || 'monza',
            laps: laps,
            name: name,
            is_oneoff: (document.getElementById('race-type') as HTMLSelectElement).value === 'oneoff'
        })
    });
}

function saveRaceResult(): void {
    const raceType = (document.getElementById('race-type') as HTMLSelectElement).value;
    const results = controllerRacers.map(r => ({
        racer_id: r.id,
        racer_name: r.name,
        position: r.position,
        points: getPointsForPosition(r.position),
        fastestLap: r.position === 1,
        finished: true
    }));
    fetch('/api/race-history', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            name: (document.getElementById('race-name') as HTMLInputElement).value || `Race ${new Date().toLocaleDateString()}`,
            race_date: new Date().toISOString().split('T')[0],
            country: ((document.getElementById('track-select') as HTMLSelectElement).selectedOptions[0]?.text?.split(' - ')[0]) || 'Unknown',
            track: ((document.getElementById('track-select') as HTMLSelectElement).selectedOptions[0]?.text?.split(' - ')[1]) || 'Unknown',
            track_id: (document.getElementById('track-select') as HTMLSelectElement).value || 'monza',
            total_laps: parseInt((document.getElementById('race-laps') as HTMLInputElement).value) || 53,
            race_type: raceType,
            results: results
        })
    }).then(() => {
        showToast('Race saved to history!', 'success');
        stopRace();
    });
}

function discardRace(): void {
    if (confirm('Discard this race? All progress will be lost.')) {
        stopRace();
    }
}

async function archiveCurrentSeason(): Promise<void> {
    const res = await fetch('/api/seasons');
    const seasons = await res.json();
    const active = Array.isArray(seasons) ? seasons.find((s: any) => s.status === 'active') : null;
    if (!active) {
        if (!confirm('No active season found. Create a new one?')) return;
        const name = prompt('Season name:', `Season ${new Date().getFullYear()}`);
        if (!name) return;
        const createRes = await fetch('/api/seasons', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
        });
        if (!createRes.ok) { showToast('Failed to create season', 'error'); return; }
        showToast('Season created! You can now archive it when ready.', 'success');
        return;
    }
    if (!confirm(`Archive "${active.name}"? This will end the season.`)) return;
    const archiveRes = await fetch(`/api/seasons/archive?id=${active.id}`, { method: 'POST' });
    if (archiveRes.ok) {
        showToast(`Season "${active.name}" archived!`, 'success');
    } else {
        showToast('Failed to archive season', 'error');
    }
}

async function takeRoundSnapshot(): Promise<void> {
    const name = (document.getElementById('race-name') as HTMLInputElement).value || `Round ${new Date().toLocaleDateString()}`;
    const res = await fetch('/api/rounds', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ race_name: name, round: 0 })
    });
    if (res.ok) {
        showToast('Round snapshot saved!', 'success');
    } else {
        const err = await res.json();
        showToast('Failed: ' + (err.error || 'Unknown error'), 'error');
    }
}

function getPointsForPosition(pos: number): number {
    const points = [25, 18, 15, 12, 10, 8, 6, 4, 2, 1];
    return points[pos - 1] || 0;
}

function broadcastMessage(msg: Record<string, unknown>): void {
    fetch('/api/flags', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(msg)
    });
}

// Weather
async function setWeather(): Promise<void> {
    const condition = (document.getElementById('weather-condition') as HTMLSelectElement).value;
    const lapStart = parseInt((document.getElementById('weather-lap-start') as HTMLInputElement).value) || 1;
    const lapEnd = 999;
    const gripMap: Record<string, number> = { dry: 1.0, damp: 0.85, wet: 0.7, torrential: 0.5 };

    await fetch('/api/weather', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ race_id: 0, condition, lap_start: lapStart, lap_end: lapEnd, grip_modifier: gripMap[condition] || 1.0 })
    });

    const weatherNames: Record<string, string> = { dry: '☀️ Dry', damp: '🌧️ Damp', wet: '🌧️ Wet', torrential: '⛈️ Torrential' };
    document.getElementById('current-weather')!.textContent = `Current: ${weatherNames[condition] || condition}`;
}

// Turbo tracking
async function refreshTurboLog(): Promise<void> {
    const res = await fetch('/api/turbo-logs');
    const logs = await res.json();
    const list = document.getElementById('turbo-list')!;
    if (!logs.length) {
        list.innerHTML = '<p class="text-center opacity-50">No turbo used yet</p>';
        return;
    }
    list.innerHTML = logs.map((l: any) =>
        `<div class="d-flex justify-content-between py-1 border-bottom border-secondary border-opacity-25">
            <span>${getRacerName(l.racer_id)}</span>
            <span class="badge bg-success">Lap ${l.lap} x${l.times_used}</span>
        </div>`
    ).join('');
}

function getRacerName(id: number): string {
    const r = controllerRacers.find(r => r.id === id);
    return r ? r.name : `#${id}`;
}

// Gear log
async function refreshGearLog(): Promise<void> {
    const res = await fetch('/api/gear-shifts');
    const shifts = await res.json();
    const log = document.getElementById('gear-log')!;
    if (!shifts.length) {
        log.innerHTML = '<p class="text-center opacity-50">No shifts reported</p>';
        return;
    }
    const recent = shifts.slice(-20).reverse();
    log.innerHTML = recent.map((s: any) =>
        `<div class="d-flex justify-content-between py-1 border-bottom border-secondary border-opacity-25">
            <span>${getRacerName(s.racer_id)}</span>
            <span>Lap ${s.lap}: Gear ${s.gear} ${s.stress ? `+${s.stress} Stress` : ''}</span>
        </div>`
    ).join('');
}

// Lap recording
async function recordCurrentLap(): Promise<void> {
    const lapNum = parseInt((document.getElementById('record-lap-number') as HTMLInputElement).value) || 1;

    const records = controllerRacers.map(r => ({
        racer_id: r.id, position: r.position, gear_used: 0, heat_generated: 0, turbo_used: false
    }));

    await fetch('/api/lap-records/batch', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ race_id: 0, lap: lapNum, records })
    });

    (document.getElementById('record-lap-number') as HTMLInputElement).value = (lapNum + 1).toString();
    document.getElementById('current-lap')!.textContent = lapNum.toString();
}

// Race events
async function addRaceEvent(): Promise<void> {
    const eventType = (document.getElementById('event-type-select') as HTMLSelectElement).value;
    const racerSelect = document.getElementById('event-racer-select') as HTMLSelectElement;
    const racerId = parseInt(racerSelect.value);
    if (!racerId) { showToast('Select a driver', 'warning'); return; }
    const lap = parseInt((document.getElementById('record-lap-number') as HTMLInputElement).value) || 1;

    await fetch('/api/race-events', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ race_id: 0, lap, event_type: eventType, racer_id: racerId })
    });
    refreshRaceEvents();
}

async function refreshRaceEvents(): Promise<void> {
    const res = await fetch('/api/race-events?race_id=0');
    const events = await res.json();
    const log = document.getElementById('race-event-log')!;
    if (!events.length) {
        log.innerHTML = '<p class="text-center opacity-50">No events</p>';
        return;
    }
    const recent = events.slice(-10).reverse();
    const eventIcons: Record<string, string> = {
        overtake: '🏎️', crash: '💥', spin: '🔄', safety_car: '🚗', pit_stop: '🔧'
    };
    log.innerHTML = recent.map((e: any) =>
        `<div class="d-flex justify-content-between py-1 border-bottom border-secondary border-opacity-25">
            <span>${eventIcons[e.event_type] || '📌'} ${getRacerName(e.racer_id)}</span>
            <small class="opacity-75">Lap ${e.lap} - ${e.event_type}</small>
        </div>`
    ).join('');
}

// Player sessions
async function refreshPlayerSessions(): Promise<void> {
    const res = await fetch('/api/player-sessions');
    const sessions = await res.json();
    const div = document.getElementById('connected-players')!;
    if (!Array.isArray(sessions) || !sessions.length) {
        div.innerHTML = '<p class="text-center opacity-50">No players connected</p>';
        return;
    }
    div.innerHTML = sessions.map((s: any) =>
        `<div class="d-flex justify-content-between py-1 border-bottom border-secondary border-opacity-25">
            <span><i class="fa-solid fa-user me-1"></i>${s.racer_name}</span>
            <small class="opacity-75">${s.device_name || ''} ${s.last_seen ? '· ' + s.last_seen : ''}</small>
        </div>`
    ).join('');
}

// Periodic refresh
setInterval(refreshTurboLog, 5000);
setInterval(refreshGearLog, 5000);
setInterval(refreshRaceEvents, 5000);
setInterval(refreshPlayerSessions, 5000);

// Sound FX
async function playSound(sound: string): Promise<void> {
    await fetch('/api/sound', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sound })
    });
}

// Onclick delegation: map data-action attributes to module-scoped function calls
// Replaces previously inline onclick attributes in controller.html
const actionHandlers: Record<string, () => void> = {
    startRace, pauseRace, stopRace,
    shuffleGrid, triggerYellowFlag, triggerChequeredFlag,
    takeRoundSnapshot, sendBlueFlag, sendBlackWhiteFlag,
    addRaceEvent, recordCurrentLap, setWeather,
    saveRaceSettings, saveRaceResult, archiveCurrentSeason, discardRace,
    triggerStartLights: () => {
        const fn = (window as any).triggerStartLights;
        if (fn) fn();
    },
};

document.addEventListener('click', (e: Event) => {
    const target = (e.target as HTMLElement).closest('[data-action]') as HTMLElement;
    if (!target) return;
    const action = target.getAttribute('data-action');
    if (!action) return;

    if (action === 'toggleSafetyCar') {
        toggleSafetyCar(target as HTMLButtonElement);
        return;
    }
    if (action === 'toggleRedFlag') {
        toggleRedFlag(target as HTMLButtonElement);
        return;
    }
    if (action === 'playSound') {
        playSound(target.getAttribute('data-value') || 'engine');
        return;
    }

    const handler = actionHandlers[action];
    if (handler) {
        e.preventDefault();
        handler();
    }
});

loadControllerData();

