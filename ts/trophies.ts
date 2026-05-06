interface Achievement {
    id: string;
    name: string;
    desc: string;
    icon: string;
    tier: string;
    req: number;
}

const raceWinAchievements: Achievement[] = [
    { id: 'win1', name: 'First Victory', desc: 'Win your first race', icon: 'fa-flag', tier: 'bronze', req: 1 },
    { id: 'win3', name: 'Hat Trick', desc: 'Win 3 races', icon: 'fa-hat-wizard', tier: 'silver', req: 3 },
    { id: 'win5', name: 'Pole Master', desc: 'Win 5 races', icon: 'fa-star', tier: 'silver', req: 5 },
    { id: 'win10', name: 'Dominant', desc: 'Win 10 races', icon: 'fa-crown', tier: 'gold', req: 10 },
    { id: 'win25', name: 'Legend', desc: 'Win 25 races', icon: 'fa-trophy', tier: 'platinum', req: 25 },
    { id: 'win50', name: 'GOAT', desc: 'Win 50 races', icon: 'fa-gem', tier: 'special', req: 50 }
];

const podiumAchievements: Achievement[] = [
    { id: 'podium1', name: 'First Podium', desc: 'Finish on podium', icon: 'fa-person-rowing', tier: 'bronze', req: 1 },
    { id: 'podium5', name: 'Consistent', desc: '5 podiums', icon: 'fa-medal', tier: 'silver', req: 5 },
    { id: 'podium10', name: 'Elite', desc: '10 podiums', icon: 'fa-award', tier: 'gold', req: 10 },
    { id: 'podium25', name: 'Champ', desc: '25 podiums', icon: 'fa-crown', tier: 'platinum', req: 25 }
];

const speedAchievements: Achievement[] = [
    { id: 'fl1', name: 'Speed Demon', desc: 'Set 1 fastest lap', icon: 'fa-bolt', tier: 'bronze', req: 1 },
    { id: 'fl3', name: 'Lightning', desc: 'Set 3 fastest laps', icon: 'fa-bolt', tier: 'silver', req: 3 },
    { id: 'fl5', name: 'Rocket', desc: 'Set 5 fastest laps', icon: 'fa-rocket', tier: 'gold', req: 5 },
    { id: 'fl10', name: 'Supersonic', desc: 'Set 10 fastest laps', icon: 'fa-meteor', tier: 'platinum', req: 10 }
];

async function loadDrivers(): Promise<void> {
    const res = await fetch('/api/racers');
    const racers = await res.json();

    const select = document.getElementById('driver-select') as HTMLSelectElement;
    racers.forEach((r: { id: number; name: string }) => {
        const opt = document.createElement('option');
        opt.value = String(r.id);
        opt.textContent = r.name;
        select.appendChild(opt);
    });

    if (racers.length > 0) {
        select.value = String(racers[0].id);
        loadAchievements(racers[0].id);
    }
}

async function loadAchievements(driverId: number): Promise<void> {
    try {
        const statsRes = await fetch(`/api/racer-stats?id=${driverId}`);
        const statsData = await statsRes.json();
        const stats = statsData.stats || {};

        const wins = stats.gold || stats.wins || 0;
        const podiums = (stats.gold || 0) + (stats.silver || 0) + (stats.bronze || 0);
        const fastestLaps = stats.fastest_laps || 0;

        let badges = 0;
        if (wins >= 1) badges++;
        if (wins >= 3) badges++;
        if (wins >= 5) badges++;
        if (podiums >= 1) badges++;
        if (podiums >= 5) badges++;
        if (fastestLaps >= 1) badges++;
        if (fastestLaps >= 3) badges++;

        document.getElementById('total-trophies')!.textContent = String(wins);
        document.getElementById('total-badges')!.textContent = String(badges);
        document.getElementById('completion-rate')!.textContent = Math.round((badges / 14) * 100) + '%';

        renderTrophyGrid('race-wins-grid', raceWinAchievements, wins);
        renderTrophyGrid('podium-grid', podiumAchievements, podiums);
        renderTrophyGrid('speed-grid', speedAchievements, fastestLaps);
        renderSpecialAchievements(wins, podiums, fastestLaps);

    } catch (err) {
        console.error('Failed to load achievements:', err);
    }
}

function renderTrophyGrid(gridId: string, achievements: Achievement[], current: number): void {
    const grid = document.getElementById(gridId)!;
    grid.innerHTML = achievements.map(a => {
        const achieved = current >= a.req;
        const progress = Math.min((current / a.req) * 100, 100);

        return `
            <div class="trophy-card ${achieved ? '' : 'locked'}">
                <div class="achievement-badge ${a.tier}">
                    <i class="fa-solid ${a.icon}"></i>
                </div>
                <h5 class="mb-1">${a.name}</h5>
                <p class="small text-muted mb-2">${a.desc}</p>
                ${!achieved ? `
                    <div class="progress-bar-mini">
                        <div class="progress-fill" style="width: ${progress}%"></div>
                    </div>
                    <small class="text-muted">${current}/${a.req}</small>
                ` : '<i class="fa-solid fa-check-circle text-success"></i>'}
            </div>
        `;
    }).join('');
}

function renderSpecialAchievements(wins: number, podiums: number, fastestLaps: number): void {
    const special: Array<{ id: string; name: string; desc: string; icon: string; tier: string; achieved: boolean }> = [
        { id: 'perfect_season', name: 'Perfect Season', desc: 'Win every race', icon: 'fa-crown', tier: 'special', achieved: wins >= 5 },
        { id: 'track_master', name: 'Track Master', desc: 'Win at 5 tracks', icon: 'fa-map', tier: 'platinum', achieved: wins >= 5 },
        { id: 'comeback_kid', name: 'Comeback Kid', desc: 'Win from last', icon: 'fa-arrow-up', tier: 'gold', achieved: false },
        { id: 'hat_trick', name: 'Hat Trick', desc: 'Win 3 in a row', icon: 'fa-gem', tier: 'gold', achieved: wins >= 3 },
        { id: 'points_leader', name: 'Points Leader', desc: 'Lead championship', icon: 'fa-chart-line', tier: 'silver', achieved: wins > 0 },
        { id: 'most_laps', name: 'Most Laps Led', desc: 'Lead 50 laps', icon: 'fa-road', tier: 'silver', achieved: fastestLaps >= 3 }
    ];

    document.getElementById('special-grid')!.innerHTML = special.map(a => `
        <div class="trophy-card ${a.achieved ? '' : 'locked'}">
            <div class="achievement-badge ${a.tier}">
                <i class="fa-solid ${a.icon}"></i>
            </div>
            <h5 class="mb-1">${a.name}</h5>
            <p class="small text-muted">${a.desc}</p>
            ${a.achieved ? '<i class="fa-solid fa-check-circle text-success"></i>' : ''}
        </div>
    `).join('');
}

document.getElementById('driver-select')!.addEventListener('change', (e: Event) => {
    const target = e.target as HTMLSelectElement;
    if (target.value) loadAchievements(parseInt(target.value));
});

loadDrivers();

