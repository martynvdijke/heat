interface PitRacer {
    id: number; name: string; car_color: string; car_name: string;
    position: number; points: number; rank: number;
}

let pitRacers: PitRacer[] = [];
let pitWs: WebSocket | null = null;
let pitSeconds = 0;

async function loadPitBoard(): Promise<void> {
    const [racersRes, raceRes, weatherRes] = await Promise.all([
        fetch('/api/racers'), fetch('/api/race-info'), fetch('/api/weather?race_id=0')
    ]);
    pitRacers = await racersRes.json();
    const race = await raceRes.json();
    document.getElementById('pit-race-name')!.textContent = `${race.country} - ${race.track}`;
    renderPitBoard();

    const weather = await weatherRes.json();
    if (weather.length > 0) {
        const w = weather[weather.length - 1];
        const icons: Record<string, string> = { dry: '☀️', damp: '🌦️', wet: '🌧️', torrential: '⛈️' };
        document.getElementById('pit-weather-icon')!.textContent = icons[w.condition] || '☀️';
        document.getElementById('pit-weather-text')!.textContent = w.condition.charAt(0).toUpperCase() + w.condition.slice(1);
    }

    connectPitWebSocket();
    setInterval(refreshPitStatus, 5000);
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
            document.getElementById('pit-lap')!.textContent = `Lap ${Math.max(...data.map((r: PitRacer) => r.position), 0)}`;
        } else if (data.type === 'flag') {
            const flagNames: Record<string, string> = { green: '🏁 Green Flag', yellow: '💛 Yellow Flag', red: '🛑 Red Flag', chequered: '🏁 Chequered Flag', safety: '🚗 Safety Car', blue: '🔵 Blue Flag', blackwhite: '🏳️ Black & White Flag' };
            document.getElementById('pit-flag-status')!.textContent = flagNames[data.flag] || data.flag;
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
