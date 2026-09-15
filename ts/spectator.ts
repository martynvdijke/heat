import './theme';
import { CommentaryTicker } from './commentary';
import { WeatherEntry, getActiveWeather, getForecast, weatherIcon, weatherLabel } from './weather';
import { connectWithRetry, type HeatSocket } from './ws';
interface SpecRacer {
    id: number; name: string; car_color: string; car_name: string;
    position: number; points: number; rank: number;
}

let specRacers: SpecRacer[] = [];
let specSocket: HeatSocket | null = null;
let specCommentary: CommentaryTicker | null = null;
let specWeather: WeatherEntry[] = [];
let specLap = 0;
type Standing = { racer_id: number; name: string; car_color: string; position: number; lap: number; gap: string };
let specStandings: Standing[] = [];
function formatElapsed(ms: number): string {
    const totalSec = Math.floor(ms / 1000);
    const h = Math.floor(totalSec / 3600);
    const m = Math.floor((totalSec % 3600) / 60);
    const s = totalSec % 60;
    if (h > 0) return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}
function applySpecRaceState(payload: any): void {
    if (!payload) return;
    if (typeof payload.current_lap === 'number') specLap = payload.current_lap;
    const lapEl = document.getElementById('spec-lap');
    if (lapEl) lapEl.textContent = `Lap ${specLap}`;
    if (typeof payload.elapsed_ms === 'number') {
        const c = document.getElementById('spec-clock');
        if (c) c.textContent = formatElapsed(payload.elapsed_ms);
    }
    renderSpecWeather();
    renderSpecGrid();
}
function applySpecStandings(payload: any): void {
    if (Array.isArray(payload)) specStandings = payload as Standing[];
    else return;
    renderSpecGrid();
}

async function loadSpecState(): Promise<void> {
    const [stateRes, eventsRes] = await Promise.all([
        fetch('/api/spectator/state'), fetch('/api/race-events?race_id=0')
    ]);
    const state = await stateRes.json();
    specRacers = state.racers || [];
    const race = state.race || {};

    document.getElementById('spec-race-name')!.textContent = race.country || 'Race';
    document.getElementById('spec-race-track')!.textContent = `${race.track || 'Track'} (${race.laps || '?'} laps)`;

    renderSpecGrid();

    const events = await eventsRes.json();
    renderSpecEvents(events);

    // Load full weather history for active/forecast logic
    try {
        const wRes = await fetch('/api/weather?race_id=0');
        specWeather = await wRes.json();
    } catch { /* ignore */ }
    if (specWeather.length === 0 && state.weather) specWeather = [state.weather as WeatherEntry];
    renderSpecWeather();

    // Also poll weather periodically via existing refresh
    setInterval(async () => {
        try {
            const r = await fetch('/api/weather?race_id=0');
            specWeather = await r.json();
            renderSpecWeather();
        } catch { /* ignore */ }
    }, 5000);

    connectSpecWebSocket();

    const commentaryEl = document.getElementById('spec-commentary');
    if (commentaryEl) {
        specCommentary = new CommentaryTicker(commentaryEl);
        specCommentary.start();
    }
}

function renderSpecGrid(): void {
    const grid = document.getElementById('spec-grid')!;
    if (specStandings.length) {
        const sorted = [...specStandings].sort((a, b) => a.position - b.position);
        grid.innerHTML = sorted.map(r => `<div class="spec-card" style="border-left-color:${r.car_color}">
            <div class="d-flex justify-content-between">
                <div class="pos">P${r.position}</div>
                <span class="badge bg-dark" style="height:fit-content;">${r.gap}</span>
            </div>
            <div class="name" style="color:${r.car_color}">${r.name}</div>
            <div class="meta">${r.name}</div>
        </div>`).join('');
    } else {
        const sorted = [...specRacers].sort((a, b) => a.position - b.position);
        const maxPos = Math.max(...sorted.map(r => r.position), 1);
        grid.innerHTML = sorted.map(r => {
            return `<div class="spec-card" style="border-left-color:${r.car_color}">
            <div class="d-flex justify-content-between">
                <div class="pos">P${r.position}</div>
                <span class="badge bg-dark" style="height:fit-content;"></span>
            </div>
            <div class="name" style="color:${r.car_color}">${r.name}</div>
            <div class="meta">${r.car_name} · ${r.points} pts</div>
        </div>`;
        }).join('');
    }
    document.getElementById('spec-lap')!.textContent = `Lap ${specLap || 0}`;
}

function renderSpecWeather(): void {
    const active = getActiveWeather(specWeather, specLap);
    const forecast = getForecast(specWeather, specLap);
    const el = document.getElementById('spec-weather')!;
    if (active) el.textContent = `${weatherIcon(active.condition)} ${weatherLabel(active.condition)}`;
    const fc = document.getElementById('spec-weather-forecast')!;
    if (fc) {
        if (forecast) { fc.textContent = `${weatherIcon(forecast.condition)} ${weatherLabel(forecast.condition)} from lap ${forecast.lap_start}`; fc.hidden = false; }
        else fc.hidden = true;
    }
}

function renderSpecEvents(events: any[]): void {
    const container = document.getElementById('spec-events')!;
    if (!events || events.length === 0) {
        container.innerHTML = '<p class="text-center opacity-50">No events yet</p>';
        return;
    }
    const eventIcons: Record<string, string> = { overtake: '🏎️', crash: '💥', spin: '🔄', safety_car: '🚗', pit_stop: '🔧' };
    container.innerHTML = events.slice(-15).reverse().map((e: any) =>
        `<div class="d-flex justify-content-between py-1 border-bottom border-secondary border-opacity-25 small">
            <span>${eventIcons[e.event_type] || '📌'} ${e.event_type}</span>
            <small>Lap ${e.lap}</small>
        </div>`
    ).join('');
}

function upsertSpecWeather(entry: WeatherEntry): void {
    const idx = specWeather.findIndex(e => e.lap_start === entry.lap_start && e.race_id === entry.race_id);
    if (idx >= 0) specWeather[idx] = entry; else specWeather.push(entry);
    specWeather.sort((a, b) => a.lap_start - b.lap_start);
    renderSpecWeather();
}

function connectSpecWebSocket(): void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    specSocket = connectWithRetry(`${protocol}//${window.location.host}/ws`, {
        topics: ['flags', 'racers', 'commentary', 'weather', 'race_state', 'standings', 'game_mechanics', 'sound', 'lap_replay'],
        onMessage: (msg) => {
            if (msg.type === 'racers') {
                specRacers = msg.payload;
                renderSpecGrid();
                document.getElementById('spec-status-indicator')!.textContent = 'RACING';
                document.getElementById('spec-status-indicator')!.className = 'spec-status racing';
                renderSpecWeather();
            } else if (msg.type === 'flag') {
                const statusEl = document.getElementById('spec-status-indicator')!;
                const flag = msg.payload.flag;
                if (flag === 'chequered') {
                    statusEl.textContent = 'FINISHED';
                    statusEl.className = 'spec-status stopped';
                } else if (flag === 'red') {
                    statusEl.textContent = 'RED FLAG';
                    statusEl.className = 'spec-status stopped';
                } else {
                    statusEl.textContent = flag.toUpperCase();
                    statusEl.className = 'spec-status racing';
                }
            } else if (msg.type === 'weather_update') {
                upsertSpecWeather(msg.payload as WeatherEntry);
            } else if (msg.type === 'commentary') {
                specCommentary?.handleEnvelope(msg);
            } else if (msg.type === 'race_state') {
                applySpecRaceState(msg.payload);
            } else if (msg.type === 'standings') {
                applySpecStandings(msg.payload);
            } else if (msg.type === 'hello' || msg.type === 'resync') {
                if (msg.snapshot?.racers) {
                    specRacers = msg.snapshot.racers;
                    renderSpecGrid();
                    renderSpecWeather();
                }
                if (msg.snapshot?.weather) {
                    const w = msg.snapshot.weather;
                    if (Array.isArray(w)) w.forEach((e: WeatherEntry) => upsertSpecWeather(e));
                    else upsertSpecWeather(w as WeatherEntry);
                }
                if (msg.snapshot?.race_state) applySpecRaceState(msg.snapshot.race_state);
                if (msg.snapshot?.standings) applySpecStandings(msg.snapshot.standings);
            }
        },
    });
}

loadSpecState();
