interface TVRacer {
    id: number; name: string; car_color: string; car_name: string;
    position: number; points: number; rank: number;
}

let tvRacers: TVRacer[] = [];
let tvWs: WebSocket | null = null;
let tvSeconds = 0;
let tvTimer: ReturnType<typeof setInterval> | null = null;
let currentFlag = 'green';

async function loadTVData(): Promise<void> {
    const [racersRes, raceRes, quoteRes] = await Promise.all([
        fetch('/api/racers'), fetch('/api/race-info'), fetch('/api/quote/random')
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

    renderLeaderboard();
    connectTVWebSocket();
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

function connectTVWebSocket(): void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    tvWs = new WebSocket(`${protocol}//${window.location.host}/ws`);

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
            playSound(data.sound || 'flag');
        }
    };
}

// Audio feedback
function tvPlaySound(sound: string): void {
    try {
        const ctx = new AudioContext();
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.connect(gain);
        gain.connect(ctx.destination);

        switch (sound) {
            case 'engine':
                osc.frequency.setValueAtTime(150, ctx.currentTime);
                osc.frequency.exponentialRampToValueAtTime(80, ctx.currentTime + 0.3);
                gain.gain.setValueAtTime(0.1, ctx.currentTime);
                gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.3);
                osc.start(); osc.stop(ctx.currentTime + 0.3);
                break;
            case 'finish':
                osc.frequency.setValueAtTime(440, ctx.currentTime);
                osc.frequency.setValueAtTime(554, ctx.currentTime + 0.15);
                osc.frequency.setValueAtTime(659, ctx.currentTime + 0.3);
                gain.gain.setValueAtTime(0.1, ctx.currentTime);
                gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.5);
                osc.start(); osc.stop(ctx.currentTime + 0.5);
                break;
            case 'flag':
                osc.type = 'square';
                osc.frequency.setValueAtTime(880, ctx.currentTime);
                osc.frequency.setValueAtTime(440, ctx.currentTime + 0.1);
                gain.gain.setValueAtTime(0.05, ctx.currentTime);
                gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.2);
                osc.start(); osc.stop(ctx.currentTime + 0.2);
                break;
            case 'crash':
                const noise = ctx.createBufferSource();
                const buf = ctx.createBuffer(1, ctx.sampleRate * 0.3, ctx.sampleRate);
                const data = buf.getChannelData(0);
                for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1;
                noise.buffer = buf;
                const ng = ctx.createGain();
                ng.gain.setValueAtTime(0.1, ctx.currentTime);
                ng.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.3);
                noise.connect(ng);
                ng.connect(ctx.destination);
                noise.start(); noise.stop(ctx.currentTime + 0.3);
                break;
        }
    } catch (e) {
        // Audio not available
    }
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
