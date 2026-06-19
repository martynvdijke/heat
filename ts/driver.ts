import './theme';
export {}; // make this a module

function escapeHtml(text: string | null | undefined): string {
    const div = document.createElement('div');
    div.textContent = text ?? '';
    return div.innerHTML;
}

async function init(): Promise<void> {
    const params = new URLSearchParams(window.location.search);
    const token = params.get('token');

    if (!token) {
        document.getElementById('driver-content')!.innerHTML = `
            <div class="stats-card mx-auto error-card">
                <div class="p-5">
                    <i class="fa-solid fa-triangle-exclamation fa-3x text-danger mb-3"></i>
                    <h5>Invalid Link</h5>
                    <p class="text-muted mb-0">No access token provided. Please use the link from your email.</p>
                </div>
            </div>
        `;
        return;
    }

    try {
        const res = await fetch('/api/shared/driver-stats?token=' + encodeURIComponent(token));
        if (!res.ok) {
            throw new Error('Invalid or expired link');
        }
        const data = await res.json();
        const r = data.racer;
        const s = data.stats;

        document.getElementById('driver-content')!.innerHTML = `
            <div class="stats-card mx-auto">
                <div class="driver-header">
                    <img src="${escapeHtml(r.profile_picture || '/static/images/helmet.svg')}" alt="${escapeHtml(r.name)}" onerror="this.src='/static/images/helmet.svg'">
                    <h2>${escapeHtml(r.name)}</h2>
                    <p class="mb-0 opacity-75"><span class="badge-stat bg-danger me-1">${escapeHtml(r.car_name || 'No Car')}</span></p>
                </div>
                <div class="p-3">
                    <h6 class="text-muted fw-bold px-3 pt-2 mb-0"><i class="fa-solid fa-trophy me-1 text-danger"></i>Career Stats</h6>
                    <div class="stat-item">
                        <span class="stat-label"><i class="fa-solid fa-flag-checkered me-1 text-muted"></i>Races</span>
                        <span class="stat-value">${s.races || 0}</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label"><i class="fa-solid fa-trophy me-1" style="color:#d4a017"></i>Wins</span>
                        <span class="stat-value gold">${s.wins || 0}</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label"><i class="fa-solid fa-medal me-1" style="color:#d4a017"></i>Gold</span>
                        <span class="stat-value gold">${s.gold || 0}</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label"><i class="fa-solid fa-medal me-1" style="color:#8a8a8a"></i>Silver</span>
                        <span class="stat-value silver">${s.silver || 0}</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label"><i class="fa-solid fa-medal me-1" style="color:#cd7f32"></i>Bronze</span>
                        <span class="stat-value bronze">${s.bronze || 0}</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label"><i class="fa-solid fa-bolt me-1 text-warning"></i>Fastest Laps</span>
                        <span class="stat-value">${s.fastest_laps || 0}</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label"><i class="fa-solid fa-star me-1" style="color:#d40000"></i>Total Points</span>
                        <span class="stat-value">${s.points || 0}</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label"><i class="fa-solid fa-circle-xmark me-1 text-danger"></i>DNF</span>
                        <span class="stat-value">${s.dnf || 0}</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label"><i class="fa-solid fa-circle-minus me-1 text-muted"></i>DNS</span>
                        <span class="stat-value">${s.dns || 0}</span>
                    </div>
                </div>
                <div class="version-footer">HEAT Racing Companion &mdash; {{VERSION}}</div>
            </div>
        `;
    } catch {
        document.getElementById('driver-content')!.innerHTML = `
            <div class="stats-card mx-auto error-card">
                <div class="p-5">
                    <i class="fa-solid fa-triangle-exclamation fa-3x text-danger mb-3"></i>
                    <h5>Link Not Found</h5>
                    <p class="text-muted mb-0">This share link is invalid or has expired. Please contact your race administrator.</p>
                </div>
            </div>
        `;
    }
}

init();
