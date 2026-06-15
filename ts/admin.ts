import { showToast, escapeHtml } from './toast';

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
    gold: number;
    silver: number;
    bronze: number;
    fastest_laps: number;
    dnf: number;
    dns: number;
}

let adminRacers: AdminRacer[] = [];
let adminTracks: AdminTrack[] = [];
let allTracks: AdminTrack[] = [];
let qualificationOrder: AdminRacer[] = [];
let shuffleInterval: ReturnType<typeof setInterval> | null = null;
let racerStats: AdminStats[] = [];
let driverShares: Record<number, string> = {};
declare const bootstrap: any;

const racerModal = new bootstrap.Modal(document.getElementById('racerModal')!);
const quoteModal = new bootstrap.Modal(document.getElementById('quoteModal')!);
const trackModal = new bootstrap.Modal(document.getElementById('trackModal')!);
const statsModal = new bootstrap.Modal(document.getElementById('statsModal')!);

async function init(): Promise<void> {
    await loadAdminTracks();
    await loadRaceInfo();
    await loadRacers();
    await loadAllTracks();
    await loadQuotes();
    await loadRacerStats();
    await loadNotificationSettings();
    await loadUmamiSettings();
    await loadAISettings();
    await loadOTelSettings();
    await loadEInkSettings();
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
        const [racersRes, sharesRes] = await Promise.all([
            fetch('/api/racers'),
            fetch('/api/driver-shares').catch(() => new Response('[]'))
        ]);
        adminRacers = await racersRes.json();
        driverShares = {};
        if (sharesRes.ok) {
            const shares: any[] = await sharesRes.json();
            for (const s of shares) {
                driverShares[s.racer_id] = s.token;
            }
        }
        const list = document.getElementById('racer-list')!;
        list.innerHTML = adminRacers.map(r => {
            const token = driverShares[r.id];
            const shareLink = token ? `${window.location.origin}/driver.html?token=${token}` : '';
            return `
            <tr>
                <td class="ps-4 fw-bold">#${r.rank}</td>
                <td>
                    <div class="d-flex align-items-center">
                        <img src="${r.profile_picture}" class="rounded-circle me-3" width="32" height="32" style="object-fit: cover" onerror="this.src='/static/images/helmet.svg'">
                        <div><div class="fw-bold">${escapeHtml(r.name)}</div></div>
                    </div>
                </td>
                <td><span class="color-dot" style="background:${r.car_color.startsWith('#')?r.car_color:'#'+r.car_color}"></span> ${escapeHtml(r.car_name)}</td>
                <td><span class="badge bg-dark">${r.points} pts</span></td>
                <td class="small text-muted">${r.position}</td>
                <td>
                    ${shareLink
                        ? `<button class="btn btn-sm btn-outline-success" onclick="copyShareLink(${r.id})" title="Copy share link"><i class="fa-solid fa-link"></i></button>`
                        : `<button class="btn btn-sm btn-outline-secondary" onclick="generateShareLink(${r.id})" title="Generate share link"><i class="fa-solid fa-share-nodes"></i></button>`
                    }
                </td>
                <td class="text-end pe-4">
                    <button class="btn btn-sm btn-outline-primary" onclick="editRacer(${r.id})"><i class="fa-solid fa-pen"></i></button>
                    <button class="btn btn-sm btn-outline-danger" onclick="deleteRacer(${r.id})"><i class="fa-solid fa-trash"></i></button>
                </td>
            </tr>
        `}).join('');
    } catch (e) { console.error('Failed to load racers', e); }
}

async function generateShareLink(racerId: number): Promise<void> {
    try {
        const res = await fetch('/api/driver-share?racer_id=' + racerId, { method: 'POST' });
        if (res.ok) {
            loadRacers();
        } else {
            const err = await res.json();
            showToast('Failed to generate link: ' + (err.error || 'Unknown error'), 'error');
        }
    } catch (e: any) {
        showToast('Error: ' + e.message, 'error');
    }
}

async function copyShareLink(racerId: number): Promise<void> {
    const token = driverShares[racerId];
    if (!token) return;
    const link = `${window.location.origin}/driver.html?token=${token}`;
    try {
        await navigator.clipboard.writeText(link);
        showToast('Share link copied to clipboard!', 'success');
    } catch {
        const input = document.createElement('input');
        input.value = link;
        document.body.appendChild(input);
        input.select();
        document.execCommand('copy');
        document.body.removeChild(input);
        showToast('Share link copied!', 'success');
    }
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
    if (res.ok) showToast('Analytics settings saved!', 'success');
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
    if (res.ok) showToast('Notification settings saved!', 'success');
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
            showToast('Test notification sent successfully!', 'success');
        } else {
            const err = await res.text();
            showToast('Failed to send: ' + err, 'error');
        }
    } catch (e: any) {
        showToast('Error: ' + e.message, 'error');
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
        list.innerHTML = '<tr><td colspan="8" class="text-center text-muted py-4">No stats yet. <button class="btn btn-sm btn-outline-primary ms-2" onclick="openStatsModal()"><i class="fa-solid fa-plus me-1"></i>Create Stats</button></td></tr>';
        return;
    }
    list.innerHTML = racerStats.map(s => `
        <tr>
            <td class="ps-4 fw-bold">${racerNameById(s.racer_id)}</td>
            <td>${s.races}</td>
            <td><span class="badge bg-warning text-dark">${s.gold || 0}</span> <span class="badge bg-secondary">${s.silver || 0}</span> <span class="badge" style="background:#cd7f32">${s.bronze || 0}</span></td>
            <td><span class="badge bg-info">${s.fastest_laps}</span></td>
            <td><span class="badge bg-danger">${s.dnf}</span></td>
            <td><span class="badge bg-dark">${s.dns}</span></td>
            <td class="text-end pe-4">
                <button class="btn btn-sm btn-outline-primary" onclick="editStats(${s.id})"><i class="fa-solid fa-pen"></i></button>
            </td>
        </tr>
    `).join('');
}

function openStatsModal(stat?: AdminStats): void {
    const select = document.getElementById('stats-racer-select') as HTMLSelectElement;
    const currentRacerId = stat ? stat.racer_id : 0;

    select.innerHTML = '<option value="">Select a driver...</option>' +
        adminRacers.map(r =>
            `<option value="${r.id}" ${r.id === currentRacerId ? 'selected' : ''}>${escapeHtml(r.name)}</option>`
        ).join('');

    if (stat) {
        (document.getElementById('stats-id') as HTMLInputElement).value = String(stat.id);
        (document.getElementById('stats-racer-id') as HTMLInputElement).value = String(stat.racer_id);
        (document.getElementById('stats-races') as HTMLInputElement).value = String(stat.races);
        (document.getElementById('stats-gold') as HTMLInputElement).value = String(stat.gold || 0);
        (document.getElementById('stats-silver') as HTMLInputElement).value = String(stat.silver || 0);
        (document.getElementById('stats-bronze') as HTMLInputElement).value = String(stat.bronze || 0);
        (document.getElementById('stats-fastest-laps') as HTMLInputElement).value = String(stat.fastest_laps);
        (document.getElementById('stats-dnf') as HTMLInputElement).value = String(stat.dnf);
        (document.getElementById('stats-dns') as HTMLInputElement).value = String(stat.dns);
        (document.getElementById('statsModalLabel') as HTMLElement).textContent = 'Edit Stats: ' + racerNameById(stat.racer_id);
    } else {
        (document.getElementById('stats-form') as HTMLFormElement).reset();
        (document.getElementById('stats-id') as HTMLInputElement).value = '';
        (document.getElementById('stats-racer-id') as HTMLInputElement).value = '';
        (document.getElementById('statsModalLabel') as HTMLElement).textContent = 'Add New Stats';
    }

    statsModal.show();
}

function editStats(id: number): void {
    const stat = racerStats.find(s => s.id === id);
    if (!stat) return;
    openStatsModal(stat);
}

document.getElementById('stats-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const select = document.getElementById('stats-racer-select') as HTMLSelectElement;
    const racerId = parseInt(select.value);
    if (!racerId) {
        showToast('Please select a driver', 'error');
        return;
    }

    const el = (id: string) => document.getElementById(id) as HTMLInputElement;
    const gold = parseInt(el('stats-gold')?.value) || 0;
    const data = {
        id: parseInt(el('stats-id')?.value) || 0,
        racer_id: racerId,
        races: parseInt(el('stats-races')?.value) || 0,
        wins: gold,
        gold: gold,
        silver: parseInt(el('stats-silver')?.value) || 0,
        bronze: parseInt(el('stats-bronze')?.value) || 0,
        fastest_laps: parseInt(el('stats-fastest-laps')?.value) || 0,
        dnf: parseInt(el('stats-dnf')?.value) || 0,
        dns: parseInt(el('stats-dns')?.value) || 0
    };

    const res = await fetch('/api/racer-stats', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) {
        statsModal.hide();
        loadRacerStats();
    } else {
        const err = await res.json();
        showToast('Failed to save stats: ' + (err.error || 'Unknown error'), 'error');
    }
});

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
    if (res.ok) showToast('Race data updated!', 'success');
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
        showToast('Failed to save racer: ' + (err.error || 'Unknown error'), 'error');
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
        showToast('No racers available for qualification!', 'info');
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
                            <div class="small opacity-75"><span class="color-dot" style="background:${r.car_color.startsWith('#')?r.car_color:'#'+r.car_color}"></span>${escapeHtml(r.car_name)}</div>
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
                <div class="small text-muted"><span class="color-dot" style="background:${r.car_color.startsWith('#')?r.car_color:'#'+r.car_color}"></span>${escapeHtml(r.car_name)}</div>
            </div>
            ${i === 0 ? '<i class="fa-solid fa-crown text-warning"></i>' : ''}
        </div>
    `).join('');
}

async function applyGridPositions(): Promise<void> {
    if (qualificationOrder.length === 0) {
        showToast('Please run qualification first!', 'error');
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

    showToast('Grid positions applied!', 'success');
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
        showToast('Please run qualification first!', 'error');
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
                <div class="small text-muted"><span class="color-dot" style="background:${r.car_color.startsWith('#')?r.car_color:'#'+r.car_color}"></span>${escapeHtml(r.car_name)}</div>
                ${i === 0 ? '<i class="fa-solid fa-crown text-warning fa-2x mt-2"></i>' : ''}
            </div>
        </div>
    `).join('');
}

function openRacerModal(): void {
    (document.getElementById('racer-form') as HTMLFormElement).reset();
    (document.getElementById('racer-id') as HTMLInputElement).value = '';
    document.getElementById('racer-pic-preview')!.style.display = 'none';
    document.getElementById('racerModalLabel')!.textContent = 'Add New Racer';
    racerModal.show();
}

function updateCarPreview(color: string): void {
    const bar = document.getElementById('car-preview-bar');
    if (bar) bar.style.background = color;
}

function initColorSwatches(): void {
    const swatches = document.getElementById('color-swatches');
    if (!swatches) return;
    const colors = ['#d40000','#ff4d4d','#ff8700','#ffd700','#4dff88','#4d94ff','#005aff','#9b59b6','#e67e22','#333333','#a6a6a6','#ffffff'];
    swatches.innerHTML = colors.map(c =>
        `<button type="button" class="btn p-0 m-0" style="width:24px;height:24px;background:${c};border:2px solid #ddd;border-radius:4px;cursor:pointer" onclick="document.getElementById('car_color').value='${c}';document.getElementById('car_color_text').value='${c}';updateCarPreview('${c}')" title="${c}"></button>`
    ).join('');
}

document.addEventListener('DOMContentLoaded', initColorSwatches);

function editRacer(id: number): void {
    const r = adminRacers.find(x => x.id === id);
    if (!r) return;
    const form = document.getElementById('racer-form') as HTMLFormElement;
    (form.elements.namedItem('id') as HTMLInputElement).value = String(r.id);
    (form.elements.namedItem('name') as HTMLInputElement).value = r.name;
    (form.elements.namedItem('profile_picture') as HTMLInputElement).value = r.profile_picture;
    (form.elements.namedItem('car_name') as HTMLInputElement).value = r.car_name;
    const color = r.car_color.startsWith('#') ? r.car_color : '#' + r.car_color;
    (document.getElementById('car_color') as HTMLInputElement).value = color;
    (document.getElementById('car_color_text') as HTMLInputElement).value = color;
    updateCarPreview(color);
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

async function uploadImage(input: HTMLInputElement): Promise<void> {
    const file = input.files?.[0];
    if (!file) return;
    const fd = new FormData();
    fd.append('image', file);
    try {
        const res = await fetch('/api/upload', { method: 'POST', body: fd });
        const data = await res.json();
        if (!res.ok) {
            showToast('Upload failed: ' + (data.error || 'Unknown error'), 'error');
            return;
        }
        (document.getElementById('profile_picture') as HTMLInputElement).value = data.url;
        const preview = document.getElementById('racer-pic-preview') as HTMLImageElement;
        preview.src = data.url;
        preview.style.display = 'inline-block';
    } catch (e: any) {
        showToast('Upload failed: ' + (e.message || 'Unknown error'), 'error');
    }
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
    try {
        const res = await fetch('/api/upload', { method: 'POST', body: formData });
        const data = await res.json();
        if (!res.ok) {
            showToast('Upload failed: ' + (data.error || 'Unknown error'), 'error');
            return;
        }
        (document.getElementById('map-image-url') as HTMLInputElement).value = data.url;
    } catch (e: any) {
        showToast('Upload failed: ' + (e.message || 'Unknown error'), 'error');
    }
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

async function loadOTelSettings(): Promise<void> {
    try {
        const res = await fetch('/api/otel-settings');
        const data = await res.json();
        (document.getElementById('otel-endpoint') as HTMLInputElement).value = data.endpoint || '';
        (document.getElementById('otel-traces-enabled') as HTMLInputElement).checked = data.traces_enabled;
        (document.getElementById('otel-metrics-enabled') as HTMLInputElement).checked = data.metrics_enabled;
        (document.getElementById('otel-logs-enabled') as HTMLInputElement).checked = data.logs_enabled;
    } catch (e) { console.error('Failed to load OTel settings', e); }
}

document.getElementById('otel-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const data = {
        endpoint: (document.getElementById('otel-endpoint') as HTMLInputElement).value,
        traces_enabled: (document.getElementById('otel-traces-enabled') as HTMLInputElement).checked,
        metrics_enabled: (document.getElementById('otel-metrics-enabled') as HTMLInputElement).checked,
        logs_enabled: (document.getElementById('otel-logs-enabled') as HTMLInputElement).checked
    };
    const res = await fetch('/api/otel-settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) showToast('Telemetry settings saved!', 'success');
    else {
        const err = await res.json();
        showToast('Failed to save telemetry settings: ' + (err.error || 'Unknown error'), 'error');
    }
});

async function loadEInkSettings(): Promise<void> {
    try {
        const res = await fetch('/api/eink-settings');
        const data = await res.json();
        (document.getElementById('eink-enabled') as HTMLInputElement).checked = data.enabled;
    } catch (e) { console.error('Failed to load e-ink settings', e); }
}

document.getElementById('eink-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const data = {
        enabled: (document.getElementById('eink-enabled') as HTMLInputElement).checked
    };
    const res = await fetch('/api/eink-settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) showToast('E-Ink settings saved!', 'success');
    else {
        const err = await res.json();
        showToast('Failed to save e-ink settings: ' + (err.error || 'Unknown error'), 'error');
    }
});

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
    if (res.ok) showToast('AI settings saved!', 'success');
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
            showToast('Please upload a map image first!', 'error');
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
        showToast('AI successfully analyzed the track! GeoJSON has been generated. Save the track to apply it.', 'success');
        console.log('AI Extracted GeoJSON:', data);
    } catch (e: any) {
        showToast('AI extraction failed: ' + e.message, 'error');
    } finally {
        btn.innerHTML = originalText;
        btn.disabled = false;
    }
}

async function uploadAIImage(input: HTMLInputElement): Promise<void> {
    if (!input.files || !input.files[0]) return;
    const formData = new FormData();
    formData.append('image', input.files[0]);
    try {
        const res = await fetch('/api/upload', { method: 'POST', body: formData });
        const data = await res.json();
        if (!res.ok) {
            showToast('Upload failed: ' + (data.error || 'Unknown error'), 'error');
            return;
        }
        (document.getElementById('ai-extract-map-url') as HTMLInputElement).value = data.url;
        const preview = document.getElementById('ai-extract-preview')!;
        const img = document.getElementById('ai-extract-preview-img') as HTMLImageElement;
        img.src = data.url;
        preview.style.display = 'block';
    } catch (e: any) {
        showToast('Upload failed: ' + (e.message || 'Unknown error'), 'error');
    }
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
            showToast('Please upload a map image first!', 'error');
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
        showToast('AI extraction failed: ' + e.message, 'error');
    } finally {
        btn.innerHTML = originalText;
        btn.disabled = false;
    }
}

async function saveExtractedTrack(): Promise<void> {
    if (!extractedGeoJSON) {
        showToast('Please extract a track first!', 'error');
        return;
    }
    const trackId = (document.getElementById('ai-extract-track-id') as HTMLInputElement).value;
    if (!trackId) {
        showToast('Please enter a track ID!', 'error');
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
        showToast('Track saved!', 'success');
        loadAdminTracks();
        loadAllTracks();
    } else {
        const err = await res.json();
        showToast('Failed to save track: ' + (err.error || 'Unknown error'), 'error');
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
    if (res.ok) showToast('Email settings saved!', 'success');
    else showToast('Failed to save email settings', 'error');
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

// Backup settings
interface BackupInfo {
    name: string;
    size: number;
    time: string;
}

async function loadBackupSettings(): Promise<void> {
    try {
        const res = await fetch('/api/backup-settings');
        const data = await res.json();
        (document.getElementById('backup-enabled') as HTMLInputElement).checked = data.enabled;
        (document.getElementById('backup-interval') as HTMLSelectElement).value = String(data.interval_hrs || 24);
        (document.getElementById('backup-retention') as HTMLInputElement).value = String(data.retention_count || 7);
    } catch (e) { console.error('Failed to load backup settings', e); }
}

async function loadBackupList(): Promise<void> {
    try {
        const res = await fetch('/api/backup/list');
        const backups: BackupInfo[] = await res.json();
        const list = document.getElementById('backup-list')!;
        if (backups.length === 0) {
            list.innerHTML = '<tr><td colspan="3" class="text-center text-muted py-4">No backups found.</td></tr>';
            return;
        }
        list.innerHTML = backups.slice().reverse().map(b => `
            <tr>
                <td class="ps-4"><code>${escapeHtml(b.name)}</code></td>
                <td>${(b.size / 1024).toFixed(1)} KB</td>
                <td class="text-end pe-4 small text-muted">${escapeHtml(b.time)}</td>
            </tr>
        `).join('');
    } catch (e) { console.error('Failed to load backup list', e); }
}

async function triggerManualBackup(): Promise<void> {
    const btn = document.getElementById('backup-manual-btn') as HTMLButtonElement;
    btn.disabled = true;
    btn.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-1"></i> Backing up...';
    try {
        const res = await fetch('/api/backup/manual', { method: 'POST' });
        const result = document.getElementById('backup-result')!;
        result.classList.remove('d-none', 'alert-success', 'alert-danger');
        if (res.ok) {
            result.classList.add('alert-success');
            result.textContent = 'Backup created successfully!';
            loadBackupList();
        } else {
            const err = await res.json();
            result.classList.add('alert-danger');
            result.textContent = 'Backup failed: ' + (err.error || 'Unknown error');
        }
    } catch (e: any) {
        const result = document.getElementById('backup-result')!;
        result.classList.remove('d-none', 'alert-success', 'alert-danger');
        result.classList.add('alert-danger');
        result.textContent = 'Error: ' + e.message;
    } finally {
        btn.disabled = false;
        btn.innerHTML = '<i class="fa-solid fa-play me-1"></i> Backup Now';
    }
}

document.getElementById('backup-form')!.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const data = {
        enabled: (document.getElementById('backup-enabled') as HTMLInputElement).checked,
        interval_hrs: parseInt((document.getElementById('backup-interval') as HTMLSelectElement).value) || 24,
        retention_count: parseInt((document.getElementById('backup-retention') as HTMLInputElement).value) || 7
    };
    const res = await fetch('/api/backup-settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (res.ok) showToast('Backup settings saved!', 'success');
    else showToast('Failed to save backup settings', 'error');
});

document.getElementById('backup-tab')?.addEventListener('shown.bs.tab', () => {
    loadBackupSettings();
    loadBackupList();
});

document.getElementById('rounds-tab')?.addEventListener('shown.bs.tab', () => {
    loadRoundsList();
});

document.getElementById('seasons-tab')?.addEventListener('shown.bs.tab', () => {
    loadSeasons();
});

async function loadRoundsList(): Promise<void> {
    try {
        const res = await fetch('/api/seasons');
        const seasons = await res.json();
        const active = Array.isArray(seasons) ? seasons.find((s: any) => s.status === 'active') : null;
        const sid = active ? active.id : 1;
        const roundsRes = await fetch(`/api/rounds?season_id=${sid}`);
        const rounds = await roundsRes.json();
        const list = document.getElementById('rounds-list')!;
        if (!Array.isArray(rounds) || rounds.length === 0) {
            list.innerHTML = '<tr><td colspan="4" class="text-center text-muted py-4">No rounds yet.</td></tr>';
            return;
        }
        list.innerHTML = rounds.map((r: any) => `
            <tr>
                <td class="ps-4 fw-bold">#${r.round || r.id}</td>
                <td>${r.race_name}</td>
                <td>${r.race_date}</td>
                <td class="text-end pe-4">
                    <button class="btn btn-sm btn-outline-danger" onclick="deleteRound(${r.id})"><i class="fa-solid fa-trash"></i></button>
                </td>
            </tr>
        `).join('');
    } catch (e) {
        console.error('Failed to load rounds', e);
    }
}

async function takeAdminRoundSnapshot(): Promise<void> {
    const name = prompt('Round name:', `Round ${new Date().toLocaleDateString()}`);
    if (!name) return;
    const seasonsRes = await fetch('/api/seasons');
    const seasons = await seasonsRes.json();
    const active = Array.isArray(seasons) ? seasons.find((s: any) => s.status === 'active') : null;
    const res = await fetch('/api/rounds', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ race_name: name, round: 0, season_id: active ? active.id : 1 })
    });
    if (res.ok) {
        showToast('Round snapshot saved!', 'success');
        loadRoundsList();
    } else {
        const err = await res.json();
        showToast('Failed: ' + (err.error || 'Unknown error'), 'error');
    }
}

async function deleteRound(id: number): Promise<void> {
    if (!confirm('Delete this round snapshot?')) return;
    await fetch(`/api/rounds?id=${id}`, { method: 'DELETE' });
    loadRoundsList();
}

async function loadSeasons(): Promise<void> {
    try {
        const res = await fetch('/api/seasons');
        const seasons = await res.json();
        const list = document.getElementById('seasons-list')!;
        if (!Array.isArray(seasons) || seasons.length === 0) {
            list.innerHTML = '<tr><td colspan="5" class="text-center text-muted py-4">No seasons yet.</td></tr>';
            return;
        }
        list.innerHTML = seasons.map((s: any) => `
            <tr>
                <td class="ps-4 fw-bold">${s.name}</td>
                <td>${s.start_date}</td>
                <td>${s.end_date || '-'}</td>
                <td><span class="badge ${s.status === 'active' ? 'bg-success' : 'bg-secondary'}">${s.status}</span></td>
                <td class="text-end pe-4">
                    ${s.status === 'active'
                        ? `<button class="btn btn-sm btn-outline-warning" onclick="archiveSeason(${s.id})"><i class="fa-solid fa-box-archive"></i></button>`
                        : ''}
                    <button class="btn btn-sm btn-outline-danger" onclick="deleteSeason(${s.id})"><i class="fa-solid fa-trash"></i></button>
                </td>
            </tr>
        `).join('');

        const select = document.getElementById('season-rounds-select') as HTMLSelectElement;
        select.innerHTML = '<option value="">Select a season...</option>' +
            seasons.map((s: any) => `<option value="${s.id}">${s.name} (${s.status})</option>`).join('');
    } catch (e) {
        console.error('Failed to load seasons', e);
    }
}

async function createSeason(): Promise<void> {
    const name = prompt('Season name:', `Season ${new Date().getFullYear()}`);
    if (!name) return;
    const res = await fetch('/api/seasons', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
    });
    if (res.ok) {
        showToast('Season created!', 'info');
        loadSeasons();
    } else {
        showToast('Failed to create season', 'error');
    }
}

async function archiveSeason(id: number): Promise<void> {
    if (!confirm('Archive this season? This will end it.')) return;
    const res = await fetch(`/api/seasons/archive?id=${id}`, { method: 'POST' });
    if (res.ok) {
        showToast('Season archived!', 'info');
        loadSeasons();
    } else {
        showToast('Failed to archive season', 'error');
    }
}

async function deleteSeason(id: number): Promise<void> {
    if (!confirm('Delete this season? All associated rounds will also be deleted.')) return;
    await fetch(`/api/rounds?season_id=${id}`, { method: 'DELETE' });
    await fetch(`/api/seasons?id=${id}`, { method: 'DELETE' });
    loadSeasons();
}

async function loadSeasonRounds(seasonId: string): Promise<void> {
    if (!seasonId) return;
    const res = await fetch(`/api/rounds?season_id=${seasonId}`);
    const rounds = await res.json();
    const list = document.getElementById('season-rounds-list')!;
    if (!Array.isArray(rounds) || rounds.length === 0) {
        list.innerHTML = '<tr><td colspan="3" class="text-center text-muted py-4">No rounds in this season.</td></tr>';
        return;
    }
    list.innerHTML = rounds.map((r: any) => `
        <tr>
            <td class="ps-4 fw-bold">#${r.round}</td>
            <td>${r.race_name}</td>
            <td>${r.race_date}</td>
        </tr>
    `).join('');
}

init();

