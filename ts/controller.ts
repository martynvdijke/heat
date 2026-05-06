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
        raceTimer = setInterval(() => {
            raceSeconds++;
            const mins = Math.floor(raceSeconds / 60);
            const secs = raceSeconds % 60;
            document.getElementById('race-timer')!.textContent =
                `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
        }, 1000);
    }
}

function pauseRace(): void {
    if (raceState === 'racing') {
        raceState = 'paused';
        clearInterval(raceTimer!);
        updateStatus();
    } else if (raceState === 'paused') {
        raceState = 'racing';
        raceTimer = setInterval(() => {
            raceSeconds++;
            const mins = Math.floor(raceSeconds / 60);
            const secs = raceSeconds % 60;
            document.getElementById('race-timer')!.textContent =
                `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
        }, 1000);
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
    const r = controllerRacers.find(r => r.id === id);
    if (r && r.position > 1) {
        r.position--;
        const other = controllerRacers.find(r => r.position === r.position);
        if (other) other.position++;
        savePositions();
    }
}

function moveDown(id: number): void {
    const r = controllerRacers.find(r => r.id === id);
    if (r) {
        r.position++;
        const other = controllerRacers.find(r => r.position === r.position);
        if (other) other.position--;
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
    if (!id) { alert('Select a driver first'); return; }
    const name = select.options[select.selectedIndex]?.text || '';
    broadcastMessage({ type: 'flag', flag: 'blue', racer_id: id, racer_name: name });
}

function sendBlackWhiteFlag(): void {
    const select = document.getElementById('flag-driver-select') as HTMLSelectElement;
    const id = parseInt(select.value);
    if (!id) { alert('Select a driver first'); return; }
    const name = select.options[select.selectedIndex]?.text || '';
    broadcastMessage({ type: 'flag', flag: 'blackwhite', racer_id: id, racer_name: name });
}

function triggerBlueFlag(id: number, name: string): void {
    broadcastMessage({ type: 'flag', flag: 'blue', racer_id: id, racer_name: name });
}

function triggerBlackWhiteFlag(id: number, name: string): void {
    broadcastMessage({ type: 'flag', flag: 'blackwhite', racer_id: id, racer_name: name });
}

function sendCommentary(): void {
    const input = document.getElementById('commentary-input') as HTMLInputElement;
    if (input.value.trim()) {
        broadcastMessage({ type: 'commentary', text: input.value.trim() });
        input.value = '';
    }
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
        alert('Race saved to history!');
        stopRace();
    });
}

function discardRace(): void {
    if (confirm('Discard this race? All progress will be lost.')) {
        stopRace();
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

document.getElementById('commentary-input')!.addEventListener('keypress', (e: KeyboardEvent) => {
    if (e.key === 'Enter') sendCommentary();
});

loadControllerData();

