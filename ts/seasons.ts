interface Season {
    id: number;
    name: string;
    start_date: string;
    end_date: string;
    status: string;
    created_at: string;
}

interface RoundSnapshot {
    id: number;
    season_id: number;
    race_name: string;
    race_date: string;
    round: number;
    created_at: string;
    scores?: RoundScore[];
}

interface RoundScore {
    id: number;
    snapshot_id: number;
    racer_id: number;
    racer_name: string;
    points: number;
    position: number;
}

async function loadSeasonsPage(): Promise<void> {
    try {
        const seasonsRes = await fetch('/api/seasons');
        const seasons: Season[] = await seasonsRes.json();
        const container = document.getElementById('seasons-container')!;

        if (!Array.isArray(seasons) || seasons.length === 0) {
            container.innerHTML = `
                <div class="col-12 text-center text-muted py-5">
                    <i class="fa-solid fa-calendar-xmark fa-3x mb-3"></i>
                    <p>No seasons configured yet.</p>
                </div>`;
            return;
        }

        const sorted = [...seasons].sort((a, b) => {
            if (a.status === 'active') return -1;
            if (b.status === 'active') return 1;
            return b.id - a.id;
        });

        let html = '';
        for (const season of sorted) {
            html += await renderSeasonCard(season);
        }
        container.innerHTML = html;
    } catch (err) {
        console.error('Failed to load seasons:', err);
        document.getElementById('seasons-container')!.innerHTML = `
            <div class="col-12 text-center text-muted py-5">
                <i class="fa-solid fa-triangle-exclamation fa-3x mb-3"></i>
                <p>Failed to load seasons.</p>
            </div>`;
    }
}

async function renderSeasonCard(season: Season): Promise<string> {
    const isActive = season.status === 'active';
    const statusBadge = isActive
        ? '<span class="badge bg-success ms-2">Active</span>'
        : '<span class="badge bg-secondary ms-2">Archived</span>';
    const dateRange = season.end_date
        ? `${season.start_date} — ${season.end_date}`
        : `${season.start_date} — Present`;
    const collapseId = `season-${season.id}`;
    const expandedClass = isActive ? 'show' : '';

    let rounds: RoundSnapshot[] = [];
    try {
        const roundsRes = await fetch(`/api/rounds?season_id=${season.id}`);
        rounds = await roundsRes.json();
    } catch {
        rounds = [];
    }

    const roundsHtml = Array.isArray(rounds) && rounds.length > 0
        ? await renderRounds(rounds)
        : `<p class="text-muted mb-0">No rounds recorded for this season.</p>`;

    return `
        <div class="col-12">
            <div class="card border-0 shadow-sm ${isActive ? 'border-start border-success border-4' : ''}">
                <div class="card-header bg-transparent border-0 p-0">
                    <button class="btn btn-link w-100 text-start text-decoration-none d-flex justify-content-between align-items-center p-3" 
                            type="button" data-bs-toggle="collapse" data-bs-target="#${collapseId}">
                        <div>
                            <h5 class="mb-0">
                                <i class="fa-solid ${isActive ? 'fa-fire text-danger' : 'fa-archive'} me-2"></i>
                                ${season.name}
                                ${statusBadge}
                            </h5>
                            <small class="text-muted">${dateRange} &middot; ${Array.isArray(rounds) ? rounds.length : 0} rounds</small>
                        </div>
                        <i class="fa-solid fa-chevron-down collapse-icon"></i>
                    </button>
                </div>
                <div class="collapse ${expandedClass}" id="${collapseId}">
                    <div class="card-body pt-0">
                        ${roundsHtml}
                    </div>
                </div>
            </div>
        </div>`;
}

async function renderRounds(rounds: RoundSnapshot[]): Promise<string> {
    const roundsWithScores = await Promise.all(
        rounds.map(async (r) => {
            try {
                const res = await fetch(`/api/rounds?id=${r.id}`);
                return await res.json() as RoundSnapshot;
            } catch {
                return r;
            }
        })
    );

    return `
        <div class="table-responsive">
            <table class="table table-sm">
                <thead>
                    <tr>
                        <th>R#</th>
                        <th>Race</th>
                        <th>Date</th>
                        <th>Top Drivers</th>
                    </tr>
                </thead>
                <tbody>
                    ${roundsWithScores.map(r => {
                        const scores = (r.scores || []).slice(0, 5);
                        return `
                            <tr>
                                <td class="fw-bold">${r.round}</td>
                                <td>${r.race_name}</td>
                                <td class="text-nowrap">${r.race_date}</td>
                                <td>
                                    <div class="d-flex gap-3 flex-wrap">
                                        ${scores.map((s, i) => `
                                            <span class="${i === 0 ? 'text-warning fw-bold' : i === 1 ? 'text-secondary' : i === 2 ? 'bronze-text' : ''}">
                                                ${i === 0 ? '<i class="fa-solid fa-trophy me-1"></i>' : ''}
                                                ${s.racer_name} <small>(${s.points})</small>
                                            </span>
                                        `).join('')}
                                    </div>
                                </td>
                            </tr>
                        `;
                    }).join('')}
                </tbody>
            </table>
        </div>`;
}

loadSeasonsPage();
