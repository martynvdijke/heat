const themeToggle = document.getElementById('theme-toggle')!;
const themeIcon = themeToggle.querySelector('i')!;
const setTheme = (theme: string): void => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
    themeIcon.className = theme === 'dark' ? 'fa-solid fa-sun' : 'fa-solid fa-moon';
};
const savedTheme = localStorage.getItem('theme') || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
setTheme(savedTheme);
themeToggle.addEventListener('click', () => setTheme(document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark'));

interface Racer {
    id: number;
    name: string;
    profile_picture: string;
    car_color: string;
    car_name: string;
    points: number;
    rank: number;
    position: number;
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

interface RaceHistory {
    ID: number;
    Name: string;
    Date: string;
    Country: string;
    Track: string;
    Results?: RaceResult[];
    TotalLaps: number;
}

interface RaceResult {
    Position: number;
    RacerName: string;
    Points: number;
    FastestLap: boolean;
}

interface RacerStats {
    Races: number;
    Wins: number;
    Podiums: number;
    FastestLaps: number;
    Points: number;
    DNF: number;
}

interface StatsResponse {
    stats: RacerStats;
    racer: Racer;
}

interface GeoJSONFeature {
    type: string;
    properties: Record<string, string>;
    geometry: {
        type: string;
        coordinates: number[][];
    };
}

const trackGeoJSON: Record<string, GeoJSONFeature> = {
    monza: {
        "type": "Feature",
        "properties": {"id": "it-1922", "Location": "Monza", "Name": "Autodromo Nazionale Monza"},
        "geometry": {
            "type": "LineString",
            "coordinates": [
                [9.281223, 45.618975], [9.281692, 45.622832], [9.281905, 45.624449], [9.281928, 45.624515],
                [9.281994, 45.624553], [9.282076, 45.624562], [9.282177, 45.624553], [9.282272, 45.624548],
                [9.28236, 45.624553], [9.28242, 45.624586], [9.282467, 45.624633], [9.282479, 45.624675],
                [9.282479, 45.624722], [9.282147, 45.625604], [9.2821, 45.625844], [9.282082, 45.626061],
                [9.282106, 45.626297], [9.28223, 45.627447], [9.282278, 45.627635], [9.282337, 45.62781],
                [9.282426, 45.627998], [9.282544, 45.628201], [9.28268, 45.628399], [9.282881, 45.628621],
                [9.283088, 45.628804], [9.283367, 45.629012], [9.283669, 45.629196], [9.284006, 45.629365],
                [9.284367, 45.629511], [9.284775, 45.629653], [9.285184, 45.629752], [9.285634, 45.629841],
                [9.286143, 45.629903], [9.286664, 45.629936], [9.288694, 45.630058], [9.29118, 45.630162],
                [9.291328, 45.630171], [9.291405, 45.6302], [9.29147, 45.630247], [9.291552, 45.63045],
                [9.291588, 45.630506], [9.291647, 45.630544], [9.291748, 45.630567], [9.292085, 45.63061],
                [9.292452, 45.630666], [9.292973, 45.630784], [9.295299, 45.631331], [9.2955, 45.631364],
                [9.29569, 45.631364], [9.295873, 45.63134], [9.296003, 45.631303], [9.296128, 45.631251],
                [9.296246, 45.631194], [9.296394, 45.631081], [9.296477, 45.630992], [9.296536, 45.630883],
                [9.296566, 45.630761], [9.296666, 45.629988], [9.296856, 45.628668], [9.29685, 45.62855],
                [9.29682, 45.628474], [9.296773, 45.628408], [9.296996, 45.628361], [9.296554, 45.628295],
                [9.295033, 45.62772], [9.293796, 45.627235], [9.293026, 45.626938], [9.292653, 45.626787],
                [9.292233, 45.62658], [9.291931, 45.626419], [9.291671, 45.626259], [9.290321, 45.625448],
                [9.289363, 45.62484], [9.287338, 45.623596], [9.286119, 45.622846], [9.285953, 45.622728],
                [9.285888, 45.622653], [9.285876, 45.622578], [9.285876, 45.622488], [9.285918, 45.62229],
                [9.28593, 45.622182], [9.28593, 45.622026], [9.285912, 45.621941], [9.285864, 45.621833],
                [9.285793, 45.621706], [9.285675, 45.621578], [9.285515, 45.621446], [9.285332, 45.621338],
                [9.285184, 45.621258], [9.285107, 45.621196], [9.285048, 45.621112], [9.284994, 45.620956],
                [9.28487, 45.620244], [9.28474, 45.619485], [9.283734, 45.612679], [9.283692, 45.612434],
                [9.283651, 45.61232], [9.283568, 45.612203], [9.283455, 45.612108], [9.283325, 45.612023],
                [9.283142, 45.611939], [9.282941, 45.611887], [9.28271, 45.611858], [9.282514, 45.611868],
                [9.282331, 45.611896], [9.282118, 45.611953], [9.281911, 45.612019], [9.281757, 45.612094],
                [9.281573, 45.612193], [9.281366, 45.612349], [9.281224, 45.61249], [9.2811, 45.612641],
                [9.280993, 45.61282], [9.280893, 45.613018], [9.280816, 45.613221], [9.28078, 45.613395],
                [9.280739, 45.613659], [9.280733, 45.613923], [9.280727, 45.614173], [9.280709, 45.614668],
                [9.280697, 45.615102], [9.280709, 45.615573], [9.280786, 45.61619], [9.281076, 45.618142],
                [9.281223, 45.618975]
            ]
        }
    },
    spa: {
        "type": "Feature",
        "properties": {"id": "be-spa", "Location": "Spa-Francorchamps", "Name": "Circuit de Spa-Francorchamps"},
        "geometry": {
            "type": "LineString",
            "coordinates": [
                [5.971389, 50.345833], [5.963056, 50.433333], [5.956944, 50.441667], [5.943056, 50.436111],
                [5.931944, 50.420833], [5.918333, 50.408333], [5.909444, 50.395], [5.900278, 50.370833],
                [5.903611, 50.351944], [5.918889, 50.343611], [5.934722, 50.343056], [5.948333, 50.346667],
                [5.958611, 50.351389], [5.969444, 50.345833]
            ]
        }
    },
    silverstone: {
        "type": "Feature",
        "properties": {"id": "uk-silverstone", "Location": "Silverstone", "Name": "Silverstone Circuit"},
        "geometry": {
            "type": "LineString",
            "coordinates": [
                [-1.016389, 52.073056], [-1.016667, 52.078333], [-1.015, 52.081667], [-1.008333, 52.081389],
                [-1.001944, 52.078333], [-0.996111, 52.073611], [-0.988611, 52.070278], [-0.980278, 52.066667],
                [-0.9725, 52.066667], [-0.968611, 52.069722], [-0.971389, 52.073611], [-0.984722, 52.076389],
                [-0.996944, 52.074722], [-1.007222, 52.072778], [-1.016389, 52.073056]
            ]
        }
    },
    monaco: {
        "type": "Feature",
        "properties": {"id": "mc-monaco", "Location": "Monaco", "Name": "Circuit de Monaco"},
        "geometry": {
            "type": "LineString",
            "coordinates": [
                [7.421667, 43.734722], [7.424722, 43.736667], [7.427778, 43.738056], [7.431111, 43.738611],
                [7.433889, 43.737222], [7.435278, 43.735278], [7.434444, 43.733333], [7.428333, 43.729722],
                [7.421389, 43.7275], [7.418611, 43.729167], [7.417778, 43.731667], [7.419722, 43.733889],
                [7.421667, 43.734722]
            ]
        }
    },
    interlagos: {
        "type": "Feature",
        "properties": {"id": "br-interlagos", "Location": "São Paulo", "Name": "Autódromo José Carlos Pace"},
        "geometry": {
            "type": "LineString",
            "coordinates": [
                [-46.699722, -23.701667], [-46.701389, -23.703333], [-46.703889, -23.705278], [-46.708333, -23.706389],
                [-46.711944, -23.705278], [-46.713611, -23.702778], [-46.712778, -23.700556], [-46.709444, -23.698333],
                [-46.703889, -23.697778], [-46.700833, -23.699167], [-46.699444, -23.701111], [-46.699722, -23.701667]
            ]
        }
    }
};

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
        const geojson = trackGeoJSON[trackId] || trackGeoJSON.monza;
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
    const geojson = trackGeoJSON[currentTrack] || trackGeoJSON.monza;
    racers.forEach(r => {
        const percentage = r.position / totalVakjes;
        const pos = getPointAtDistance(geojson, percentage);
        const colorMap: Record<string, string> = {
            'red': '#ff4444', 'blue': '#4444ff', 'green': '#44ff44',
            'yellow': '#ffff44', 'grey': '#aaaaaa', 'silver': '#aaaaaa',
            'black': '#333333', 'purple': '#9b59b6', 'orange': '#e67e22'
        };
        if (racerMarkers[r.id]) {
            racerMarkers[r.id].setLatLng(pos);
        } else {
            racerMarkers[r.id] = L.circleMarker(pos, {
                radius: 8, fillColor: colorMap[r.car_color] || '#ffffff',
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
                        <span class="driver-name"><span class="color-indicator ${r.car_color} me-2"></span>${r.name}</span>
                    </div>
                </td>
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
            <span class="color-indicator ${r.car_color}"></span> ${r.car_color}
        </div>
        <div class="row g-3">
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.Races || 0}</span><span class="stat-label">Races</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.Wins || 0}</span><span class="stat-label">Wins</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.Podiums || 0}</span><span class="stat-label">Podiums</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.FastestLaps || 0}</span><span class="stat-label">Fastest Laps</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.Points || r.points || 0}</span><span class="stat-label">Total Points</span></div></div>
            <div class="col-6"><div class="stat-box"><span class="stat-value">${s.DNF || 0}</span><span class="stat-label">DNF</span></div></div>
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
                    <div class="small text-muted"><span class="color-indicator ${r.car_color}"></span>${r.car_name}</div>
                    ${i === 0 ? '<i class="fa-solid fa-crown text-warning"></i>' : ''}
                </div>
            </div>
        `).join('');
    } catch (err) {
        console.error('Failed to load qualification grid:', err);
    }
}

async function loadRaceHistory(): Promise<void> {
    const res = await fetch('/api/race-history');
    const history: RaceHistory[] = await res.json();
    const container = document.getElementById('race-history-container')!;
    if (!history || history.length === 0) {
        container.innerHTML = '<div class="col-12 text-center text-muted py-5"><i class="fa-solid fa-inbox fa-3x mb-3"></i><p>No race history yet. Completed races will appear here.</p></div>';
        return;
    }
    container.innerHTML = history.map(h => {
        const results = h.Results || [];
        const winner = results.find(r => r.Position === 1);
        return `
            <div class="col-md-6 col-lg-4">
                <div class="history-card">
                    <div class="history-header">
                        <span class="history-date">${h.Date || 'Unknown Date'}</span>
                        <span class="history-winner"><i class="fa-solid fa-trophy me-1"></i>${winner ? winner.RacerName : 'TBD'}</span>
                    </div>
                    <div class="history-body">
                        <h5><i class="fa-solid fa-flag-checkered me-2"></i>${h.Country}</h5>
                        <p class="mb-1">${h.Track || 'Unknown Track'}</p>
                        <span class="badge bg-secondary">${h.TotalLaps} Laps</span>
                    </div>
                    <div class="history-results">
                        ${results.slice(0, 3).map(r => `
                            <div class="result-item ${r.Position === 1 ? 'gold' : ''} ${r.Position === 2 ? 'silver' : ''} ${r.Position === 3 ? 'bronze' : ''}">
                                <span class="pos">${r.Position}</span>
                                <span class="name">${r.RacerName || 'Unknown'}</span>
                                <span class="pts">${r.Points} pts</span>
                                ${r.FastestLap ? '<i class="fa-solid fa-stopwatch text-warning ms-1"></i>' : ''}
                            </div>
                        `).join('')}
                    </div>
                </div>
            </div>
        `;
    }).join('');
    document.getElementById('total-races')!.textContent = String(history.length);
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

function startQuoteRotation(): void {
    loadRandomQuote();
    if (quoteInterval) clearInterval(quoteInterval);
    quoteInterval = window.setInterval(loadRandomQuote, 15000);
}

async function loadData(): Promise<void> {
    try {
        const [raceResp, racerResp, historyResp] = await Promise.all([
            fetch('/api/race-info'),
            fetch('/api/racers'),
            fetch('/api/race-history')
        ]);
        const race: RaceInfo = await raceResp.json();
        racers = await racerResp.json();
        document.getElementById('race-country')!.innerHTML = `${race.country} <i class="fa-solid fa-location-dot ms-2 text-warning"></i>`;
        document.getElementById('race-track')!.textContent = race.track;
        document.getElementById('race-laps')!.textContent = String(race.laps);
        document.getElementById('total-drivers')!.textContent = String(racers.length);
        currentTrack = race.track_id || 'monza';
        const track = tracks.find(t => t.id === currentTrack);
        if (track) {
            document.getElementById('race-length')!.textContent = String(track.length_km);
            document.getElementById('race-record')!.textContent = track.lap_record;
        }
        initMap();
        renderRacers();
        loadRaceHistory();
        loadQualificationGrid();
        startQuoteRotation();
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
        ws.onmessage = (event) => {
            racers = JSON.parse(event.data);
            renderRacers();
            loadQualificationGrid();
        };
        ws.onclose = () => setTimeout(loadData, 5000);
    } catch (err) {
        console.error("Failed to load data:", err);
    }
}

fetch('/api/version').then(r => r.json()).then((d: {version: string}) => {
    document.getElementById('version-display')!.textContent = `v${d.version}`;
}).catch(() => {
    document.getElementById('version-display')!.textContent = 'v1.0.0';
});

(window as any).showDriverStats = showDriverStats;

export {};
