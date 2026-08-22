import './theme';
import { escapeHtml } from './toast';
import { weatherIcon, weatherLabel, formatGrip } from './weather';
declare const Chart: any;
let pointsChart: any, winsChart: any, battleChart: any, incidentsChart: any;
let seasonScope: number[] = [];
let seasonsCache: any[] = [];
let comparisonChart: any;

function scopeQueryString(): string {
    return seasonScope.length ? `?season_ids=${seasonScope.join(',')}` : '';
}

function parseScopeFromUrl(): void {
    const raw = new URLSearchParams(window.location.search).get('seasons');
    if (!raw || raw === 'all') {
        seasonScope = [];
        return;
    }
    seasonScope = raw.split(',').map(v => parseInt(v, 10)).filter(v => Number.isFinite(v) && v > 0);
}

function syncSeasonControl(): void {
    const container = document.getElementById('stats-season-select');
    if (!container) return;
    container.querySelectorAll('.season-check').forEach((cb) => {
        const input = cb as HTMLInputElement;
        input.checked = seasonScope.includes(parseInt(input.value, 10));
    });
    const allBox = document.getElementById('season-all') as HTMLInputElement | null;
    if (allBox) allBox.checked = seasonScope.length === 0;
}

function applyScope(): void {
    const params = new URLSearchParams(window.location.search);
    params.set('seasons', seasonScope.length ? seasonScope.join(',') : 'all');
    history.replaceState(null, '', `${window.location.pathname}?${params.toString()}`);
    loadSeasonStats().then(() => loadDeeperStats(scopeQueryString()));
}

function buildSeasonControl(seasons: any[]): void {
    const container = document.getElementById('stats-season-select');
    if (!container) return;
    if (container.querySelector('.season-check')) {
        syncSeasonControl();
        return;
    }
    seasons.forEach((s: any) => {
        const row = document.createElement('div');
        row.className = 'form-check';
        row.innerHTML = `<input class="form-check-input season-check" type="checkbox" value="${s.id}" id="season-${s.id}"><label class="form-check-label" for="season-${s.id}">${escapeHtml(String(s.name))} (${escapeHtml(String(s.status))})</label>`;
        container.appendChild(row);
    });
    const allBox = document.getElementById('season-all') as HTMLInputElement | null;
    allBox?.addEventListener('change', () => {
        if (allBox.checked) {
            seasonScope = [];
            syncSeasonControl();
            applyScope();
        }
    });
    container.querySelectorAll('.season-check').forEach((cb) => {
        cb.addEventListener('change', () => {
            seasonScope = Array.from(container.querySelectorAll('.season-check:checked'))
                .map((el) => parseInt((el as HTMLInputElement).value, 10))
                .filter(v => Number.isFinite(v) && v > 0);
            syncSeasonControl();
            applyScope();
        });
    });
    syncSeasonControl();
}

function getCanvas(id: string): CanvasRenderingContext2D | null {
    const el = document.getElementById(id) as HTMLCanvasElement | null;
    return el?.getContext('2d') || null;
}

async function loadSeasonStats(): Promise<void> {
    try {
        const seasonsRes = await fetch('/api/seasons');
        const seasons = await seasonsRes.json();
        seasonsCache = Array.isArray(seasons) ? seasons : [];
        buildSeasonControl(seasonsCache);
        const scopeQS = scopeQueryString();

        const [racersRes, statsRes, snapshotsRes] = await Promise.all([
            fetch('/api/racers'),
            fetch(`/api/racer-stats${scopeQS}`),
            fetch(`/api/rounds${scopeQS}`)
        ]);

        const racers = await racersRes.json();
        const allStats = await statsRes.json();
        const snapshots = await snapshotsRes.json();

        const hasStats = Array.isArray(allStats) && allStats.length > 0;
        const hasSnapshots = Array.isArray(snapshots) && snapshots.length > 0;

        document.getElementById('total-races')!.textContent = String(hasSnapshots ? snapshots.length : 0);
        document.getElementById('total-drivers')!.textContent = String(racers.length);

        const totalFL = hasStats ? allStats.reduce((sum: number, s: any) => sum + (s.fastest_laps || 0), 0) : 0;
        document.getElementById('fastest-laps')!.textContent = String(totalFL);

        const driverData = hasStats ? allStats.filter((s: any) => s.races > 0) : [];
        const hasDrivers = driverData.length > 0;

        let allScores: any[] = [];
        if (hasSnapshots) {
            const ids = snapshots.map((s: any) => s.id).join(',');
            const batchRes = await fetch(`/api/rounds/batch?ids=${ids}`);
            allScores = await batchRes.json() as any[];
            renderPointsChart(snapshots, allScores);
            renderBattleChart(allScores);
            renderTrackStatsTable(allScores);
            renderIncidentsOverTime(snapshots, allScores);
        } else {
            document.getElementById('championships')!.textContent = '0';
            document.querySelector('#track-stats-table tbody')!.innerHTML = '<tr><td colspan="3" class="text-center text-muted py-4">No round snapshots yet</td></tr>';
            document.querySelector('#incidents-over-time-table tbody')!.innerHTML = '<tr><td colspan="2" class="text-center text-muted py-4">No round snapshots yet</td></tr>';
        }

        if (hasDrivers) {
            renderDriverStatsTable(allStats, racers);
            renderWinsChart(driverData, racers);
            renderPointsLeaderboard(allStats, racers, allScores);
        } else {
            document.querySelector('#driver-stats-table tbody')!.innerHTML = '<tr><td colspan="8" class="text-center text-muted py-4">No driver stats yet</td></tr>';
            document.querySelector('#points-body')!.innerHTML = '<tr><td colspan="7" class="text-center text-muted py-4">No points data yet</td></tr>';
        }

        // Comparison mode: 2+ seasons selected → per-racer × per-season card.
        if (seasonScope.length >= 2) {
            const perSeason = await Promise.all(seasonScope.map(id =>
                fetch(`/api/racer-stats?season_id=${id}`).then(r => r.json())
            ));
            renderComparisonCard(perSeason);
        } else {
            document.getElementById('comparison-card')?.classList.add('d-none');
        }

        const banner = document.getElementById('scope-empty-banner');
        if (banner) banner.classList.toggle('d-none', !(seasonScope.length > 0 && !hasStats && !hasSnapshots));
    } catch (err) {
        console.error('Failed to load stats:', err);
    }
}

function renderPointsChart(snapshots: any[], allScores: any[]): void {
    const ctx = getCanvas('points-chart');
    if (!ctx) return;

    const labels = snapshots.map((s: any) => s.race_name || `R${s.round}`);
    const colors = ['#ff4444', '#4444ff', '#44ff44', '#ffff44', '#ff00ff'];

    const N = snapshots.length;
    const racerData: Record<number, { name: string; roundPts: number[]; cum: number[] }> = {};
    allScores.forEach((snap: any, i: number) => {
        (snap.scores || []).forEach((sc: any) => {
            if (!racerData[sc.racer_id]) {
                racerData[sc.racer_id] = { name: sc.racer_name, roundPts: new Array(N).fill(0), cum: new Array(N).fill(0) };
            }
            racerData[sc.racer_id].roundPts[i] = sc.points;
        });
    });

    for (const id in racerData) {
        const d = racerData[id];
        let running = 0;
        for (let i = 0; i < N; i++) {
            running += d.roundPts[i];
            d.cum[i] = running;
        }
    }

    const sorted = Object.entries(racerData)
        .sort(([, a]: any, [, b]: any) => {
            const aLast = a.cum[a.cum.length - 1] || 0;
            const bLast = b.cum[b.cum.length - 1] || 0;
            if (bLast !== aLast) return bLast - aLast;
            return a.name.localeCompare(b.name);
        })
        .slice(0, 5);

    const datasets = sorted.map(([id, d]: [string, any], i: number) => ({
        label: d.name,
        data: d.cum,
        borderColor: colors[i % colors.length],
        backgroundColor: colors[i % colors.length] + '20',
        fill: true,
        tension: 0.4,
    }));

    const championships = Math.max(...Object.values(racerData).map((d: any) => {
        let champCount = 0;
        for (let r = 0; r < N; r++) {
            let topScore = 0;
            for (const id in racerData) {
                if (racerData[id].roundPts[r] > topScore) topScore = racerData[id].roundPts[r];
            }
            if (topScore > 0 && d.roundPts[r] === topScore) champCount++;
        }
        return champCount;
    }), 0);
    document.getElementById('championships')!.textContent = String(championships);

    if (pointsChart) pointsChart.destroy();
    pointsChart = new Chart(ctx, {
        type: 'line',
        data: { labels, datasets },
        options: {
            responsive: true,
            maintainAspectRatio: true,
            plugins: {
                legend: { position: 'bottom', labels: { boxWidth: 12, padding: 15 } },
            },
            scales: {
                y: { beginAtZero: true, title: { display: true, text: 'Points' } },
                x: { title: { display: true, text: 'Round' } },
            },
        },
    });
}

function renderWinsChart(driverData: any[], racers: any[]): void {
    const ctx = getCanvas('wins-chart');
    if (!ctx) return;

    const nameMap: Record<number, string> = {};
    racers.forEach((r: any) => { nameMap[r.id] = r.name; });

    const sorted = [...driverData].sort((a: any, b: any) => (b.gold || 0) - (a.gold || 0)).slice(0, 5);

    if (winsChart) winsChart.destroy();

    if (sorted.length === 0 || sorted.every((d: any) => !d.gold)) {
        winsChart = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['No wins yet'],
                datasets: [{
                    data: [1],
                    backgroundColor: ['#444'],
                }],
            },
            options: {
                responsive: true,
                maintainAspectRatio: true,
                plugins: { legend: { position: 'bottom', labels: { boxWidth: 12, padding: 15 } } },
            },
        });
        return;
    }
    winsChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: sorted.map((d: any) => nameMap[d.racer_id] || d.racer_name || `Racer #${d.racer_id}`),
            datasets: [{
                data: sorted.map((d: any) => d.gold || 0),
                backgroundColor: ['#ffd700', '#c0c0c0', '#cd7f32', '#ff6b6b', '#4ecdc4'],
            }],
        },
        options: {
            responsive: true,
            maintainAspectRatio: true,
            plugins: { legend: { position: 'bottom', labels: { boxWidth: 12, padding: 15 } } },
        },
    });
}

function renderDriverStatsTable(allStats: any[], racers: any[]): void {
    const tbody = document.querySelector('#driver-stats-table tbody')!;
    const sorted = [...allStats].sort((a: any, b: any) => (b.points || 0) - (a.points || 0) || (b.gold || 0) - (a.gold || 0));

    const nameMap: Record<number, string> = {};
    racers.forEach((r: any) => { nameMap[r.id] = r.name; });

    tbody.innerHTML = sorted.map((s: any) => {
        const totalPodiums = (s.gold || 0) + (s.silver || 0) + (s.bronze || 0);
        const name = nameMap[s.racer_id] || s.racer_name || `Racer #${s.racer_id}`;
        return `
            <tr>
                <td>${name}</td>
                <td>${s.races || 0}</td>
                <td class="text-warning fw-bold">${s.gold || s.wins || 0}</td>
                <td><span class="text-warning">${s.gold || 0}</span> / <span class="text-secondary">${s.silver || 0}</span> / <span class="bronze-text">${s.bronze || 0}</span></td>
                <td>${totalPodiums}</td>
                <td>${s.points || 0}</td>
                <td>${s.spins || 0}</td>
                <td>${s.overheated || 0}</td>
            </tr>
        `;
    }).join('');
}

function renderTrackStatsTable(allScores: any[]): void {
    const tbody = document.querySelector('#track-stats-table tbody')!;
    const trackMap: Record<string, any> = {};

    allScores.forEach((snap: any, i: number) => {
        const trackName = snap.race_name || `Round ${i + 1}`;
        if (!trackMap[trackName]) {
            trackMap[trackName] = { track: trackName, races: 0, winner: null };
        }
        trackMap[trackName].races++;
        const scores = snap.scores || [];
        const top = scores.reduce((best: any, s: any) => (!best || s.points > best.points ? s : best), null);
        if (top) trackMap[trackName].winner = top.racer_name;
    });

    const sorted = Object.values(trackMap).sort((a: any, b: any) => b.races - a.races);

    tbody.innerHTML = sorted.map((t: any) => `
        <tr>
            <td>${t.track}</td>
            <td>${t.races}</td>
            <td>${t.winner || '-'}</td>
        </tr>
    `).join('');
}

function renderIncidentsOverTime(snapshots: any[], allScores: any[]): void {
    // Chart: season-wide spins and overheated totals per round
    const ctx = getCanvas('incidents-chart');
    if (ctx) {
        const labels = snapshots.map((s: any) => s.race_name || `R${s.round}`);
        const roundTotals = allScores.map((snap: any) => {
            let spins = 0, overheated = 0;
            (snap.scores || []).forEach((sc: any) => {
                spins += sc.spins || 0;
                overheated += sc.overheated || 0;
            });
            return { spins, overheated };
        });

        if (incidentsChart) incidentsChart.destroy();
        incidentsChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels,
                datasets: [
                    { label: 'Spins', data: roundTotals.map((t: any) => t.spins), borderColor: '#4ecdc4', backgroundColor: '#4ecdc420', fill: true, tension: 0.4 },
                    { label: 'Overheated', data: roundTotals.map((t: any) => t.overheated), borderColor: '#e74c3c', backgroundColor: '#e74c3c20', fill: true, tension: 0.4 },
                ],
            },
            options: {
                responsive: true,
                maintainAspectRatio: true,
                plugins: { legend: { position: 'bottom', labels: { boxWidth: 12, padding: 15 } } },
                scales: {
                    y: { beginAtZero: true, title: { display: true, text: 'Incidents' } },
                    x: { title: { display: true, text: 'Round' } },
                },
            },
        });
    }

    // Table: per driver per round, cell shows "spins/overheated"
    const tbody = document.querySelector('#incidents-over-time-table tbody')!;
    const thead = document.querySelector('#incidents-over-time-table thead tr')!;

    const driverOrder: { id: number; name: string; pts: number }[] = [];
    allScores.forEach((snap: any) => {
        (snap.scores || []).forEach((sc: any) => {
            if (!sc.racer_id) return;
            const existing = driverOrder.find((d) => d.id === sc.racer_id);
            if (existing) {
                existing.pts += sc.points || 0;
            } else {
                driverOrder.push({ id: sc.racer_id, name: sc.racer_name || `Racer #${sc.racer_id}`, pts: sc.points || 0 });
            }
        });
    });
    driverOrder.sort((a, b) => b.pts - a.pts);

    thead.innerHTML = '<th>Round</th>' + driverOrder.map((d) => `<th>${escapeHtml(d.name)}</th>`).join('');

    if (driverOrder.length === 0) {
        tbody.innerHTML = '<tr><td colspan="2" class="text-center text-muted py-4">No round snapshots yet</td></tr>';
        return;
    }

    tbody.innerHTML = allScores.map((snap: any, i: number) => {
        const label = snap.race_name || `Round ${i + 1}`;
        const byRacer: Record<number, any> = {};
        (snap.scores || []).forEach((sc: any) => { byRacer[sc.racer_id] = sc; });
        const cells = driverOrder.map((d) => {
            const sc = byRacer[d.id];
            if (!sc) return '<td class="text-muted">–</td>';
            return `<td title="${escapeHtml(d.name)}: ${sc.spins || 0} spins, ${sc.overheated || 0} overheated">${sc.spins || 0}/${sc.overheated || 0}</td>`;
        }).join('');
        return `<tr><td class="fw-bold">${escapeHtml(label)}</td>${cells}</tr>`;
    }).join('');
}

function renderBattleChart(allScores: any[]): void {
    const ctx = getCanvas('battle-chart');
    if (!ctx) return;

    const latest = allScores[allScores.length - 1];
    if (!latest) return;
    const scores = (latest.scores || []).sort((a: any, b: any) => b.points - a.points).slice(0, 4);
    if (scores.length === 0 || scores.every((s: any) => s.points === 0)) return;

    if (battleChart) battleChart.destroy();
    battleChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: scores.map((s: any) => s.racer_name),
            datasets: [{
                label: 'Points',
                data: scores.map((s: any) => s.points),
                backgroundColor: ['#ffd700', '#c0c0c0', '#cd7f32', '#ff6b6b'],
            }],
        },
        options: {
            indexAxis: 'y',
            responsive: true,
            maintainAspectRatio: true,
            plugins: { legend: { display: false } },
            scales: { x: { beginAtZero: true } },
        },
    });
}

function renderPointsLeaderboard(allStats: any[], racers: any[], allScores?: any[]): void {
    const tbody = document.querySelector('#points-body')!;
    const sorted = [...allStats].filter((s: any) => s.races > 0).sort((a: any, b: any) => (b.points || 0) - (a.points || 0));

    const avgFinishMap: Record<number, number> = {};
    if (allScores && allScores.length > 0) {
        const posSums: Record<number, { total: number; count: number }> = {};
        allScores.forEach((snap: any) => {
            (snap.scores || []).forEach((sc: any) => {
                if (sc.racer_id && sc.position > 0) {
                    if (!posSums[sc.racer_id]) posSums[sc.racer_id] = { total: 0, count: 0 };
                    posSums[sc.racer_id].total += sc.position;
                    posSums[sc.racer_id].count++;
                }
            });
        });
        Object.entries(posSums).forEach(([id, data]) => {
            avgFinishMap[Number(id)] = Math.round((data.total / data.count) * 10) / 10;
        });
    }

    const nameMap: Record<number, string> = {};
    const carMap: Record<number, string> = {};
    racers.forEach((r: any) => {
        nameMap[r.id] = r.name;
        carMap[r.id] = r.car_name || '';
    });

    tbody.innerHTML = sorted.map((s: any, i: number) => {
        const podiums = (s.gold || 0) + (s.silver || 0) + (s.bronze || 0);
        const name = nameMap[s.racer_id] || s.racer_name || `Racer #${s.racer_id}`;
        const car = carMap[s.racer_id] || '-';
        const avgFinish = avgFinishMap[s.racer_id] !== undefined ? avgFinishMap[s.racer_id].toFixed(1) : '-';
        return `
            <tr>
                <td>${i + 1}</td>
                <td>${name}</td>
                <td>${car}</td>
                <td class="text-end">${s.points || 0}</td>
                <td class="text-end">${s.gold || s.wins || 0}</td>
                <td class="text-end">${podiums}</td>
                <td class="text-end">${avgFinish}</td>
            </tr>
        `;
    }).join('');
}

function seasonName(id: number): string {
    const s = seasonsCache.find((x: any) => x.id === id);
    return s ? String(s.name) : `Season ${id}`;
}

function renderComparisonCard(perSeason: any[][]): void {
    const card = document.getElementById('comparison-card');
    if (!card) return;
    card.classList.remove('d-none');

    // Union of drivers across the selected seasons, sorted by total points.
    const totals = new Map<number, { name: string; total: number }>();
    perSeason.flat().forEach((s: any) => {
        const cur = totals.get(s.racer_id) || { name: s.racer_name || `Racer ${s.racer_id}`, total: 0 };
        cur.total += s.points || 0;
        totals.set(s.racer_id, cur);
    });
    const order = Array.from(totals.entries()).sort((a, b) => b[1].total - a[1].total);

    const head = document.getElementById('comparison-head');
    if (head) {
        head.innerHTML = '<th>Driver</th>' +
            seasonScope.map(id => `<th>${escapeHtml(seasonName(id))}</th>`).join('') +
            '<th>Total Points</th>';
    }

    const body = document.getElementById('comparison-body');
    if (body) {
        body.innerHTML = order.map(([racerId, info]) => {
            let totalPts = 0;
            const cells = perSeason.map((stats) => {
                const s = stats.find((x: any) => x.racer_id === racerId);
                const pts = s?.points || 0;
                totalPts += pts;
                const races = s?.races || 0;
                const wins = s?.wins || 0;
                const podiums = (s?.gold || 0) + (s?.silver || 0) + (s?.bronze || 0);
                return `<td>${races} races · ${wins} wins · ${podiums} podiums · <strong>${pts}</strong> pts</td>`;
            }).join('');
            return `<tr><td>${escapeHtml(info.name)}</td>${cells}<td class="fw-bold">${totalPts}</td></tr>`;
        }).join('');
    }

    // Grouped bar chart: points per racer per season.
    comparisonChart?.destroy();
    const ctx = getCanvas('comparison-chart');
    if (!ctx) return;
    const labels = order.slice(0, 8).map(([, info]) => info.name);
    const palette = ['#fbbf24', '#3b82f6', '#ef4444', '#22c55e', '#a855f7', '#f97316', '#14b8a6', '#eab308'];
    const datasets = seasonScope.map((id, i) => ({
        label: seasonName(id),
        data: order.slice(0, 8).map(([racerId]) => {
            const s = (perSeason[i] || []).find((x: any) => x.racer_id === racerId);
            return s?.points || 0;
        }),
        backgroundColor: palette[i % palette.length]
    }));
    comparisonChart = new Chart(ctx, {
        type: 'bar',
        data: { labels, datasets },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { position: 'bottom' } },
            scales: { y: { beginAtZero: true } }
        }
    });
}

function switchSeason(value: string): void {
    seasonScope = value ? [parseInt(value, 10)].filter(v => Number.isFinite(v) && v > 0) : [];
    syncSeasonControl();
    applyScope();
}
(window as any).switchSeason = switchSeason;

parseScopeFromUrl();
loadSeasonStats().then(() => loadDeeperStats(scopeQueryString()));

// Deeper Stats
async function loadDeeperStats(scopeQS: string = ''): Promise<void> {
    try {
        const [deltaRes, consistencyRes, incidentsRes, paceRes] = await Promise.all([
            fetch(`/api/stats/qualifying-delta${scopeQS}`),
            fetch(`/api/stats/consistency${scopeQS}`),
            fetch(`/api/stats/incidents${scopeQS}`),
            fetch(`/api/stats/pace-heatmap${scopeQS}`)
        ]);

        const delta = await deltaRes.json();
        const consistency = await consistencyRes.json();
        const incidents = await incidentsRes.json();
        const pace = await paceRes.json();

        // Qualifying Delta
        const deltaBody = document.getElementById('qualifying-delta-body')!;
        if (Array.isArray(delta) && delta.length > 0) {
            deltaBody.innerHTML = delta.map((d: any) => `
                <tr>
                    <td>${d.racer_name}</td>
                    <td>${d.races}</td>
                    <td>${d.avg_race_position}</td>
                    <td class="fw-bold">${d.total_points}</td>
                </tr>
            `).join('');
        }

        // Consistency
        const consBody = document.getElementById('consistency-body')!;
        if (Array.isArray(consistency) && consistency.length > 0) {
            consBody.innerHTML = consistency.map((c: any) => `
                <tr>
                    <td>${c.racer_name}</td>
                    <td>${c.races}</td>
                    <td>${c.avg_position}</td>
                    <td>${c.std_dev}</td>
                    <td><span class="badge bg-${c.consistency_score >= 80 ? 'success' : c.consistency_score >= 60 ? 'warning' : 'danger'}">${c.consistency_score}</span></td>
                    <td>${c.best_position}</td>
                    <td>${c.worst_position}</td>
                </tr>
            `).join('');
        }

        // Incidents
        const incBody = document.getElementById('incidents-body')!;
        if (Array.isArray(incidents) && incidents.length > 0) {
            const eventIcons: Record<string, string> = { overtake: '🏎️', crash: '💥', spin: '🔄', safety_car: '🚗', pit_stop: '🔧' };
            incBody.innerHTML = incidents.slice(-20).reverse().map((e: any) => `
                <div class="d-flex justify-content-between py-1 border-bottom border-secondary border-opacity-25">
                    <span>${eventIcons[e.event_type] || '📌'} <strong>${e.racer1_name}</strong>${e.racer2_name ? ` vs ${e.racer2_name}` : ''}</span>
                    <small class="opacity-75">${e.event_type}${e.lap ? ` · Lap ${e.lap}` : ''}</small>
                </div>
            `).join('');
        }

        // Pace Heatmap
        const paceBody = document.getElementById('pace-heatmap-body')!;
        if (Array.isArray(pace) && pace.length > 0) {
            const grouped: Record<string, any[]> = {};
            const hasWeather = pace.some((p: any) => p.condition && p.condition !== 'dry');
            pace.forEach((p: any) => {
                if (!grouped[p.racer_name]) grouped[p.racer_name] = [];
                grouped[p.racer_name].push(p);
            });
            const legend = hasWeather ? `<div class="small text-muted mb-2">Weather: ☀️ Dry · 🌦️ Damp · 🌧️ Wet · ⛈️ Torrential</div>` : '';
            paceBody.innerHTML = legend + Object.entries(grouped).map(([name, laps]: [string, any[]]) => `
                <div class="mb-2">
                    <strong>${escapeHtml(name)}</strong>
                    <div class="d-flex flex-wrap gap-1 mt-1">
                        ${laps.slice(-10).map((l: any) => {
                            const cond = l.condition || 'dry';
                            const badge = cond !== 'dry' ? `<span title="${escapeHtml(weatherLabel(cond))} · ${formatGrip(l.grip_modifier || 1)}">${weatherIcon(cond)}</span> ` : '';
                            return `<div class="small px-2 py-1 rounded" style="background:${l.turbo_used ? '#2ecc71' : l.heat_generated > 2 ? '#e74c3c' : '#444'};" title="${escapeHtml(weatherLabel(cond))} · ${formatGrip(l.grip_modifier || 1)}">
                                ${badge}L${l.lap} P${l.position}${l.turbo_used ? ' ⚡' : ''}${l.heat_generated ? ' 🔥' + l.heat_generated : ''}
                            </div>`;
                        }).join('')}
                    </div>
                </div>
            `).join('');
        }
    } catch (err) {
        console.error('Failed to load deeper stats:', err);
    }
}


