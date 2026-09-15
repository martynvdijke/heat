import './theme';
import { WeatherEntry, getActiveWeather, getForecast, weatherIcon, weatherLabel, formatGrip } from './weather';
import { connectWithRetry, type HeatSocket } from './ws';
interface PitRacer {
    id: number; name: string; car_color: string; car_name: string;
    position: number; points: number; rank: number;
}

let pitRacers: PitRacer[] = [];
let pitSocket: HeatSocket | null = null;
let pitWeather: WeatherEntry[] = [];
let pitLap = 0;
type Standing = { racer_id: number; name: string; car_color: string; position: number; lap: number; gap: string };
let pitStandings: Standing[] = [];
function formatElapsed(ms: number): string {
    const totalSec = Math.floor(ms / 1000);
    const h = Math.floor(totalSec / 3600);
    const m = Math.floor((totalSec % 3600) / 60);
    const s = totalSec % 60;
    if (h > 0) return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}
function applyPitRaceState(payload: any): void {
    if (!payload) return;
    if (typeof payload.current_lap === 'number') pitLap = payload.current_lap;
    const lapEl = document.getElementById('pit-lap');
    if (lapEl) lapEl.textContent = `Lap ${pitLap}`;
    if (typeof payload.elapsed_ms === 'number') {
        const c = document.getElementById('pit-clock');
        if (c) c.textContent = formatElapsed(payload.elapsed_ms);
    }
    renderPitWeather();
    renderPitBoard();
}
function applyPitStandings(payload: any): void {
    if (Array.isArray(payload)) pitStandings = payload as Standing[];
    else return;
    renderPitBoard();
}

let pitRadioTimer: ReturnType<typeof setTimeout> | null = null;
function showPitRadio(message: string): void {
    let el = document.getElementById('pit-radio-toast');
    if (!el) {
        el = document.createElement('div');
        el.id = 'pit-radio-toast';
        el.style.cssText = 'position:fixed;left:50%;bottom:24px;transform:translateX(-50%);background:rgba(0,0,0,.85);color:#fff;padding:8px 14px;border-radius:6px;z-index:9999;font-size:1rem;max-width:80vw;text-align:center;';
        document.body.appendChild(el);
    }
    el.textContent = `📻 ${message}`;
    el.style.display = 'block';
    if (pitRadioTimer) clearTimeout(pitRadioTimer);
    pitRadioTimer = setTimeout(() => { if (el) el.style.display = 'none'; }, 6000);
}

async function loadPitBoard(): Promise<void> {
    const [racersRes, raceRes, weatherRes] = await Promise.all([
        fetch('/api/racers'), fetch('/api/race-info'), fetch('/api/weather?race_id=0')
    ]);
    pitRacers = await racersRes.json();
    const race = await raceRes.json();
    document.getElementById('pit-race-name')!.textContent = `${race.country} - ${race.track}`;
    renderPitBoard();

    pitWeather = await weatherRes.json();
    renderPitWeather();

    connectPitWebSocket();
    setInterval(refreshPitStatus, 5000);
}

function renderPitWeather(): void {
    const active = getActiveWeather(pitWeather, pitLap);
    const forecast = getForecast(pitWeather, pitLap);
    if (active) {
        document.getElementById('pit-weather-icon')!.textContent = weatherIcon(active.condition);
        document.getElementById('pit-weather-text')!.textContent = `${weatherLabel(active.condition)} · ${formatGrip(active.grip_modifier)}`;
    }
    const fc = document.getElementById('pit-weather-forecast')!;
    if (fc) {
        if (forecast) {
            fc.textContent = `${weatherIcon(forecast.condition)} ${weatherLabel(forecast.condition)} from lap ${forecast.lap_start}`;
            fc.hidden = false;
        } else fc.hidden = true;
    }
}

function upsertPitWeather(entry: WeatherEntry): void {
    const idx = pitWeather.findIndex(e => e.lap_start === entry.lap_start && e.race_id === entry.race_id);
    if (idx >= 0) pitWeather[idx] = entry;
    else pitWeather.push(entry);
    pitWeather.sort((a, b) => a.lap_start - b.lap_start);
    renderPitWeather();
}

function renderPitBoard(): void {
    const board = document.getElementById('pit-board')!;
    if (pitStandings.length) {
        const sorted = [...pitStandings].sort((a, b) => a.position - b.position);
        board.innerHTML = sorted.map(r => {
            const heatDots = Array(3).fill(0).map(() => '<div class="pit-heat-dot"></div>').join('');
            return `<div class="pit-driver-card pos${Math.min(r.position, 3)}">
            <div class="d-flex justify-content-between align-items-start">
                <div class="pit-pos">#${r.position}</div>
                <div class="d-flex gap-2">
                    <span class="pit-heat">${heatDots}</span>
                    <span class="pit-turbo"><i class="fa-solid fa-bolt"></i></span>
                </div>
            </div>
            <div class="pit-name mt-1" style="color:${r.car_color}">${r.name}</div>
            <small class="opacity-50">${r.gap}</small>
            <div class="row g-2 mt-2">
                <div class="col-6"><div class="pit-stat"><div class="value">${r.lap}</div><div class="label">Lap</div></div></div>
                <div class="col-6"><div class="pit-stat"><div class="value">${r.gap}</div><div class="label">Gap</div></div></div>
            </div>
        </div>`;
        }).join('');
        return;
    }
    const sorted = [...pitRacers].sort((a, b) => a.position - b.position);
    board.innerHTML = sorted.map(r => {
        const heatDots = Array(3).fill(0).map(() => '<div class="pit-heat-dot"></div>').join('');
        return `<div class="pit-driver-card pos${Math.min(r.position, 3)}">
            <div class="d-flex justify-content-between align-items-start">
                <div class="pit-pos">#${r.position}</div>
                <div class="d-flex gap-2">
                    <span class="pit-heat">${heatDots}</span>
                    <span class="pit-turbo"><i class="fa-solid fa-bolt"></i></span>
                </div>
            </div>
            <div class="pit-name mt-1" style="color:${r.car_color}">${r.name}</div>
            <small class="opacity-50">${r.car_name}</small>
            <div class="row g-2 mt-2">
                <div class="col-6"><div class="pit-stat"><div class="value">${r.points}</div><div class="label">Points</div></div></div>
                <div class="col-6"><div class="pit-stat"><div class="value">#${r.rank}</div><div class="label">Rank</div></div></div>
            </div>
        </div>`;
    }).join('');
}

async function refreshPitStatus(): Promise<void> {
    const racesRes = await fetch('/api/racers');
    pitRacers = await racesRes.json();
    renderPitBoard();
}

function connectPitWebSocket(): void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    pitSocket = connectWithRetry(`${protocol}//${window.location.host}/ws`, {
        topics: ['flags', 'racers', 'commentary', 'weather', 'race_state', 'standings', 'game_mechanics', 'sound', 'lap_replay'],
        onMessage: (msg) => {
            if (msg.type === 'racers') {
                pitRacers = msg.payload;
                renderPitBoard();
                renderPitWeather();
            } else if (msg.type === 'flag') {
                const flagNames: Record<string, string> = { green: '🏁 Green Flag', yellow: '💛 Yellow Flag', red: '🛑 Red Flag', chequered: '🏁 Chequered Flag', safety: '🚗 Safety Car', blue: '🔵 Blue Flag', blackwhite: '🏳️ Black & White Flag' };
                document.getElementById('pit-flag-status')!.textContent = flagNames[msg.payload.flag] || msg.payload.flag;
            } else if (msg.type === 'weather_update') {
                upsertPitWeather(msg.payload as WeatherEntry);
            } else if (msg.type === 'race_state') {
                applyPitRaceState(msg.payload);
            } else if (msg.type === 'standings') {
                applyPitStandings(msg.payload);
            } else if (msg.type === 'race_radio') {
                const radio = msg.payload;
                if (radio) showPitRadio(`${radio.racer_name || 'Race control'}: ${radio.message}`);
            } else if (msg.type === 'hello' || msg.type === 'resync') {
                if (msg.snapshot?.racers) {
                    pitRacers = msg.snapshot.racers;
                    renderPitBoard();
                    renderPitWeather();
                }
                if (msg.snapshot?.weather) {
                    const w = msg.snapshot.weather;
                    if (Array.isArray(w)) w.forEach((e: WeatherEntry) => upsertPitWeather(e));
                    else upsertPitWeather(w as WeatherEntry);
                }
                if (msg.snapshot?.race_state) applyPitRaceState(msg.snapshot.race_state);
                if (msg.snapshot?.standings) applyPitStandings(msg.snapshot.standings);
            }
        },
    });
}

function pitToggleFullscreen(): void {
    if (!document.fullscreenElement) {
        document.documentElement.requestFullscreen();
    } else {
        document.exitFullscreen();
    }
}

loadPitBoard();
