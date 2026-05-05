interface AdminRacer {
    id: number;
    name: string;
    profile_picture: string;
    car_color: string;
    car_name: string;
    points: number;
    rank: number;
    position: number;
}

interface AdminTrack {
    id: string;
    name: string;
    country: string;
    length_km: number;
    lap_record: string;
    use_map_image: boolean;
    map_image_url: string;
    refresh_geojson: boolean;
}

interface AdminStats {
    id: number;
    racer_id: number;
    races: number;
    wins: number;
    podiums: number;
    fastest_laps: number;
    dnf: number;
}

interface AdminHistory {
    id: number;
    name: string;
    race_date: string;
    country: string;
    track: string;
    track_id: string;
}

let adminRacers: AdminRacer[] = [];
let adminTracks: AdminTrack[] = [];
let allTracks: AdminTrack[] = [];
let qualificationOrder: AdminRacer[] = [];
let shuffleInterval: ReturnType<typeof setInterval> | null = null;
let racerStats: AdminStats[] = [];
declare const bootstrap: any;

const racerModal = new bootstrap.Modal(document.getElementById('racerModal')!);
const quoteModal = new bootstrap.Modal(document.getElementById('quoteModal')!);
const trackModal = new bootstrap.Modal(document.getElementById('trackModal')!);

async function init(): Promise<void> {
    (document.getElementById('archive-date') as HTMLInputElement).valueAsDate = new Date();
    await loadAdminTracks();
    await loadRaceInfo();
    await loadRacers();
    await loadAllTracks();
    await loadQuotes();
    await loadRacerStats();
    await loadNotificationSettings();
    await loadUmamiSettings();
    await loadHistory();
    await loadAISettings();
}

async function loadAdminTracks(): Promise<void> {
    try {
        const res = await fetch('/api/tracks');
        adminTracks = await res.json();
        const selector = document.getElementById('track-select') as HTMLSelectElement;
        selector.innerHTML = '<option value="">Choose a circuit...</option>' +
            adminTracks.map(t => `<option value="${t.id}">${t.country} - ${t.name}</option>`).join('');
    } catch (e) { console.error('Failed to load tracks', e); }
}

async function loadAllTracks(): Promise<void> {
    try {
        const res = await fetch('/api/tracks');
        allTracks = await res.json();
        renderTrackList();
    } catch (e) { console.error('Failed to load all tracks', e); }
}

function renderTrackList(): void {
    const list = document.getElementById('track-list')!;
    list.innerHTML = allTracks.map(t => `
        <tr>
            <td class="ps-4"><code>${escapeHtml(t.id)}</code></td>
            <td class="fw-bold">${escapeHtml(t.name)}</td>
            <td>${escapeHtml(t.country)}</td>
            <td>${t.length_km} km</td>
            <td>
                ${t.use_map_image ? '<span class="badge bg-info text-dark">Image Map</span>' : '<span class="badge bg-secondary">GeoJSON</span>'}
                ${t.refresh_geojson ? '<i class="fa-solid fa-sync fa-spin ms-1 text-success small" title="Live Refresh On"></i>' : ''}
            </td>
            <td class="text-end pe-4">
                <button class="btn btn-sm btn-outline-primary" onclick="editTrack('${t.id}')"><i class="fa-solid fa-pen"></i></button>
                <button class="btn btn-sm btn-outline-danger" onclick="deleteTrack('${t.id}')"><i class="fa-solid fa-trash"></i></button>
            </td>
        </tr>
    `).join('');
}

function applyTrackPreset(id: string): void {
    const t = adminTracks.find(x => x.id === id);
    if (!t) return;
    const form = document.getElementById('race-form') as HTMLFormElement;
    (form.elements.namedItem('country') as HTMLInputElement).value = t.country;
    (form.elements.namedItem('track') as HTMLInputElement).value = t.name;
    (form.elements.namedItem('track_id') as HTMLInputElement).value = t.id;
    (form.elements.namedItem('laps') as HTMLInputElement).value = '50';
}

async function loadRaceInfo(): Promise<void> {
    try {
        const res = await fetch('/api/race-info');
        const data = await res.json();
        const form = document.getElementById('race-form') as HTMLFormElement;
        (form.elements.namedItem('country') as HTMLInputElement).value = data.country || '';
        (form.elements.namedItem('track') as HTMLInputElement).value = data.track || '';
        (form.elements.namedItem('laps') as HTMLInputElement).value = data.laps || 0;
        (form.elements.namedItem('track_id') as HTMLInputElement).value = data.track_id || '';
    } catch (e) { console.error('Failed to load race info', e); }
}

async function loadRacers(): Promise<void> {
    try {
        const res = await fetch('/api/racers');
        adminRacers = await res.json();
        const list = document.getElementById('racer-list')!;
        list.innerHTML = adminRacers.map(r => `
            <tr>
                <td class="ps-4 fw-bold">#${r.rank}</td>
                <td>
                    <div class="d-flex align-items-center">
                        <img src="${r.profile_picture}" class="rounded-circle me-3" width="32" height="32" style="object-fit: cover" onerror="this.src='/static/images/helmet.svg'">
                        <div><div class="fw-bold">${escapeHtml(r.name)}</div></div>
                    </div>
                </td>
                <td><span class="color-dot bg-${r.car_color}"></span> ${escapeHtml(r.car_name)}</td>
                <td><span class="badge bg-dark">${r.points} pts</span></td>
                <td class="small text-muted">${r.position}</td>
                <td class="text-end pe-4">
                    <button class="btn btn-sm btn-outline-primary" onclick="editRacer(${r.id})"><i class="fa-solid fa-pen"></i></button>
                    <button class="btn btn-sm btn-outline-danger" onclick="deleteRacer(${r.id})"><i class="fa-solid fa-trash"></i></button>
                </td>
            </tr>
        `).join('');
    } catch (e) { console.error('Failed to load racers', e); }
}

interface Quote {
    id: number;
    text: string;
    author: string;
}

async function loadQuotes(): Promise<void> {
    try {
        const res = await fetch('/api/quotes');
        const quotes: Quote[] = await res.json();
        const list = document.getElementById('quote-list')!;
        list.innerHTML = quotes.map(q => `
            <tr>
                <td class="ps-4"><em>"${escapeHtml(q.text)}"</em></td>
                <td class="fw-bold">${escapeHtml(q.author)}</td>
                <td class="text-end pe-4">
                    <button class="btn btn-sm btn-outline-primary" onclick="editQuote(${q.id})"><i class="fa-solid fa-pen"></i></button>
                    <button class="btn btn-sm btn-outline-danger" onclick="deleteQuote(${q.id})"><i class="fa-solid fa-trash"></i></button>
                </td>
            </tr>
        `).join('');
    } catch (e) { console.error('Failed to load quotes', e); }
}

async function loadHistory(): Promise<void> {
    try {
        const res = await fetch('/api/race-history');
        const history: AdminHistory[] = await res.json();
        const list = document.getElementById('history-list')!;
        list.innerHTML = history.map(h => `
            <tr>
                <td class="ps-4 fw-bold">${escapeHtml(h.name || h.race_date)}</td>
                <td>${h.race_date}</td>
                <td>${escapeHtml(h.country)}</td>
                <td>${escapeHtml(h.track)}</td>
                <td class="text-end pe-4">
                    <button class="btn btn-sm btn-outline-danger" onclick="deleteHistory(${h.id})"><i class="fa-solid fa-trash"></i></button>
                </td>
            </tr>
        `).join('');
    } catch (e) { console.error('Failed to load history', e); }
}

async function loadRacerStats(): Promise<void> {
    try {
        const res = await fetch('/api/racer-stats');
        const data: AdminStats[] = await res.json();
        racerStats = data;
        renderStatsList();
    } catch (e) { console.error('Failed to load racer stats', e); }
}

async function loadUmamiSettings(): Promise<void> {
    try {
        const res = await fetch('/api/umami-settings');
        const data = await res.json();
        (document.getElementById('umami-url') as HTMLInputElement).value = data.url || '';
        (document.getElementById('umami-website-id') as HTMLInputElement).value = data.website_id || '';
        (document.getElementById('umami-enabled') as HTMLInputElement).checked = data.enabled;
    } catch (e) { console.error('Failed to load umami settings', e); }
}

document.getElementById('umami-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const data = {
        url: (document.getElementById('umami-url') as HTMLInputElement).value,
        website_id: (document.getElementById('umami-website-id') as HTMLInputElement).value,
        enabled: (document.getElementById('umami-enabled') as HTMLInputElement).checked
    };
    const res = await fetch('/api/umami-settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) alert('Analytics settings saved!');
});

async function loadNotificationSettings(): Promise<void> {
    try {
        const res = await fetch('/api/notification-settings');
        const data = await res.json();
        (document.getElementById('gotify-url') as HTMLInputElement).value = data.gotify_url || '';
        (document.getElementById('gotify-token') as HTMLInputElement).value = data.gotify_token || '';
        (document.getElementById('notify-winner') as HTMLInputElement).checked = data.notify_winner;
        (document.getElementById('notify-podium') as HTMLInputElement).checked = data.notify_podium;
        (document.getElementById('notify-race-start') as HTMLInputElement).checked = data.notify_race_start;
    } catch (e) { console.error('Failed to load notification settings', e); }
}

document.getElementById('notify-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const data = {
        gotify_url: (document.getElementById('gotify-url') as HTMLInputElement).value,
        gotify_token: (document.getElementById('gotify-token') as HTMLInputElement).value,
        notify_winner: (document.getElementById('notify-winner') as HTMLInputElement).checked,
        notify_podium: (document.getElementById('notify-podium') as HTMLInputElement).checked,
        notify_race_start: (document.getElementById('notify-race-start') as HTMLInputElement).checked
    };
    const res = await fetch('/api/notification-settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) alert('Notification settings saved!');
});

document.getElementById('notify-winner')!.addEventListener('change', saveNotifyToggle);
document.getElementById('notify-podium')!.addEventListener('change', saveNotifyToggle);
document.getElementById('notify-race-start')!.addEventListener('change', saveNotifyToggle);

async function saveNotifyToggle(): Promise<void> {
    const data = {
        gotify_url: (document.getElementById('gotify-url') as HTMLInputElement).value,
        gotify_token: (document.getElementById('gotify-token') as HTMLInputElement).value,
        notify_winner: (document.getElementById('notify-winner') as HTMLInputElement).checked,
        notify_podium: (document.getElementById('notify-podium') as HTMLInputElement).checked,
        notify_race_start: (document.getElementById('notify-race-start') as HTMLInputElement).checked
    };
    await fetch('/api/notification-settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
}

async function testNotification(this: HTMLButtonElement, event: MouseEvent): Promise<void> {
    const btn = event.target as HTMLButtonElement;
    btn.disabled = true;
    btn.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-1"></i> Sending...';
    try {
        const res = await fetch('/api/test-notification', { method: 'POST' });
        if (res.ok) {
            alert('Test notification sent successfully!');
        } else {
            const err = await res.text();
            alert('Failed to send: ' + err);
        }
    } catch (e: any) {
        alert('Error: ' + e.message);
    }
    btn.disabled = false;
    btn.innerHTML = '<i class="fa-solid fa-paper-plane me-1"></i> Send Test Notification';
}

function racerNameById(id: number): string {
    const r = adminRacers.find(r => r.id === id);
    return r ? r.name : `Racer #${id}`;
}

function renderStatsList(): void {
    const list = document.getElementById('stats-list')!;
    if (racerStats.length === 0) {
        list.innerHTML = '<tr><td colspan="7" class="text-center text-muted py-4">No stats yet. <button class="btn btn-sm btn-outline-primary ms-2" onclick="addStats()"><i class="fa-solid fa-plus me-1"></i>Create Stats</button></td></tr>';
        return;
    }
    list.innerHTML = racerStats.map(s => `
        <tr>
            <td class="ps-4 fw-bold">${racerNameById(s.racer_id)}</td>
            <td>${s.races}</td>
            <td><span class="badge bg-warning text-dark">${s.wins}</span></td>
            <td><span class="badge bg-secondary">${s.podiums}</span></td>
            <td><span class="badge bg-info">${s.fastest_laps}</span></td>
            <td><span class="badge bg-danger">${s.dnf}</span></td>
            <td class="text-end pe-4">
                <button class="btn btn-sm btn-outline-primary" onclick="editStats(${s.id}, ${s.racer_id}, ${s.races}, ${s.wins}, ${s.podiums}, ${s.fastest_laps}, ${s.dnf})"><i class="fa-solid fa-pen"></i></button>
            </td>
        </tr>
    `).join('');
}

function addStats(): void {
    const racerId = prompt('Racer ID (check Racers tab for ID):');
    if (!racerId) return;
    const races = prompt('Races:', '0');
    if (races === null) return;
    const wins = prompt('Wins:', '0');
    if (wins === null) return;
    const podiums = prompt('Podiums:', '0');
    if (podiums === null) return;
    const fastestLaps = prompt('Fastest Laps:', '0');
    if (fastestLaps === null) return;
    const dnf = prompt('DNF:', '0');
    if (dnf === null) return;

    fetch('/api/racer-stats', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            id: 0,
            racer_id: parseInt(racerId),
            races: parseInt(races),
            wins: parseInt(wins),
            podiums: parseInt(podiums),
            fastest_laps: parseInt(fastestLaps),
            dnf: parseInt(dnf)
        })
    }).then(() => loadRacerStats());
}

function editStats(id: number, racerId: number, races: number, wins: number, podiums: number, fastestLaps: number, dnf: number): void {
    const newRaces = prompt('Races:', String(races))!;
    if (newRaces === null) return;
    const newWins = prompt('Wins:', String(wins))!;
    if (newWins === null) return;
    const newPodiums = prompt('Podiums:', String(podiums))!;
    if (newPodiums === null) return;
    const newFastestLaps = prompt('Fastest Laps:', String(fastestLaps))!;
    if (newFastestLaps === null) return;
    const newDnf = prompt('DNF:', String(dnf))!;
    if (newDnf === null) return;

    fetch('/api/racer-stats', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            id: id,
            racer_id: racerId,
            races: parseInt(newRaces),
            wins: parseInt(newWins),
            podiums: parseInt(newPodiums),
            fastest_laps: parseInt(newFastestLaps),
            dnf: parseInt(newDnf)
        })
    }).then(() => loadRacerStats());
}

document.getElementById('race-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const data: Record<string, any> = {};
    for (const pair of new FormData(form)) {
        data[pair[0] as string] = pair[1];
    }
    data.laps = parseInt(data.laps);
    const res = await fetch('/api/race-info', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) alert('Race data updated!');
});

document.getElementById('racer-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const data: Record<string, any> = {};
    for (const pair of new FormData(form)) {
        data[pair[0] as string] = pair[1];
    }
    data.id = data.id ? parseInt(data.id) : 0;
    data.points = parseInt(data.points) || 0;
    data.rank = parseInt(data.rank) || 0;
    data.position = parseInt(data.position) || 0;
    if (!data.profile_picture) {
        data.profile_picture = '/static/images/helmet.svg';
    }

    const res = await fetch('/api/racers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) {
        racerModal.hide();
        loadRacers();
    } else {
        const err = await res.json();
        alert('Failed to save racer: ' + (err.error || 'Unknown error'));
    }
});

document.getElementById('quote-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const data: Record<string, any> = {};
    for (const pair of new FormData(form)) {
        data[pair[0] as string] = pair[1];
    }
    data.id = data.id ? parseInt(data.id) : 0;
    const method = data.id ? 'PUT' : 'POST';

    const res = await fetch('/api/quotes', {
        method: method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) {
        quoteModal.hide();
        loadQuotes();
    }
});

document.getElementById('track-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const data: Record<string, any> = {};
    for (const pair of new FormData(form)) {
        data[pair[0] as string] = pair[1];
    }
    data.id = (document.getElementById('track-id-visible') as HTMLInputElement).value;
    data.length_km = parseInt(data.length_km);
    data.use_map_image = (document.getElementById('use-map-image') as HTMLInputElement).checked;
    data.refresh_geojson = (document.getElementById('refresh-geojson') as HTMLInputElement).checked;

    const res = await fetch('/api/tracks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) {
        trackModal.hide();
        loadAdminTracks();
        loadAllTracks();
    }
});

let lockedPositions = 0;

async function startQualification(): Promise<void> {
    const btn = document.querySelector('[onclick="startQualification()"]') as HTMLButtonElement;

    if (shuffleInterval) {
        clearInterval(shuffleInterval);
        shuffleInterval = null;
        btn.innerHTML = '<i class="fa-solid fa-play me-1"></i> Start Shuffle';
        return;
    }

    const speed = parseInt((document.getElementById('shuffle-speed') as HTMLInputElement).value);
    const gridDiv = document.getElementById('qualification-grid')!;
    const finalDiv = document.getElementById('final-grid')!;

    const res = await fetch('/api/racers');
    const currentRacers: AdminRacer[] = await res.json();

    if (currentRacers.length === 0) {
        alert('No racers available for qualification!');
        return;
    }

    qualificationOrder = [...currentRacers].sort(() => Math.random() - 0.5);
    lockedPositions = 0;
    finalDiv.innerHTML = '<div class="text-center text-muted py-4"><i class="fa-solid fa-spinner fa-spin fa-2x mb-2"></i><p class="small mb-0">Shuffling positions...</p></div>';
    btn.innerHTML = '<i class="fa-solid fa-stop me-1"></i> Stop Shuffle';

    let shuffleCount = 0;
    shuffleInterval = setInterval(() => {
        for (let i = qualificationOrder.length - 1; i >= lockedPositions; i--) {
            const j = Math.floor(Math.random() * (i - lockedPositions + 1)) + lockedPositions;
            [qualificationOrder[i], qualificationOrder[j]] = [qualificationOrder[j], qualificationOrder[i]];
        }

        renderQualificationGrid(qualificationOrder, lockedPositions);
        shuffleCount++;

        if (shuffleCount % 5 === 0 && lockedPositions < qualificationOrder.length) {
            lockedPositions++;
            setTimeout(() => {
                if (shuffleInterval) renderQualificationGrid(qualificationOrder, lockedPositions);
            }, 50);
        }

        if (lockedPositions >= qualificationOrder.length) {
            clearInterval(shuffleInterval!);
            shuffleInterval = null;
            btn.innerHTML = '<i class="fa-solid fa-play me-1"></i> Start Shuffle';
            renderFinalGrid(qualificationOrder);
        }
    }, speed);
}

function renderQualificationGrid(order: AdminRacer[], locked: number): void {
    const gridDiv = document.getElementById('qualification-grid')!;
    const cols = parseInt((document.getElementById('grid-layout') as HTMLSelectElement).value);

    gridDiv.innerHTML = order.map((r, i) => {
        const isLocked = i < locked;
        const animationClass = !isLocked ? 'shuffle-animation' : 'locked-position';
        return `
            <div class="col-md-${12/cols}">
                <div class="qualification-card ${animationClass}" ${!isLocked ? 'style="animation-delay: ' + (i * 50) + 'ms"' : ''}>
                    <div class="d-flex align-items-center p-2 border rounded ${isLocked ? 'bg-success' : 'bg-dark'} text-white">
                        <span class="badge ${isLocked ? 'bg-warning' : 'bg-secondary'} text-dark me-2">
                            ${isLocked ? '<i class="fa-solid fa-lock me-1"></i>' : ''}P${i + 1}
                        </span>
                        <img src="${r.profile_picture}" class="rounded-circle me-2" width="32" height="32" style="object-fit: cover" onerror="this.src='/static/images/helmet.svg'">
                        <div class="flex-grow-1">
                            <div class="fw-bold small">${escapeHtml(r.name)}</div>
                            <div class="small opacity-75"><span class="color-dot bg-${r.car_color}"></span>${escapeHtml(r.car_name)}</div>
                        </div>
                        ${isLocked ? '<i class="fa-solid fa-check text-warning"></i>' : '<i class="fa-solid fa-arrows-rotate fa-spin text-muted"></i>'}
                    </div>
                </div>
            </div>
        `;
    }).join('') + `
        <div class="col-12 text-center mt-3">
            <div class="progress" style="height: 8px;">
                <div class="progress-bar bg-success" role="progressbar" style="width: ${Math.round((locked / order.length) * 100)}%"></div>
                <div class="progress-bar bg-warning progress-bar-striped progress-bar-animated" role="progressbar" style="width: ${Math.round(((order.length - locked) / order.length) * 100)}%"></div>
            </div>
            <small class="text-muted">Locked: ${locked}/${order.length} | Remaining: ${order.length - locked}</small>
        </div>
    `;
}

document.getElementById('grid-layout')!.addEventListener('change', function(this: HTMLSelectElement) {
    if (qualificationOrder.length > 0 && !shuffleInterval) {
        renderFinalGrid(qualificationOrder);
    }
});

function renderFinalGrid(order: AdminRacer[]): void {
    const finalDiv = document.getElementById('final-grid')!;
    qualificationOrder = order;

    finalDiv.innerHTML = order.map((r, i) => `
        <div class="d-flex align-items-center p-2 mb-2 border rounded ${i === 0 ? 'border-warning bg-warning bg-opacity-10' : ''}" style="animation: lockFlash 0.5s ease-in-out ${i * 0.1}s">
            <span class="badge ${i === 0 ? 'bg-warning text-dark' : 'bg-secondary'} me-2" style="width: 40px">P${i + 1}</span>
            <img src="${r.profile_picture}" class="rounded-circle me-2" width="32" height="32" style="object-fit: cover" onerror="this.src='/static/images/helmet.svg'">
            <div class="flex-grow-1">
                <div class="fw-bold small">${escapeHtml(r.name)}</div>
                <div class="small text-muted"><span class="color-dot bg-${r.car_color}"></span>${escapeHtml(r.car_name)}</div>
            </div>
            ${i === 0 ? '<i class="fa-solid fa-crown text-warning"></i>' : ''}
        </div>
    `).join('');
}

async function applyGridPositions(): Promise<void> {
    if (qualificationOrder.length === 0) {
        alert('Please run qualification first!');
        return;
    }

    if (!confirm('Apply this grid order to the race? This will update racer ranks.')) return;

    for (let i = 0; i < qualificationOrder.length; i++) {
        const racer = qualificationOrder[i];
        const newRank = i + 1;

        await fetch('/api/racers', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                id: racer.id,
                name: racer.name,
                profile_picture: racer.profile_picture,
                car_name: racer.car_name,
                car_color: racer.car_color,
                points: racer.points,
                rank: newRank,
                position: racer.position
            })
        });
    }

    alert('Grid positions applied!');
    loadRacers();
}

function resetQualification(): void {
    if (shuffleInterval) {
        clearInterval(shuffleInterval);
        shuffleInterval = null;
        const btn = document.querySelector('[onclick="startQualification()"]') as HTMLButtonElement;
        btn.innerHTML = '<i class="fa-solid fa-play me-1"></i> Start Shuffle';
    }
    qualificationOrder = [];
    lockedPositions = 0;
    document.getElementById('qualification-grid')!.innerHTML = `
        <div class="col-12 text-center text-muted py-5">
            <i class="fa-solid fa-shuffle fa-3x mb-3"></i>
            <p>Click "Start Shuffle" to begin qualification</p>
        </div>
    `;
    document.getElementById('final-grid')!.innerHTML = `
        <div class="text-center text-muted py-4">
            <i class="fa-solid fa-trophy fa-2x mb-2"></i>
            <p class="small mb-0">Final grid order will appear here</p>
        </div>
    `;
}

function previewGrid(): void {
    const cols = parseInt((document.getElementById('grid-layout') as HTMLSelectElement).value);
    const gridDiv = document.getElementById('qualification-grid')!;

    if (qualificationOrder.length === 0) {
        alert('Please run qualification first!');
        return;
    }

    gridDiv.innerHTML = qualificationOrder.map((r, i) => `
        <div class="col-md-${12/cols}">
            <div class="border rounded p-3 text-center ${i === 0 ? 'bg-warning bg-opacity-10 border-warning' : ''}">
                <div class="mb-2">
                    <span class="badge ${i === 0 ? 'bg-warning text-dark' : 'bg-secondary'} fs-6">P${i + 1}</span>
                </div>
                <img src="${r.profile_picture}" class="rounded-circle mb-2" width="64" height="64" style="object-fit: cover" onerror="this.src='https://via.placeholder.com/64'">
                <div class="fw-bold">${escapeHtml(r.name)}</div>
                <div class="small text-muted"><span class="color-dot bg-${r.car_color}"></span>${escapeHtml(r.car_name)}</div>
                ${i === 0 ? '<i class="fa-solid fa-crown text-warning fa-2x mt-2"></i>' : ''}
            </div>
        </div>
    `).join('');
}

async function archiveRace(): Promise<void> {
    const name = (document.getElementById('archive-name') as HTMLInputElement).value;
    const date = (document.getElementById('archive-date') as HTMLInputElement).value;
    if (!name) { alert('Please provide a name for the archive'); return; }
    if (!confirm(`Archive current race as "${name}"?`)) return;

    const [race, racerData] = await Promise.all([
        fetch('/api/race-info').then(r => r.json()),
        fetch('/api/racers').then(r => r.json())
    ]);

    const results = (racerData as AdminRacer[]).map(r => ({
        racer_id: r.id, racer_name: r.name,
        position: r.rank, points: r.points,
        fastest_lap: false, finished: true
    }));

    const res = await fetch('/api/race-history', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            name: name,
            race_date: date,
            country: race.country,
            track: race.track,
            track_id: race.track_id,
            total_laps: race.laps,
            results: results
        })
    });

    if (res.ok) {
        alert('Race archived!');
        loadHistory();
        (document.getElementById('archive-name') as HTMLInputElement).value = '';
    }
}

function openRacerModal(): void {
    (document.getElementById('racer-form') as HTMLFormElement).reset();
    (document.getElementById('racer-id') as HTMLInputElement).value = '';
    document.getElementById('racer-pic-preview')!.style.display = 'none';
    document.getElementById('racerModalLabel')!.textContent = 'Add New Racer';
    racerModal.show();
}

function editRacer(id: number): void {
    const r = adminRacers.find(x => x.id === id);
    if (!r) return;
    const form = document.getElementById('racer-form') as HTMLFormElement;
    (form.elements.namedItem('id') as HTMLInputElement).value = String(r.id);
    (form.elements.namedItem('name') as HTMLInputElement).value = r.name;
    (form.elements.namedItem('profile_picture') as HTMLInputElement).value = r.profile_picture;
    (form.elements.namedItem('car_name') as HTMLInputElement).value = r.car_name;
    (form.elements.namedItem('car_color') as HTMLSelectElement).value = r.car_color;
    (form.elements.namedItem('points') as HTMLInputElement).value = String(r.points);
    (form.elements.namedItem('rank') as HTMLInputElement).value = String(r.rank);
    (form.elements.namedItem('position') as HTMLInputElement).value = String(r.position);
    const preview = document.getElementById('racer-pic-preview') as HTMLImageElement;
    preview.src = r.profile_picture;
    preview.style.display = 'inline-block';
    document.getElementById('racerModalLabel')!.textContent = 'Edit Racer: ' + r.name;
    racerModal.show();
}

async function deleteRacer(id: number): Promise<void> {
    if (confirm('Delete this racer?')) {
        await fetch(`/api/racers?id=${id}`, { method: 'DELETE' });
        loadRacers();
    }
}

function openQuoteModal(): void {
    (document.getElementById('quote-form') as HTMLFormElement).reset();
    (document.getElementById('quote-id') as HTMLInputElement).value = '';
    document.getElementById('quoteModalLabel')!.textContent = 'Add Quote';
    quoteModal.show();
}

async function editQuote(id: number): Promise<void> {
    const res = await fetch('/api/quotes');
    const quotes: Quote[] = await res.json();
    const q = quotes.find(x => x.id === id);
    if (!q) return;
    const form = document.getElementById('quote-form') as HTMLFormElement;
    (form.elements.namedItem('id') as HTMLInputElement).value = String(q.id);
    (form.elements.namedItem('text') as HTMLTextAreaElement).value = q.text;
    (form.elements.namedItem('author') as HTMLInputElement).value = q.author;
    document.getElementById('quoteModalLabel')!.textContent = 'Edit Quote';
    quoteModal.show();
}

async function deleteQuote(id: number): Promise<void> {
    if (confirm('Delete this quote?')) {
        await fetch(`/api/quotes?id=${id}`, { method: 'DELETE' });
        loadQuotes();
    }
}

async function deleteHistory(id: number): Promise<void> {
    if (confirm('Remove this entry from history?')) {
        await fetch(`/api/race-history?id=${id}`, { method: 'DELETE' });
        loadHistory();
    }
}

async function uploadImage(input: HTMLInputElement): Promise<void> {
    const file = input.files?.[0];
    if (!file) return;
    const fd = new FormData();
    fd.append('image', file);
    const res = await fetch('/api/upload', { method: 'POST', body: fd });
    const data = await res.json();
    if (!res.ok) {
        alert('Upload failed: ' + (data.error || 'Unknown error'));
        return;
    }
    (document.getElementById('profile_picture') as HTMLInputElement).value = data.url;
    const preview = document.getElementById('racer-pic-preview') as HTMLImageElement;
    preview.src = data.url;
    preview.style.display = 'inline-block';
}

function openTrackModal(): void {
    (document.getElementById('track-form') as HTMLFormElement).reset();
    (document.getElementById('track-id') as HTMLInputElement).value = '';
    document.getElementById('trackModalLabel')!.textContent = 'Add New Track';
    trackModal.show();
}

function editTrack(id: string): void {
    const t = allTracks.find(x => x.id === id);
    if (!t) return;
    const form = document.getElementById('track-form') as HTMLFormElement;
    (form.elements.namedItem('id') as HTMLInputElement).value = t.id;
    (document.getElementById('track-id-visible') as HTMLInputElement).value = t.id;
    (form.elements.namedItem('name') as HTMLInputElement).value = t.name;
    (form.elements.namedItem('country') as HTMLInputElement).value = t.country;
    (form.elements.namedItem('length_km') as HTMLInputElement).value = String(t.length_km);
    (form.elements.namedItem('lap_record') as HTMLInputElement).value = t.lap_record || '';
    (document.getElementById('refresh-geojson') as HTMLInputElement).checked = t.refresh_geojson;
    (document.getElementById('use-map-image') as HTMLInputElement).checked = t.use_map_image;
    (document.getElementById('map-image-url') as HTMLInputElement).value = t.map_image_url || '';
    document.getElementById('map-image-settings')!.style.display = t.use_map_image ? 'block' : 'none';
    document.getElementById('trackModalLabel')!.innerText = 'Edit Track';
    trackModal.show();
}

async function uploadMapImage(input: HTMLInputElement): Promise<void> {
    if (!input.files || !input.files[0]) return;
    const formData = new FormData();
    formData.append('image', input.files[0]);
    const res = await fetch('/api/upload', { method: 'POST', body: formData });
    const data = await res.json();
    if (!res.ok) {
        alert('Upload failed: ' + (data.error || 'Unknown error'));
        return;
    }
    (document.getElementById('map-image-url') as HTMLInputElement).value = data.url;
}

async function loadAISettings(): Promise<void> {
    try {
        const res = await fetch('/api/ai-settings');
        const data = await res.json();
        (document.getElementById('ai-track-extract-url') as HTMLInputElement).value = data.track_extract_url || '';
        (document.getElementById('ai-api-key') as HTMLInputElement).value = data.api_key || '';
        (document.getElementById('ai-enabled') as HTMLInputElement).checked = data.enabled;
    } catch (e) { console.error('Failed to load AI settings', e); }
}

document.getElementById('ai-settings-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const data = {
        track_extract_url: (document.getElementById('ai-track-extract-url') as HTMLInputElement).value,
        api_key: (document.getElementById('ai-api-key') as HTMLInputElement).value,
        enabled: (document.getElementById('ai-enabled') as HTMLInputElement).checked
    };
    const res = await fetch('/api/ai-settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) alert('AI settings saved!');
});

async function extractTrackFromAI(): Promise<void> {
    const btn = document.querySelector('[onclick="extractTrackFromAI()"]') as HTMLButtonElement;
    if (!btn) return;
    const originalText = btn.innerHTML;
    btn.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-1"></i> Analyzing...';
    btn.disabled = true;

    try {
        const mapUrl = (document.getElementById('map-image-url') as HTMLInputElement).value;
        if (!mapUrl) {
            alert('Please upload a map image first!');
            btn.innerHTML = originalText;
            btn.disabled = false;
            return;
        }

        const res = await fetch('/api/tracks/ai-extract', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ image_url: mapUrl })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'AI extraction failed');
        }
        const data = await res.json();
        alert('AI successfully analyzed the track! GeoJSON has been generated. Save the track to apply it.');
        console.log('AI Extracted GeoJSON:', data);
    } catch (e: any) {
        alert('AI extraction failed: ' + e.message);
    } finally {
        btn.innerHTML = originalText;
        btn.disabled = false;
    }
}

async function uploadAIImage(input: HTMLInputElement): Promise<void> {
    if (!input.files || !input.files[0]) return;
    const formData = new FormData();
    formData.append('image', input.files[0]);
    const res = await fetch('/api/upload', { method: 'POST', body: formData });
    const data = await res.json();
    if (!res.ok) {
        alert('Upload failed: ' + (data.error || 'Unknown error'));
        return;
    }
    (document.getElementById('ai-extract-map-url') as HTMLInputElement).value = data.url;
    const preview = document.getElementById('ai-extract-preview')!;
    const img = document.getElementById('ai-extract-preview-img') as HTMLImageElement;
    img.src = data.url;
    preview.style.display = 'block';
}

let extractedGeoJSON: any = null;

async function extractTrackFromAIStandalone(): Promise<void> {
    const btn = document.querySelector('[onclick="extractTrackFromAIStandalone()"]') as HTMLButtonElement;
    if (!btn) return;
    const originalText = btn.innerHTML;
    btn.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-1"></i> Analyzing...';
    btn.disabled = true;

    try {
        const mapUrl = (document.getElementById('ai-extract-map-url') as HTMLInputElement).value;
        if (!mapUrl) {
            alert('Please upload a map image first!');
            btn.innerHTML = originalText;
            btn.disabled = false;
            return;
        }

        const res = await fetch('/api/tracks/ai-extract', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ image_url: mapUrl })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'AI extraction failed');
        }
        const data = await res.json();
        extractedGeoJSON = data;
        const result = document.getElementById('ai-extract-result')!;
        const pre = document.getElementById('ai-extract-geojson')!;
        pre.textContent = JSON.stringify(data, null, 2);
        result.style.display = 'block';
    } catch (e: any) {
        alert('AI extraction failed: ' + e.message);
    } finally {
        btn.innerHTML = originalText;
        btn.disabled = false;
    }
}

async function saveExtractedTrack(): Promise<void> {
    if (!extractedGeoJSON) {
        alert('Please extract a track first!');
        return;
    }
    const trackId = (document.getElementById('ai-extract-track-id') as HTMLInputElement).value;
    if (!trackId) {
        alert('Please enter a track ID!');
        return;
    }
    const res = await fetch('/api/tracks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            id: trackId,
            name: trackId,
            country: 'Unknown',
            length_km: 5,
            lap_record: '--',
            use_map_image: true,
            map_image_url: (document.getElementById('ai-extract-map-url') as HTMLInputElement).value,
            refresh_geojson: true
        })
    });
    if (res.ok) {
        alert('Track saved!');
        loadAdminTracks();
        loadAllTracks();
    } else {
        const err = await res.json();
        alert('Failed to save track: ' + (err.error || 'Unknown error'));
    }
}

async function deleteTrack(id: string): Promise<void> {
    if (confirm('Delete this track?')) {
        await fetch(`/api/tracks?id=${id}`, { method: 'DELETE' });
        loadAdminTracks();
        loadAllTracks();
    }
}

async function logout(): Promise<void> {
    await fetch('/api/logout', { method: 'POST' });
    window.location.href = '/login.html';
}

function escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text || '';
    return div.innerHTML;
}

// Email settings
interface RacerEmail {
    id: number;
    racer_id: number;
    email: string;
}

async function loadEmailSettings(): Promise<void> {
    try {
        const res = await fetch('/api/email-settings');
        const data = await res.json();
        (document.getElementById('smtp-host') as HTMLInputElement).value = data.smtp_host || '';
        (document.getElementById('smtp-port') as HTMLInputElement).value = String(data.smtp_port || 587);
        (document.getElementById('smtp-username') as HTMLInputElement).value = data.username || '';
        (document.getElementById('smtp-password') as HTMLInputElement).value = data.password || '';
        (document.getElementById('smtp-from') as HTMLInputElement).value = data.from_addr || '';
        (document.getElementById('email-enabled') as HTMLInputElement).checked = data.enabled;
    } catch (e) { console.error('Failed to load email settings', e); }
}

document.getElementById('email-settings-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const data = {
        smtp_host: (document.getElementById('smtp-host') as HTMLInputElement).value,
        smtp_port: parseInt((document.getElementById('smtp-port') as HTMLInputElement).value) || 587,
        username: (document.getElementById('smtp-username') as HTMLInputElement).value,
        password: (document.getElementById('smtp-password') as HTMLInputElement).value,
        from_addr: (document.getElementById('smtp-from') as HTMLInputElement).value,
        enabled: (document.getElementById('email-enabled') as HTMLInputElement).checked
    };
    const res = await fetch('/api/email-settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) alert('Email settings saved!');
    else alert('Failed to save email settings');
});

async function loadRacerEmails(): Promise<void> {
    try {
        const [racersRes, emailsRes] = await Promise.all([
            fetch('/api/racers'),
            fetch('/api/racer-emails')
        ]);
        const racers: AdminRacer[] = await racersRes.json();
        const emails: RacerEmail[] = await emailsRes.json();
        const container = document.getElementById('racer-email-list')!;
        container.innerHTML = racers.map(r => {
            const match = emails.find(e => e.racer_id === r.id);
            return `
                <div class="d-flex align-items-center mb-2 p-2 border rounded">
                    <img src="${escapeHtml(r.profile_picture)}" class="rounded-circle me-2" width="28" height="28" style="object-fit:cover" onerror="this.src='/static/images/helmet.svg'">
                    <span class="fw-bold me-auto small">${escapeHtml(r.name)}</span>
                    <input type="email" class="form-control form-control-sm" style="width:200px" placeholder="email@example.com" value="${escapeHtml(match?.email || '')}" data-racer-id="${r.id}" onchange="saveRacerEmailField(this)">
                </div>
            `;
        }).join('');
    } catch (e) { console.error('Failed to load racer emails', e); }
}

async function saveRacerEmailField(input: HTMLInputElement): Promise<void> {
    const racerId = parseInt(input.dataset.racerId || '0');
    if (!racerId) return;
    const email = input.value.trim();
    await fetch('/api/racer-emails', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ racer_id: racerId, email: email })
    });
}

document.getElementById('email-tab')?.addEventListener('shown.bs.tab', () => {
    loadEmailSettings();
    loadRacerEmails();
});

init();

