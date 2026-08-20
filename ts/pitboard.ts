import './theme';
import { WeatherEntry, getActiveWeather, getForecast, weatherIcon, weatherLabel, formatGrip } from './weather';
interface PitRacer {
    id: number; name: string; car_color: string; car_name: string;
    position: number; points: number; rank: number;
}

let pitRacers: PitRacer[] = [];
let pitWs: WebSocket | null = null;
let pitSeconds = 0;
let pitWeather: WeatherEntry[] = [];
let pitLap = 0;

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
    const sorted = [...pitRacers].sort((a, b) => a.position - b.position);
    const board = document.getElementById('pit-board')!;
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
    pitSeconds++;
    const m = Math.floor(pitSeconds / 60);
    const s = pitSeconds % 60;
    document.getElementById('pit-clock')!.textContent = `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

function connectPitWebSocket(): void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    pitWs = new WebSocket(`${protocol}//${window.location.host}/ws`);
    pitWs.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (Array.isArray(data)) {
            pitRacers = data;
            renderPitBoard();
            pitLap = Math.max(...data.map((r: PitRacer) => r.position), pitLap || 0);
            document.getElementById('pit-lap')!.textContent = `Lap ${pitLap}`;
            renderPitWeather();
        } else if (data.type === 'flag') {
            const flagNames: Record<string, string> = { green: '🏁 Green Flag', yellow: '💛 Yellow Flag', red: '🛑 Red Flag', chequered: '🏁 Chequered Flag', safety: '🚗 Safety Car', blue: '🔵 Blue Flag', blackwhite: '🏳️ Black & White Flag' };
            document.getElementById('pit-flag-status')!.textContent = flagNames[data.flag] || data.flag;
        } else if (data.type === 'weather_update') {
            upsertPitWeather(data as WeatherEntry);
        }
    };
}

function pitToggleFullscreen(): void {
    if (!document.fullscreenElement) {
        document.documentElement.requestFullscreen();
    } else {
        document.exitFullscreen();
    }
}

loadPitBoard();
