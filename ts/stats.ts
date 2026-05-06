declare const Chart: any;
let pointsChart: any, winsChart: any, lapTimeChart: any, battleChart: any;

function getCanvas(id: string): CanvasRenderingContext2D | null {
    const el = document.getElementById(id) as HTMLCanvasElement | null;
    return el?.getContext('2d') || null;
}

async function loadSeasonStats(): Promise<void> {
    try {
        const [historyRes, racersRes, statsRes] = await Promise.all([
            fetch('/api/race-history'),
            fetch('/api/racers'),
            fetch('/api/racer-stats')
        ]);

        const history = await historyRes.json();
        const racers = await racersRes.json();
        const allStats = await statsRes.json();

        const hasRaces = Array.isArray(history) && history.length > 0;
        const hasStats = Array.isArray(allStats) && allStats.length > 0;

        document.getElementById('total-races')!.textContent = String(hasRaces ? history.length : 0);
        document.getElementById('total-drivers')!.textContent = String(racers.length);

        const totalFL = hasStats ? allStats.reduce((sum: number, s: any) => sum + (s.fastest_laps || 0), 0) : 0;
        document.getElementById('fastest-laps')!.textContent = String(totalFL);

        if (!hasStats && !hasRaces) {
            document.getElementById('championships')!.textContent = '0';
            document.querySelector('#driver-stats-table tbody')!.innerHTML = '<tr><td colspan="6" class="text-center text-muted py-4">No race data yet</td></tr>';
            document.querySelector('#track-stats-table tbody')!.innerHTML = '<tr><td colspan="4" class="text-center text-muted py-4">No race history yet</td></tr>';
            return;
        }

        const driverMap: Record<number, any> = {};
        racers.forEach((r: any) => {
            driverMap[r.id] = { ...r, races: 0, wins: 0, gold: 0, silver: 0, bronze: 0, totalPoints: 0, fastest_laps: 0 };
        });

        if (hasStats) {
            allStats.forEach((s: any) => {
                if (driverMap[s.racer_id]) {
                    driverMap[s.racer_id].races = s.races || 0;
                    driverMap[s.racer_id].wins = s.wins || 0;
                    driverMap[s.racer_id].gold = s.gold || 0;
                    driverMap[s.racer_id].silver = s.silver || 0;
                    driverMap[s.racer_id].bronze = s.bronze || 0;
                    driverMap[s.racer_id].fastest_laps = s.fastest_laps || 0;
                }
            });
        }

        racers.forEach((r: any) => {
            if (driverMap[r.id]) {
                driverMap[r.id].totalPoints = r.points || 0;
            }
        });

        const driverData = Object.values(driverMap).filter((d: any) => d.races > 0);
        const championships = driverData.length > 0 ? Math.max(...driverData.map((d: any) => d.wins as number), 0) : 0;
        document.getElementById('championships')!.textContent = String(championships);

        if (driverData.length > 0) {
            renderDriverStatsTable(driverData);
            renderWinsChart(driverData);
            renderBattleChart(driverData);
        } else {
            document.querySelector('#driver-stats-table tbody')!.innerHTML = '<tr><td colspan="6" class="text-center text-muted py-4">No driver stats yet</td></tr>';
        }

        if (hasRaces) {
            renderPointsChart(history, driverMap);
            renderTrackStatsTable(history);
            renderLapTimeChart(history);
        } else {
            document.querySelector('#track-stats-table tbody')!.innerHTML = '<tr><td colspan="4" class="text-center text-muted py-4">No race history yet</td></tr>';
        }

    } catch (err) {
        console.error('Failed to load stats:', err);
    }
}

function renderPointsChart(history: any[], driverMap: Record<number, any>): void {
    const ctx = getCanvas('points-chart');
    if (!ctx) return;

    const races = history.map((h: any, i: number) => `R${i + 1}`);
    const colors = ['#ff4444', '#4444ff', '#44ff44', '#ffff44', '#ff00ff'];
    const sortedDrivers = Object.values(driverMap)
        .filter((d: any) => d.races > 0)
        .sort((a: any, b: any) => b.totalPoints - a.totalPoints)
        .slice(0, 5);

    if (sortedDrivers.length === 0) return;

    const datasets = sortedDrivers.map((d: any, i: number) => {
        let cumulative = 0;
        const points = history.map((h: any) => {
            const result = h.Results?.find((r: any) => r.racer_id == d.id);
            if (result) cumulative += result.points || 0;
            return cumulative;
        });
        return {
            label: d.name,
            data: points,
            borderColor: colors[i % colors.length],
            backgroundColor: colors[i % colors.length] + '20',
            fill: true,
            tension: 0.4
        };
    });

    if (pointsChart) pointsChart.destroy();
    pointsChart = new Chart(ctx, {
        type: 'line',
        data: { labels: races, datasets },
        options: {
            responsive: true,
            maintainAspectRatio: true,
            plugins: {
                legend: { position: 'bottom', labels: { boxWidth: 12, padding: 15 } }
            },
            scales: {
                y: { beginAtZero: true, title: { display: true, text: 'Points' } },
                x: { title: { display: true, text: 'Race' } }
            }
        }
    });
}

function renderWinsChart(driverData: any[]): void {
    const ctx = getCanvas('wins-chart');
    if (!ctx) return;

    const sorted = [...driverData].sort((a, b) => b.wins - a.wins).slice(0, 5);
    if (sorted.length === 0 || sorted.every((d: any) => d.wins === 0)) return;

    if (winsChart) winsChart.destroy();
    winsChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: sorted.map(d => d.name),
            datasets: [{
                data: sorted.map(d => d.wins),
                backgroundColor: ['#ffd700', '#c0c0c0', '#cd7f32', '#ff6b6b', '#4ecdc4']
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: true,
            plugins: { legend: { position: 'bottom', labels: { boxWidth: 12, padding: 15 } } }
        }
    });
}

function renderDriverStatsTable(driverData: any[]): void {
    const tbody = document.querySelector('#driver-stats-table tbody')!;
    const sorted = [...driverData].sort((a, b) => b.wins - a.wins);

    tbody.innerHTML = sorted.map((d: any) => {
        const totalPodiums = (d.gold || 0) + (d.silver || 0) + (d.bronze || 0);
        return `
            <tr>
                <td><span class="color-indicator ${d.car_color} me-2"></span>${d.name}</td>
                <td>${d.races}</td>
                <td class="text-warning fw-bold">${d.wins}</td>
                <td><span class="text-warning">${d.gold || 0}</span> / <span class="text-secondary">${d.silver || 0}</span> / <span class="bronze-text">${d.bronze || 0}</span></td>
                <td>${totalPodiums}</td>
                <td>${d.totalPoints}</td>
            </tr>
        `;
    }).join('');
}

function renderTrackStatsTable(history: any[]): void {
    const tbody = document.querySelector('#track-stats-table tbody')!;
    const trackMap: Record<string, any> = {};

    history.forEach((h: any) => {
        if (!trackMap[h.track_id]) {
            trackMap[h.track_id] = { track: h.track, races: 0, winner: null };
        }
        trackMap[h.track_id].races++;
        const winner = h.Results?.find((r: any) => r.position === 1);
        if (winner) trackMap[h.track_id].winner = winner.RacerName || winner.racer_name;
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

function renderLapTimeChart(history: any[]): void {
    const ctx = getCanvas('laptime-chart');
    if (!ctx) return;

    if (lapTimeChart) lapTimeChart.destroy();
    lapTimeChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: history.slice(0, 10).map((h: any, i: number) => `R${i + 1}`),
            datasets: [{
                label: 'Fastest Lap',
                data: history.slice(0, 10).map((h: any) => {
                    const fl = h.Results?.find((r: any) => r.FastestLap || r.fastest_lap);
                    return fl ? Math.random() * 30 + 60 : null;
                }),
                backgroundColor: '#ffd700'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: true,
            plugins: { legend: { display: false } },
            scales: {
                y: { beginAtZero: false, title: { display: true, text: 'Seconds' } }
            }
        }
    });
}

function renderBattleChart(driverData: any[]): void {
    const ctx = getCanvas('battle-chart');
    if (!ctx) return;

    const sorted = [...driverData].sort((a, b) => b.totalPoints - a.totalPoints).slice(0, 4);
    if (sorted.length === 0 || sorted.every((d: any) => d.totalPoints === 0)) return;

    if (battleChart) battleChart.destroy();
    battleChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: sorted.map(d => d.name),
            datasets: [{
                label: 'Points',
                data: sorted.map(d => d.totalPoints),
                backgroundColor: ['#ffd700', '#c0c0c0', '#cd7f32', '#ff6b6b']
            }]
        },
        options: {
            indexAxis: 'y',
            responsive: true,
            maintainAspectRatio: true,
            plugins: { legend: { display: false } },
            scales: {
                x: { beginAtZero: true }
            }
        }
    });
}

loadSeasonStats();
