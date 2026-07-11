import './i18n';
import './theme';
import { normalizeHex } from './color';
import { escapeHtml } from './toast';

interface Racer {
    id: number;
    name: string;
    profile_picture: string;
    car_color: string;
    car_name: string;
    points: number;
    rank: number;
    position: number;
    team_name?: string;
}

interface TrackInfo {
    id: string;
    name: string;
    country: string;
    geojson: string;
    length_km: number;
    lap_record: string;
    use_map_image: boolean;
    map_image_url: string;
    refresh_geojson: boolean;
}

interface RaceInfo {
    country: string;
    track: string;
    laps: number;
    track_id: string;
}

interface RacerStats {
    races: number;
    wins: number;
    gold: number;
    silver: number;
    bronze: number;
    fastest_laps: number;
    points: number;
    dnf: number;
    dns: number;
    spins: number;
    overheated: number;
}

interface StatsResponse {
    stats: RacerStats;
    racer: any;
}

interface GeoJSONFeature {
    type: string;
    properties: Record<string, string>;
    geometry: {
        type: string;
        coordinates: number[][];
    };
}

let trackGeoJSON: Record<string, any> = {};

declare const L: any;
let map: any;
let racerMarkers: Record<number, any> = {};
let currentTrack = 'monza';
let racers: Racer[] = [];
let sortColumn = 'rank';
let sortAsc = true;
let tracks: TrackInfo[] = [];

function getPointAtDistance(lineString: GeoJSONFeature, percentage: number): [number, number] {
    const tempLayer = L.geoJSON(lineString);
    const polyline = tempLayer.getLayers()[0];
    const latLngs = polyline.getLatLngs();
    const totalLength = latLngs.reduce((acc: number, curr: any, i: number, arr: any[]) => {
        if (i === 0) return 0;
        return acc + arr[i-1].distanceTo(curr);
    }, 0);
    const targetLength = totalLength * (percentage % 1);
    let currentLength = 0;
    for (let i = 0; i < latLngs.length - 1; i++) {
        const p1 = latLngs[i];
        const p2 = latLngs[i+1];
        const dist = p1.distanceTo(p2);
        if (currentLength + dist >= targetLength) {
            const ratio = (targetLength - currentLength) / dist;
            return [p1.lat + (p2.lat - p1.lat) * ratio, p1.lng + (p2.lng - p1.lng) * ratio];
        }
        currentLength += dist;
    }
    return [lineString.geometry.coordinates[0][1], lineString.geometry.coordinates[0][0]];
}

function initMap(): void {
    try {
        if (typeof L === 'undefined') {
            console.warn('Leaflet is not loaded');
            return;
        }
        if (map) {
            map.remove();
            racerMarkers = {};
        }
        const mapEl = document.getElementById('circuit-map');
        if (!mapEl) return;
        map = L.map('circuit-map', { zoomControl: false, attributionControl: false }).setView([45.621, 9.288], 14);
        L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', { maxZoom: 19 }).addTo(map);
        updateMap(currentTrack);
    } catch (e) {
        console.error('Error initializing map:', e);
    }
}

function updateMap(trackId: string): void {
    if (!map) {
        try { initMap(); } catch (e) { return; }
    }
    if (!map) return;
    try {
        Object.values(racerMarkers).forEach((m: any) => map.removeLayer(m));
        racerMarkers = {};
        map.eachLayer((layer: any) => {
            if (layer instanceof L.GeoJSON || layer instanceof L.ImageOverlay) {
                map.removeLayer(layer);
            }
        });
        const t = tracks.find(x => x.id === trackId) || { id: 'monza', use_map_image: false } as TrackInfo;
        const geojson = trackGeoJSON[trackId];
        if (!geojson) {
            console.warn('No GeoJSON data found for track:', trackId);
            return;
        }
        if (t.use_map_image && t.map_image_url) {
            const tempLayer = L.geoJSON(geojson);
            const bounds = tempLayer.getBounds();
            if (bounds.isValid()) {
                L.imageOverlay(t.map_image_url, bounds, { opacity: 0.8 }).addTo(map);
                map.fitBounds(bounds, { padding: [20, 20] });
            }
        } else {
            const layer = L.geoJSON(geojson, {
                style: { color: '#d40000', weight: 5, opacity: 0.8 }
            }).addTo(map);
            const bounds = layer.getBounds();
            if (bounds.isValid()) {
                map.fitBounds(bounds, { padding: [20, 20] });
            }
        }
    } catch (e) {
        console.error('Error updating map:', e);
    }
}

function updateMapMarkers(racers: Racer[]): void {
    if (!map) return;
    const t = tracks.find(x => x.id === currentTrack) || { id: 'monza', refresh_geojson: true } as TrackInfo;
    if (!t.refresh_geojson) {
        Object.values(racerMarkers).forEach((m: any) => map.removeLayer(m));
        racerMarkers = {};
        return;
    }
    const totalVakjes = 100;
    const geojson = trackGeoJSON[currentTrack];
    if (!geojson) return;
    racers.forEach(r => {
        const percentage = r.position / totalVakjes;
        const pos = getPointAtDistance(geojson, percentage);
        if (racerMarkers[r.id]) {
            racerMarkers[r.id].setLatLng(pos);
        } else {
            racerMarkers[r.id] = L.circleMarker(pos, {
                radius: 8, fillColor: normalizeHex(r.car_color),
                color: '#000', weight: 2, opacity: 1, fillOpacity: 0.8
            }).addTo(map).bindTooltip(r.name, { permanent: false, direction: 'top' });
        }
    });
    Object.keys(racerMarkers).forEach(id => {
        if (!racers.find(r => r.id == Number(id))) {
            map.removeLayer(racerMarkers[Number(id)]);
            delete racerMarkers[Number(id)];
        }
    });
}

function sortRacers(a: Racer, b: Racer): number {
    let aVal: string | number, bVal: string | number;
    switch(sortColumn) {
        case 'rank': aVal = a.rank; bVal = b.rank; break;
        case 'name': aVal = a.name.toLowerCase(); bVal = b.name.toLowerCase(); break;
        case 'team': aVal = (a.team_name || '').toLowerCase(); bVal = (b.team_name || '').toLowerCase(); break;
        case 'car_name': aVal = a.car_name.toLowerCase(); bVal = b.car_name.toLowerCase(); break;
        case 'gap': aVal = a.position; bVal = b.position; break;
        case 'points': aVal = a.points; bVal = b.points; break;
        default: aVal = a.rank; bVal = b.rank;
    }
    if (typeof aVal === 'number') {
        return sortAsc ? aVal - (bVal as number) : (bVal as number) - aVal;
    }
    return sortAsc ? (aVal as string).localeCompare(bVal as string) : (bVal as string).localeCompare(aVal as string);
}

document.querySelectorAll('#leaderboard-table th.sortable').forEach(th => {
    th.addEventListener('click', () => {
        const col = (th as HTMLElement).dataset.sort || 'rank';
        if (sortColumn === col) {
            sortAsc = !sortAsc;
        } else {
            sortColumn = col;
            sortAsc = true;
        }
        document.querySelectorAll('#leaderboard-table th.sortable i').forEach(i => (i as HTMLElement).className = 'fa-solid fa-sort small');
        (th.querySelector('i') as HTMLElement).className = `fa-solid fa-sort-${sortAsc ? 'up' : 'down'} small`;
        renderRacers();
    });
});

function renderRacers(): void {
    updateMapMarkers(racers);
    const lbBody = document.getElementById('leaderboard-body')!;
    const sorted = [...racers].sort(sortRacers);
    const leaderPosition = sorted.length > 0 ? sorted[0].position : 0;
    lbBody.innerHTML = sorted.map((r, i) => {
        let gapText = "-";
        if (i > 0) {
            const gap = leaderPosition - r.position;
            gapText = gap > 0 ? `+${gap}` : "0";
        }
        return `
            <tr class="${i === 0 ? 'winner-row' : ''} clickable-row" style="cursor:pointer" onclick="showDriverStats(${r.id})">
                <td class="ps-4">${r.rank}</td>
                <td>
                    <div class="driver-info">
                        <img src="${r.profile_picture}" class="avatar" alt="${r.name}" onerror="this.src='/static/images/helmet.svg'">
                        <span class="driver-name"><span class="color-indicator me-2" style="background:${normalizeHex(r.car_color)}"></span>${r.name}</span>
                    </div>
                </td>
                <td>${r.team_name || '-'}</td>
                <td>${r.car_name}</td>
                <td class="gap-cell">${gapText}</td>
                <td class="pe-4 text-end ${i === 0 ? 'fw-bold' : ''}">${r.points}</td>
            </tr>
        `;
    }).join('');
    const standings = [...racers].sort((a, b) => b.points - a.points);
    document.getElementById('standings-container')!.innerHTML = standings.map((r, i) => `
        <div class="standing-item" style="cursor:pointer" onclick="showDriverStats(${r.id})">
            <span class="rank">${i + 1}</span>
            <img src="${r.profile_picture}" class="avatar-sm" alt="${r.name}" onerror="this.src='/static/images/helmet.svg'">
            <span class="driver text-uppercase fw-bold">${r.name}</span>
            <span class="points ms-auto">${r.points} PTS</span>
        </div>
    `).join('');
}

async function showDriverStats(id: number): Promise<void> {
    const res = await fetch(`/api/racer-stats?id=${id}`);
    const data: StatsResponse = await res.json();
    const r = data.racer;
    const s = data.stats;
    document.getElementById('statsModalTitle')!.textContent = r.name || 'Unknown Driver';
    document.getElementById('statsModalBody')!.innerHTML = `
        <div class="text-center mb-4">
            <img src="${r.profile_picture || '/static/images/helmet.svg'}" class="rounded-circle mb-3" width="100" height="100" onerror="this.src='/static/images/helmet.svg'">
            <h4 class="mb-1">${r.car_name || 'Unknown Car'}</h4>
            <span class="color-indicator" style="background:${normalizeHex(r.car_color)}"></span> ${escapeHtml(r.car_color)}
        </div>
        <div class="row g-3">
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.races || 0}</span><span class="stat-label">Races</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.gold || s.wins || 0}</span><span class="stat-label">Wins</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.gold || 0}</span><span class="stat-label">Gold</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.silver || 0}</span><span class="stat-label">Silver</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.bronze || 0}</span><span class="stat-label">Bronze</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.fastest_laps || 0}</span><span class="stat-label">Fastest Laps</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.points || r.points || 0}</span><span class="stat-label">Total Points</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.dnf || 0}</span><span class="stat-label">DNF</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.spins || 0}</span><span class="stat-label">Spins</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.overheated || 0}</span><span class="stat-label">Overheated</span></div></div>
        </div>
    `;
    new (window as any).bootstrap.Modal(document.getElementById('statsModal')!).show();
}

async function loadQualificationGrid(): Promise<void> {
    try {
        const res = await fetch('/api/racers');
        const racers: Racer[] = await res.json();
        const sorted = [...racers].sort((a, b) => a.rank - b.rank);
        const gridDiv = document.getElementById('qualification-grid-display')!;
        if (sorted.length === 0) {
            gridDiv.innerHTML = '<div class="col-12 text-center text-muted py-4"><i class="fa-solid fa-shuffle fa-2x mb-2"></i><p>Qualification grid will appear here after admin shuffle</p></div>';
            return;
        }
        gridDiv.innerHTML = sorted.map((r, i) => `
            <div class="col-md-${sorted.length <= 4 ? Math.floor(12/sorted.length) : 3}">
                <div class="text-center p-3 border rounded ${i === 0 ? 'bg-warning bg-opacity-10 border-warning' : ''}">
                    <div class="mb-2"><span class="badge ${i === 0 ? 'bg-warning text-dark' : 'bg-secondary'} fs-6">P${i + 1}</span></div>
                    <img src="${r.profile_picture}" class="rounded-circle mb-2" width="48" height="48" style="object-fit: cover" onerror="this.src='/static/images/helmet.svg'">
                    <div class="fw-bold small">${r.name}</div>
                    <div class="small text-muted"><span class="color-indicator" style="background:${normalizeHex(r.car_color)}"></span>${r.car_name}</div>
                    ${i === 0 ? '<i class="fa-solid fa-crown text-warning"></i>' : ''}
                </div>
            </div>
        `).join('');
    } catch (err) {
        console.error('Failed to load qualification grid:', err);
    }
}

async function loadTracks(): Promise<void> {
    try {
        const res = await fetch('/api/tracks');
        if (!res.ok) throw new Error('Failed to load tracks: ' + res.status);
        tracks = await res.json();
        document.getElementById('total-tracks')!.textContent = String(tracks.length);
    } catch (e) {
        console.error('Failed to load tracks', e);
        document.getElementById('total-tracks')!.textContent = '5';
    }
}

loadTracks().then(() => loadData());

let quoteInterval: number | undefined;
async function loadRandomQuote(): Promise<void> {
    try {
        const res = await fetch('/api/quote/random');
        const quote = await res.json();
        const quoteText = document.getElementById('random-quote')!;
        const quoteAuthor = document.getElementById('quote-author')!;
        quoteText.classList.add('updating');
        setTimeout(() => {
            quoteText.textContent = quote.text || "The engines roar as these legends battle for glory!";
            quoteAuthor.textContent = `— ${quote.author || 'Commentator'}`;
            quoteText.classList.remove('updating');
        }, 300);
    } catch (err) {
        console.error('Failed to load quote:', err);
    }
}

let activeFlags: Record<string, boolean> = {};

function showFlagFullscreen(flagClass: string, icon: string, label: string, sub: string): void {
    const el = document.getElementById('flagModal');
    const body = document.getElementById('flagModalBody');
    const content = document.getElementById('flagModalContent');
    if (!el || !body || !content) return;
    content.className = 'modal-content border-0 rounded-0 ' + flagClass;
    body.innerHTML = `
        <div style="animation:flagPulse 0.6s ease-in-out infinite">
            <div style="font-size:8rem;margin-bottom:1.5rem"><i class="fa-solid ${icon}"></i></div>
            <div style="font-size:6rem;font-weight:900;letter-spacing:12px;text-transform:uppercase;font-family:'Bebas Neue',sans-serif;line-height:1.1">${label}</div>
            <div style="font-size:1.4rem;opacity:0.85;margin-top:1rem;letter-spacing:4px;text-transform:uppercase">${sub}</div>
        </div>`;
    el.style.display = 'block';
    el.classList.add('show');
    document.body.classList.add('modal-open');
}

function hideFlagFullscreen(): void {
    const el = document.getElementById('flagModal');
    if (!el) return;
    el.style.display = 'none';
    el.classList.remove('show');
    document.body.classList.remove('modal-open');
}

function handleFlagCommand(cmd: any): void {
    const isOn = cmd.state !== 'off';

    if (cmd.flag === 'clear') {
        activeFlags = {};
        hideFlagFullscreen();
        return;
    }

    if (cmd.flag === 'safety' || cmd.flag === 'red' || cmd.flag === 'chequered') {
        if (isOn) {
            activeFlags[cmd.flag] = true;
            const map: Record<string, any> = {
                safety: { cls: 'flag-safety', icon: 'fa-car-side', label: 'SAFETY CAR', sub: 'Yellow flag — no overtaking' },
                red: { cls: 'flag-red', icon: 'fa-circle-exclamation', label: 'RED FLAG', sub: 'Race suspended — return to pits' },
                chequered: { cls: 'flag-chequered', icon: 'fa-flag-checkered', label: 'CHEQUERED FLAG', sub: 'Race finished!' },
            };
            const m = map[cmd.flag];
            showFlagFullscreen(m.cls, m.icon, m.label, m.sub);
        } else {
            delete activeFlags[cmd.flag];
            if (Object.keys(activeFlags).length === 0) hideFlagFullscreen();
        }
        return;
    }

    if (cmd.flag === 'yellow') {
        if (isOn) {
            showFlagFullscreen('flag-yellow', 'fa-flag', 'YELLOW FLAG', 'Caution — danger ahead');
            setTimeout(() => {
                if (Object.keys(activeFlags).length === 0) hideFlagFullscreen();
            }, 5000);
        } else {
            if (Object.keys(activeFlags).length === 0) hideFlagFullscreen();
        }
        return;
    }

    if (cmd.flag === 'blue') {
        showBootstrapToast('primary', 'BLUE FLAG', `${cmd.racer_name} — let faster car through!`);
    } else if (cmd.flag === 'blackwhite') {
        showBootstrapToast('secondary', 'BLACK & WHITE FLAG', `${cmd.racer_name} — unsportsmanlike conduct warning!`);
    }
}

function showBootstrapToast(color: string, title: string, message: string): void {
    const container = document.getElementById('toast-container')!;
    const id = 'toast-' + Date.now();
    const html = `
        <div id="${id}" class="toast align-items-center text-bg-${color} border-0 mb-2" role="alert">
            <div class="d-flex">
                <div class="toast-body">
                    <strong>${title}</strong><br>${message}
                </div>
                <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
            </div>
        </div>
    `;
    container.insertAdjacentHTML('beforeend', html);
    const el = document.getElementById(id);
    if (el) {
        const toast = new (window as any).bootstrap.Toast(el, { autohide: true, delay: 8000 });
        toast.show();
        el.addEventListener('hidden.bs.toast', () => el.remove());
    }
}

function startQuoteRotation(): void {
    loadRandomQuote();
    if (quoteInterval) clearInterval(quoteInterval);
    quoteInterval = window.setInterval(loadRandomQuote, 15000);
}

async function loadData(): Promise<void> {
    try {
        const [raceResp, racerResp, seasonsResp, geoResp] = await Promise.all([
            fetch('/api/race-info'),
            fetch('/api/racers'),
            fetch('/api/seasons'),
            fetch('/api/tracks/geojson').catch(() => new Response('{}'))
        ]);
        const race: RaceInfo = await raceResp.json();
        racers = await racerResp.json();
        const seasons = await seasonsResp.json();
        try { trackGeoJSON = await geoResp.json(); } catch { trackGeoJSON = {}; }
        document.getElementById('race-country')!.innerHTML = `${race.country} <i class="fa-solid fa-location-dot ms-2 text-warning"></i>`;
        document.getElementById('race-track')!.textContent = race.track;
        document.getElementById('race-laps')!.textContent = String(race.laps);
        document.getElementById('total-drivers')!.textContent = String(racers.length);
        document.getElementById('total-seasons')!.textContent = String(Array.isArray(seasons) ? seasons.length : 1);
        currentTrack = race.track_id || 'monza';
        const track = tracks.find(t => t.id === currentTrack);
        if (track) {
            document.getElementById('race-length')!.textContent = String(track.length_km);
            document.getElementById('race-record')!.textContent = track.lap_record;
        }
        initMap();
        renderRacers();
        loadQualificationGrid();
        startQuoteRotation();
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'flag') {
                handleFlagCommand(data);
            } else if (Array.isArray(data)) {
                racers = data;
                renderRacers();
                loadQualificationGrid();
            }
        };
        ws.onclose = () => setTimeout(loadData, 5000);
    } catch (err) {
        console.error("Failed to load data:", err);
    }
}

(window as any).showDriverStats = showDriverStats;

