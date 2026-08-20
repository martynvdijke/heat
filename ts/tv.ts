import './theme';
import { playCategory } from './sound';
import './sound-settings';
import { CommentaryTicker } from './commentary';
import { WeatherEntry, getActiveWeather, getForecast, weatherIcon, weatherLabel, formatGrip } from './weather';
interface TVRacer {
    id: number; name: string; car_color: string; car_name: string;
    position: number; points: number; rank: number;
}

let tvRacers: TVRacer[] = [];
let tvWs: WebSocket | null = null;
let tvSeconds = 0;
let tvTimer: ReturnType<typeof setInterval> | null = null;
let currentFlag = 'green';
let tvCommentary: CommentaryTicker | null = null;
let weatherEntries: WeatherEntry[] = [];

async function loadTVData(): Promise<void> {
    const [racersRes, raceRes, quoteRes, weatherRes] = await Promise.all([
        fetch('/api/racers'), fetch('/api/race-info'), fetch('/api/quote/random'), fetch('/api/weather?race_id=0')
    ]);
    tvRacers = await racersRes.json();
    const race = await raceRes.json();

    document.getElementById('tv-race-name')!.textContent = race.country || 'Race';
    document.getElementById('tv-race-track')!.textContent = `${race.track || 'Track'} • ${race.laps || 0} Laps`;

    const quote = await quoteRes.json();
    const quoteEl = document.getElementById('tv-quote')!;
    if (quote && quote.text) {
        quoteEl.textContent = `"${quote.text}" — ${quote.author || 'Commentator'}`;
    }

    weatherEntries = await weatherRes.json();
    renderWeather();

    renderLeaderboard();
    connectTVWebSocket();

    const commentaryEl = document.getElementById('tv-commentary');
    if (commentaryEl) {
        tvCommentary = new CommentaryTicker(commentaryEl);
        tvCommentary.start();
    }
}

function renderWeather(): void {
    const panel = document.getElementById('tv-weather-panel');
    if (!panel) return;
    const active = getActiveWeather(weatherEntries, currentTVLap);
    const forecast = getForecast(weatherEntries, currentTVLap);

    // Color-code the banner by the active condition.
    panel.classList.remove('weather-dry', 'weather-damp', 'weather-wet', 'weather-torrential');
    if (active) {
        panel.classList.add(`weather-${active.condition}`);
        document.getElementById('tv-weather-icon')!.textContent = weatherIcon(active.condition);
        document.getElementById('tv-weather-text')!.textContent = weatherLabel(active.condition);
        const gripEl = document.getElementById('tv-weather-grip')!;
        gripEl.textContent = active.grip_modifier ? ` · ${formatGrip(active.grip_modifier)}` : '';
    }

    const forecastEl = document.getElementById('tv-weather-forecast')!;
    if (forecast) {
        forecastEl.textContent = `${weatherIcon(forecast.condition)} ${weatherLabel(forecast.condition)} from lap ${forecast.lap_start}`;
        forecastEl.hidden = false;
    } else {
        forecastEl.hidden = true;
    }
}

function renderLeaderboard(): void {
    const sorted = [...tvRacers].sort((a, b) => a.position - b.position);
    const board = document.getElementById('tv-leaderboard')!;

    const s = (n: number) => { const d = Math.floor(n / 86400); const h = Math.floor((n % 86400) / 3600); const m = Math.floor((n % 3600) / 60); const sec = n % 60; return d > 0 ? `${d}d ${h}h` : h > 0 ? `${h}:${m.toString().padStart(2, '0')}:${sec.toString().padStart(2, '0')}` : `${m.toString().padStart(2, '0')}:${sec.toString().padStart(2, '0')}`; };

    board.innerHTML = sorted.map((r, i) => {
        const gap = i === 0 ? 'LEAD' : `+${s(Math.floor(Math.random() * 30) + 1)}`;
        return `<div class="tv-driver-row pos${Math.min(r.position, 3)}" style="animation-delay:${i * 0.05}s">
            <div class="tv-position">${r.position}</div>
            <span class="tv-car-color" style="background:${r.car_color}"></span>
            <div class="tv-name">${r.name}</div>
            <div class="tv-gap">${gap}</div>
        </div>`;
    }).join('');

    document.getElementById('tv-lap')!.textContent = String(currentTVLap).padStart(2, '0');
}

let currentTVLap = 1;

function upsertWeather(entry: WeatherEntry): void {
    const idx = weatherEntries.findIndex(e => e.lap_start === entry.lap_start && e.race_id === entry.race_id);
    if (idx >= 0) weatherEntries[idx] = entry;
    else weatherEntries.push(entry);
    weatherEntries.sort((a, b) => a.lap_start - b.lap_start);
    renderWeather();
}

function connectTVWebSocket(): void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    tvWs = new WebSocket(`${protocol}//${window.location.host}/ws`);
    tvCommentary?.connect(tvWs);

    tvWs.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (Array.isArray(data)) {
            tvRacers = data;
            renderLeaderboard();
        } else if (data.type === 'flag') {
            handleTVFlag(data);
            tvPlaySound('flag');
        } else if (data.type === 'self_service') {
            updateEvent(`⚡ ${data.racer_name || `Racer #${data.racer_id}`} used turbo`);
        } else if (data.type === 'sound') {
            tvPlaySound(data.sound || 'flag');
        } else if (data.type === 'weather_update') {
            upsertWeather(data as WeatherEntry);
        }
    };
}

// Audio feedback - routed through the customizable sound module
function tvPlaySound(sound: string): void {
    playCategory(sound as 'engine' | 'horn' | 'finish' | 'crash' | 'flag');
}

function handleTVFlag(data: any): void {
    currentFlag = data.flag || 'green';
    const flagIcons: Record<string, string> = {
        green: '🏁', yellow: '💛', red: '🛑', chequered: '🏁',
        safety: '🚗', blue: '🔵', blackwhite: '🏳️'
    };
    const flagNames: Record<string, string> = {
        green: 'Green Flag', yellow: 'Yellow Flag', red: 'Red Flag',
        chequered: 'Chequered Flag', safety: 'Safety Car',
        blue: 'Blue Flag', blackwhite: 'Black & White Flag'
    };

    document.getElementById('tv-flag-icon')!.textContent = flagIcons[data.flag] || '🏁';
    document.getElementById('tv-flag-text')!.textContent = flagNames[data.flag] || data.flag;

    const el = document.getElementById('tv-flag-display')!;
    if (data.flag === 'red' || data.flag === 'yellow') {
        el.classList.add('flag-active');
    } else {
        el.classList.remove('flag-active');
    }

    if (data.flag === 'chequered') {
        updateEvent('🏁 CHEQUERED FLAG! Race finished!');
    }
}

function updateEvent(text: string): void {
    const container = document.getElementById('tv-events')!;
    const first = container.querySelector('.text-center');
    if (first) first.remove();

    const el = document.createElement('div');
    el.className = 'tv-event';
    const now = new Date();
    el.innerHTML = `<span class="event-time">${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}</span> ${text}`;
    container.prepend(el);

    while (container.children.length > 10) {
        container.removeChild(container.lastChild!);
    }
}

function tvToggleFullscreen(): void {
    if (!document.fullscreenElement) {
        document.documentElement.requestFullscreen();
    } else {
        document.exitFullscreen();
    }
}

// Clock
tvTimer = setInterval(() => {
    tvSeconds++;
    const m = Math.floor(tvSeconds / 60);
    const s = tvSeconds % 60;
    document.getElementById('tv-clock')!.textContent = `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}, 1000);

loadTVData();
