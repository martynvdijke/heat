import './theme';
interface PlayerRacer {
    id: number; name: string; profile_picture: string; car_color: string;
    car_name: string; points: number; rank: number; position: number;
}

let playerToken = '';
let playerRacerId = 0;
let playerRacerName = '';
let playerCarColor = '';
let playerCurrentLap = 1;
let playerWs: WebSocket | null = null;

async function playerLoadRacers(): Promise<void> {
    const res = await fetch('/api/racers');
    const racers: PlayerRacer[] = await res.json();
    const select = document.getElementById('player-select') as HTMLSelectElement;
    racers.forEach(r => {
        const opt = document.createElement('option');
        opt.value = r.id.toString();
        opt.textContent = `${r.name} (${r.car_name})`;
        select.appendChild(opt);
    });
}

async function playerLogin(): Promise<void> {
    const select = document.getElementById('player-select') as HTMLSelectElement;
    const deviceName = (document.getElementById('device-name') as HTMLInputElement).value;
    const racerId = parseInt(select.value);
    if (!racerId) { alert('Please select a driver'); return; }

    const res = await fetch('/api/player/login', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ racer_id: racerId, device_name: deviceName || 'Phone' })
    });
    if (!res.ok) { alert('Login failed'); return; }
    const data = await res.json();
    playerToken = data.token;
    playerRacerId = data.racer_id;
    playerRacerName = data.racer_name;
    localStorage.setItem('player_token', playerToken);
    localStorage.setItem('player_racer_id', playerRacerId.toString());
    showDashboard();
    connectWebSocket();
}

function showDashboard(): void {
    document.getElementById('login-screen')!.style.display = 'none';
    document.getElementById('dashboard-screen')!.style.display = 'block';
    document.getElementById('my-name')!.textContent = playerRacerName;

    // Get car color
    fetch(`/api/racers`).then(r => r.json()).then((racers: PlayerRacer[]) => {
        const me = racers.find(r => r.id === playerRacerId);
        if (me) {
            playerCarColor = me.car_color;
            document.getElementById('my-color')!.style.background = me.car_color;
        }
    });
    refreshStatus();
}

function playerLogout(): void {
    fetch('/api/player/logout', { method: 'POST', headers: { 'X-Player-Token': playerToken } });
    localStorage.removeItem('player_token');
    localStorage.removeItem('player_racer_id');
    playerToken = '';
    if (playerWs) playerWs.close();
    document.getElementById('login-screen')!.style.display = 'block';
    document.getElementById('dashboard-screen')!.style.display = 'none';
}

async function refreshStatus(): Promise<void> {
    if (!playerToken) return;
    const res = await fetch('/api/player/status', { headers: { 'X-Player-Token': playerToken } });
    if (!res.ok) return;
    const data = await res.json();
    const r = data.racer;
    document.getElementById('my-pos')!.textContent = r.position;
    document.getElementById('hand-count')!.textContent = data.heat_cards.hand;
    document.getElementById('deck-count')!.textContent = data.heat_cards.deck;
    document.getElementById('discard-count')!.textContent = data.heat_cards.discard;
    document.getElementById('engine-count')!.textContent = data.heat_cards.engine;

    // Render hand cards visually
    const handDiv = document.getElementById('hand-cards')!;
    const handCount = data.heat_cards.hand;
    handDiv.innerHTML = Array(handCount).fill(0).map(() =>
        `<div class="heat-card">H</div>`
    ).join('') || '<small class="opacity-50">No cards in hand</small>';
}

// Heat Card Actions
async function addHeatToEngine(): Promise<void> {
    await fetch('/api/player/heat', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: playerToken, card_type: 'heat', location: 'engine', count: 1 })
    });
    refreshStatus();
}

async function addStress(): Promise<void> {
    await fetch('/api/player/heat', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: playerToken, card_type: 'stress', location: 'engine', count: 1 })
    });
    refreshStatus();
}

async function discardHeat(): Promise<void> {
    await fetch('/api/player/heat', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: playerToken, card_type: 'heat', location: 'discard', count: 1 })
    });
    refreshStatus();
}

async function drawCard(): Promise<void> {
    await fetch('/api/player/heat', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: playerToken, card_type: 'heat', location: 'hand', count: 1 })
    });
    refreshStatus();
}

// Gear
async function reportGear(gear: number): Promise<void> {
    document.querySelectorAll('.gear-btn').forEach(b => b.classList.remove('active', 'btn-warning'));
    const btn = document.querySelector(`.gear-btn[data-gear="${gear}"]`)!;
    btn.classList.add('active', 'btn-warning');

    await fetch('/api/player/gear', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: playerToken, lap: playerCurrentLap, gear, stress: 0 })
    });
    addLapEntry(`Gear ${gear}`, 'gear');
}

async function reportStress(amount: number): Promise<void> {
    await fetch('/api/player/gear', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: playerToken, lap: playerCurrentLap, gear: 0, stress: amount })
    });
    addLapEntry(`+${amount} Stress`, 'stress');
}

// Turbo
async function useTurbo(): Promise<void> {
    await fetch('/api/player/turbo', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: playerToken, lap: playerCurrentLap })
    });
    const btn = document.getElementById('turbo-btn') as HTMLButtonElement;
    btn.classList.add('used');
    btn.disabled = true;
    document.getElementById('turbo-count')!.textContent = 'Used';
    addLapEntry('Turbo used!', 'turbo');
}

// Lap History
function addLapEntry(text: string, type: string): void {
    const history = document.getElementById('my-lap-history')!;
    if (history.querySelector('.opacity-50')) history.innerHTML = '';
    const entry = document.createElement('div');
    entry.className = 'd-flex justify-content-between align-items-center py-1 border-bottom border-secondary border-opacity-25';
    entry.innerHTML = `<span><small>Lap ${playerCurrentLap}</small></span><span class="badge bg-${type === 'gear' ? 'info' : type === 'stress' ? 'warning' : 'success'}">${text}</span>`;
    history.prepend(entry);
}

// WebSocket for live updates
function connectWebSocket(): void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    playerWs = new WebSocket(`${protocol}//${window.location.host}/ws`);

    playerWs.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (Array.isArray(data)) {
            // Racer update
            const me = data.find((r: PlayerRacer) => r.id === playerRacerId);
            if (me) {
                document.getElementById('my-pos')!.textContent = me.position;
                document.getElementById('my-lap')!.textContent = me.position;
                playerCarColor = me.car_color;
            }
        } else if (data.type === 'self_service' && data.racer_id === playerRacerId) {
            // Our own action confirmed
        }
    };
}

// Auto-login check
const savedToken = localStorage.getItem('player_token');
const savedRacerId = localStorage.getItem('player_racer_id');
if (savedToken && savedRacerId) {
    fetch('/api/player/validate', { headers: { 'X-Player-Token': savedToken } })
        .then(res => {
            if (res.ok) {
                playerToken = savedToken;
                playerRacerId = parseInt(savedRacerId);
                return res.json();
            }
            throw new Error('Invalid token');
        })
        .then(data => {
            playerRacerName = data.racer_name;
            showDashboard();
            connectWebSocket();
        })
        .catch(() => {
            localStorage.removeItem('player_token');
            localStorage.removeItem('player_racer_id');
            playerLoadRacers();
        });
} else {
    playerLoadRacers();
}
