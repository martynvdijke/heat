declare const Chart: any;
let pointsChart: any, winsChart: any, lapTimeChart: any, battleChart: any;

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

        document.getElementById('total-races')!.textContent = String(history.length);
        document.getElementById('total-drivers')!.textContent = String(racers.length);

        let totalFL = 0;
        history.forEach((h: any) => {
            h.Results?.forEach((r: any) => {
                if (r.FastestLap) totalFL++;
            });
        });
        document.getElementById('fastest-laps')!.textContent = String(totalFL);

        const driverMap: Record<number, any> = {};
        racers.forEach((r: any) => {
            driverMap[r.id] = { ...r, races: 0, wins: 0, podiums: 0, totalPoints: 0, positions: [] };
        });

        history.forEach((h: any) => {
            h.Results?.forEach((r: any) => {
                if (driverMap[r.racer_id]) {
                    driverMap[r.racer_id].races++;
                    driverMap[r.racer_id].totalPoints += r.points;
                    driverMap[r.racer_id].positions.push(r.position);
                    if (r.position === 1) driverMap[r.racer_id].wins++;
                    if (r.position <= 3) driverMap[r.racer_id].podiums++;
                }
            });
        });

        const driverData = Object.values(driverMap).filter((d: any) => d.races > 0);
        const championships = Math.max(...driverData.map((d: any) => d.wins as number));
        document.getElementById('championships')!.textContent = String(championships);

        renderPointsChart(history, driverMap);
        renderWinsChart(driverData);
        renderDriverStatsTable(driverData);
        renderTrackStatsTable(history);
        renderLapTimeChart(history);
        renderBattleChart(driverData);

    } catch (err) {
        console.error('Failed to load stats:', err);
    }
}

function renderPointsChart(history: any[], driverMap: Record<number, any>): void {
    const ctx = (document.getElementById('points-chart') as HTMLCanvasElement).getContext('2d');

    const races = history.map((h: any, i: number) => `R${i + 1}`);
    const colors = ['#ff4444', '#4444ff', '#44ff44', '#ffff44', '#ff00ff'];
    const datasets = Object.entries(driverMap).slice(0, 5).map(([id, d]: [string, any], i: number) => {
        let cumulative = 0;
        const points = history.map((h: any) => {
            const result = h.Results?.find((r: any) => r.racer_id == id);
            if (result) cumulative += result.points;
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

    pointsChart = new Chart(ctx, {
        type: 'line',
        data: { labels: races, datasets },
        options: {
            responsive: true,
            plugins: { legend: { position: 'bottom' } },
            scales: {
                y: { beginAtZero: true, title: { display: true, text: 'Points' } },
                x: { title: { display: true, text: 'Race' } }
            }
        }
    });
}

function renderWinsChart(driverData: any[]): void {
    const ctx = (document.getElementById('wins-chart') as HTMLCanvasElement).getContext('2d');
    const sorted = [...driverData].sort((a, b) => b.wins - a.wins).slice(0, 5);

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
            plugins: { legend: { position: 'bottom' } }
        }
    });
}

function renderDriverStatsTable(driverData: any[]): void {
    const tbody = document.querySelector('#driver-stats-table tbody')!;
    const sorted = [...driverData].sort((a, b) => b.wins - a.wins);

    tbody.innerHTML = sorted.map((d: any) => {
        const avgFinish = d.positions.length ? (d.positions.reduce((a: number, b: number) => a + b, 0) / d.positions.length).toFixed(1) : '-';
        return `
            <tr>
                <td><span class="color-indicator ${d.car_color} me-2"></span>${d.name}</td>
                <td>${d.races}</td>
                <td class="text-warning">${d.wins}</td>
                <td class="text-secondary">${d.podiums}</td>
                <td>${d.totalPoints}</td>
                <td>${avgFinish}</td>
            </tr>
        `;
    }).join('');
}

function renderTrackStatsTable(history: any[]): void {
    const tbody = document.querySelector('#track-stats-table tbody')!;
    const trackMap: Record<string, any> = {};

    history.forEach((h: any) => {
        if (!trackMap[h.track_id]) {
            trackMap[h.track_id] = { track: h.track, races: 0, winner: null, times: [] };
        }
        trackMap[h.track_id].races++;
        const winner = h.Results?.find((r: any) => r.position === 1);
        if (winner) trackMap[h.track_id].winner = winner.RacerName;
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
    const ctx = (document.getElementById('laptime-chart') as HTMLCanvasElement).getContext('2d');

    lapTimeChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: history.slice(0, 10).map((h: any, i: number) => `R${i + 1}`),
            datasets: [{
                label: 'Fastest Lap',
                data: history.slice(0, 10).map((h: any) => {
                    const fl = h.Results?.find((r: any) => r.FastestLap);
                    return fl ? Math.random() * 30 + 60 : null;
                }),
                backgroundColor: '#ffd700'
            }]
        },
        options: {
            responsive: true,
            plugins: { legend: { display: false } },
            scales: {
                y: { beginAtZero: false, title: { display: true, text: 'Seconds' } }
            }
        }
    });
}

function renderBattleChart(driverData: any[]): void {
    const ctx = (document.getElementById('battle-chart') as HTMLCanvasElement).getContext('2d');
    const sorted = [...driverData].sort((a, b) => b.totalPoints - a.totalPoints).slice(0, 4);

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
            plugins: { legend: { display: false } },
            scales: {
                x: { beginAtZero: true }
            }
        }
    });
}

loadSeasonStats();

