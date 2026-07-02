import './theme';
declare const Chart: any;
let pointsChart: any, winsChart: any, lapTimeChart: any, battleChart: any;

function getCanvas(id: string): CanvasRenderingContext2D | null {
    const el = document.getElementById(id) as HTMLCanvasElement | null;
    return el?.getContext('2d') || null;
}

async function loadSeasonStats(seasonId?: string): Promise<void> {
    try {
        const seasonsRes = await fetch('/api/seasons');
        const seasons = await seasonsRes.json();
        const select = document.getElementById('stats-season-select') as HTMLSelectElement;
        if (select.options.length <= 1) {
            (Array.isArray(seasons) ? seasons : []).forEach((s: any) => {
                const opt = document.createElement('option');
                opt.value = String(s.id);
                opt.textContent = `${s.name} (${s.status})`;
                select.appendChild(opt);
            });
        }
        const active = Array.isArray(seasons) ? seasons.find((s: any) => s.status === 'active') : null;
        const sid = seasonId || (active ? String(active.id) : '');
        if (sid && select) select.value = sid;
        const roundsUrl = sid ? `/api/rounds?season_id=${sid}` : '/api/rounds';

        const statsUrl = sid ? `/api/racer-stats?season_id=${sid}` : '/api/racer-stats';

        const [racersRes, statsRes, snapshotsRes] = await Promise.all([
            fetch('/api/racers'),
            fetch(statsUrl),
            fetch(roundsUrl)
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

        if (hasSnapshots) {
            const ids = snapshots.map((s: any) => s.id).join(',');
            const batchRes = await fetch(`/api/rounds/batch?ids=${ids}`);
            const allScores = await batchRes.json() as any[];
            renderPointsChart(snapshots, allScores);
            renderBattleChart(allScores);
            renderTrackStatsTable(allScores);
        } else {
            document.getElementById('championships')!.textContent = '0';
            document.querySelector('#track-stats-table tbody')!.innerHTML = '<tr><td colspan="4" class="text-center text-muted py-4">No round snapshots yet</td></tr>';
        }

        if (hasDrivers) {
            renderDriverStatsTable(allStats, racers);
            renderWinsChart(driverData, racers);
        } else {
            document.querySelector('#driver-stats-table tbody')!.innerHTML = '<tr><td colspan="6" class="text-center text-muted py-4">No driver stats yet</td></tr>';
        }
    } catch (err) {
        console.error('Failed to load stats:', err);
    }
}

function renderPointsChart(snapshots: any[], allScores: any[]): void {
    const ctx = getCanvas('points-chart');
    if (!ctx) return;

    const labels = snapshots.map((s: any) => s.race_name || `R${s.round}`);
    const colors = ['#ff4444', '#4444ff', '#44ff44', '#ffff44', '#ff00ff'];

    const racerPoints: Record<number, { name: string; pts: number[] }> = {};
    allScores.forEach((snap: any) => {
        (snap.scores || []).forEach((sc: any) => {
            if (!racerPoints[sc.racer_id]) {
                racerPoints[sc.racer_id] = { name: sc.racer_name, pts: [] };
            }
            racerPoints[sc.racer_id].pts.push(sc.points);
        });
    });

    const sorted = Object.entries(racerPoints)
        .sort(([, a]: any, [, b]: any) => {
            const aLast = a.pts[a.pts.length - 1] || 0;
            const bLast = b.pts[b.pts.length - 1] || 0;
            return bLast - aLast;
        })
        .slice(0, 5);

    const datasets = sorted.map(([id, d]: [string, any], i: number) => ({
        label: d.name,
        data: d.pts,
        borderColor: colors[i % colors.length],
        backgroundColor: colors[i % colors.length] + '20',
        fill: true,
        tension: 0.4,
    }));

    const championships = Math.max(...Object.values(racerPoints).map((d: any) => {
        const pts = d.pts;
        let champCount = 0;
        for (let r = 0; r < snapshots.length; r++) {
            const roundScores = allScores[r]?.scores || [];
            const topScore = Math.max(...roundScores.map((sc: any) => sc.points), 0);
            const thisRacer = roundScores.find((sc: any) => sc.racer_name === d.name);
            if (thisRacer && thisRacer.points === topScore && topScore > 0) {
                champCount++;
            }
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
            <td>--:--</td>
        </tr>
    `).join('');
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

function switchSeason(value: string): void {
    loadSeasonStats(value || undefined);
}

loadSeasonStats().then(() => loadDeeperStats());
(window as any).switchSeason = switchSeason;

// Deeper Stats
async function loadDeeperStats(): Promise<void> {
    try {
        const [deltaRes, consistencyRes, incidentsRes, paceRes] = await Promise.all([
            fetch('/api/stats/qualifying-delta'),
            fetch('/api/stats/consistency'),
            fetch('/api/stats/incidents'),
            fetch('/api/stats/pace-heatmap')
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
            pace.forEach((p: any) => {
                if (!grouped[p.racer_name]) grouped[p.racer_name] = [];
                grouped[p.racer_name].push(p);
            });
            paceBody.innerHTML = Object.entries(grouped).map(([name, laps]: [string, any[]]) => `
                <div class="mb-2">
                    <strong>${name}</strong>
                    <div class="d-flex flex-wrap gap-1 mt-1">
                        ${laps.slice(-10).map((l: any) => `
                            <div class="small px-2 py-1 rounded" style="background:${l.turbo_used ? '#2ecc71' : l.heat_generated > 2 ? '#e74c3c' : '#444'};">
                                L${l.lap} P${l.position}${l.turbo_used ? ' ⚡' : ''}${l.heat_generated ? ' 🔥' + l.heat_generated : ''}
                            </div>
                        `).join('')}
                    </div>
                </div>
            `).join('');
        }
    } catch (err) {
        console.error('Failed to load deeper stats:', err);
    }
}


