import './theme';
export {}; // make this a module

interface MeRacer {
    id: number; name: string; profile_picture?: string; car_color?: string;
    car_name?: string; points?: number;
}
interface MeStats {
    races: number; wins: number; gold: number; silver: number; bronze: number;
    fastest_laps: number; points: number; dnf: number; dns: number; spins: number; overheated: number;
}
interface MeRecent {
    race_id: number; name: string; race_date: string; track: string; country: string;
    position: number; points: number; fastest_lap: boolean; race_type: string;
}
interface MeStanding { rank: number; points: number; races: number; wins: number; }
interface MeResponse {
    racer: MeRacer; stats: MeStats; recent: MeRecent[];
    season?: { id: number; name: string }; standing?: MeStanding;
}
interface UpgradeCard {
    id: number; name: string; description: string; card_type: string; cost: number; effects?: string;
}
interface PlayerUpgrade {
    id: number; racer_id: number; upgrade_id: number; season_id: number;
    equipped: boolean; round_bought: number; upgrade?: UpgradeCard;
}
interface RacerOption { id: number; name: string; }
interface HeadToHead {
    racer1: string; racer2: string; races: number;
    racer1_wins: number; racer2_wins: number;
    racer1_avg_position: number; racer2_avg_position: number;
}
interface PointsProgression { race_id: number; race_name: string; race_date: string; points: number; }

let meRacer: MeRacer | null = null;
let meSeasonID = 0;

function escapeHtml(text: string | number | null | undefined): string {
    const div = document.createElement('div');
    div.textContent = text == null ? '' : String(text);
    return div.innerHTML;
}

function content(): HTMLElement {
    return document.getElementById('me-content')!;
}

async function init(): Promise<void> {
    try {
        const res = await fetch('/api/me');
        if (res.status === 401) { renderLogin(); return; }
        if (!res.ok) throw new Error('failed');
        const data: MeResponse = await res.json();
        meRacer = data.racer;
        meSeasonID = data.season?.id || 0;
        renderDashboard(data);
        void loadProgression();
        void loadUpgrades();
        void loadHeadToHeadOptions();
    } catch {
        renderLogin();
    }
}

function renderLogin(): void {
    content().innerHTML = `
        <div class="profile-card mx-auto" style="max-width:440px">
            <div class="driver-header"><img src="/static/images/helmet.svg" alt="HEAT"><div><h4 class="mb-0">Sign in</h4><small class="opacity-75">Use the email on file for your racer</small></div></div>
            <div class="p-4">
                <p class="text-muted">Enter your email and we'll send you a one-time sign-in link.</p>
                <form id="link-form">
                    <input type="email" class="form-control mb-3" id="link-email" placeholder="you@example.com" required>
                    <button class="btn btn-danger w-100" type="submit">Email me a link</button>
                </form>
                <div id="link-msg" class="mt-3"></div>
            </div>
            <div class="version-footer">HEAT Racing Companion &mdash; {{VERSION}}</div>
        </div>`;
    document.getElementById('link-form')!.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = (document.getElementById('link-email') as HTMLInputElement).value;
        await fetch('/api/me/request-link', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email }),
        });
        document.getElementById('link-msg')!.innerHTML =
            '<div class="alert alert-success mb-0">If that email is on file, a sign-in link is on its way.</div>';
    });
}

function statRow(label: string, value: string | number, cls = ''): string {
    return `<div class="d-flex justify-content-between border-bottom py-1"><span class="text-muted">${escapeHtml(label)}</span><span class="fw-bold ${cls}">${escapeHtml(value)}</span></div>`;
}

function renderDashboard(data: MeResponse): void {
    const r = data.racer;
    const s = data.stats;
    const pic = r.profile_picture || '/static/images/helmet.svg';
    const color = r.car_color || '#d40000';

    let seasonHTML = '<p class="text-muted mb-0">No active season.</p>';
    if (data.season && data.standing) {
        seasonHTML = `
            <div class="fw-bold mb-1">${escapeHtml(data.season.name)}</div>
            <div class="d-flex gap-3">
                <div><div class="section-title">Position</div><div class="fs-4 fw-bold">#${data.standing.rank}</div></div>
                <div><div class="section-title">Points</div><div class="fs-4 fw-bold">${data.standing.points}</div></div>
                <div><div class="section-title">Wins</div><div class="fs-4 fw-bold">${data.standing.wins}</div></div>
                <div><div class="section-title">Races</div><div class="fs-4 fw-bold">${data.standing.races}</div></div>
            </div>`;
    }

    const recentHTML = data.recent.length
        ? data.recent.map(x => `
            <div class="d-flex justify-content-between border-bottom py-1">
                <span>${escapeHtml(x.name)} <small class="text-muted">${escapeHtml(x.track || '')}</small></span>
                <span class="fw-bold">P${x.position}${x.fastest_lap ? ' ⚡' : ''} <small class="text-muted">+${x.points}</small></span>
            </div>`).join('')
        : '<p class="text-muted mb-0">No results yet.</p>';

    content().innerHTML = `
        <div class="profile-card mb-3">
            <div class="driver-header">
                <img src="${escapeHtml(pic)}" alt="${escapeHtml(r.name)}" onerror="this.src='/static/images/helmet.svg'">
                <div class="flex-grow-1">
                    <h4 class="mb-0">${escapeHtml(r.name)}</h4>
                    <small class="opacity-75"><span class="colored-dot me-1" style="background:${escapeHtml(color)}"></span>${escapeHtml(r.car_name || 'No car')}</small>
                </div>
                <button class="btn btn-outline-light btn-sm" id="logout-btn"><i class="fa-solid fa-right-from-bracket"></i> Sign out</button>
            </div>
        </div>
        <div class="row g-3">
            <div class="col-md-6">
                <div class="profile-card p-3">
                    <div class="section-title mb-2"><i class="fa-solid fa-trophy text-danger me-1"></i>Career</div>
                    ${statRow('Races', s.races)}
                    ${statRow('Wins', s.wins, 'gold')}
                    ${statRow('Gold / Silver / Bronze', `${s.gold} / ${s.silver} / ${s.bronze}`)}
                    ${statRow('Fastest laps', s.fastest_laps)}
                    ${statRow('Total points', s.points)}
                    ${statRow('DNF / DNS', `${s.dnf} / ${s.dns}`)}
                </div>
            </div>
            <div class="col-md-6">
                <div class="profile-card p-3 mb-3">
                    <div class="section-title mb-2"><i class="fa-solid fa-flag-checkered text-danger me-1"></i>Season</div>
                    ${seasonHTML}
                </div>
                <div class="profile-card p-3">
                    <div class="section-title mb-2"><i class="fa-solid fa-clock-rotate-left text-danger me-1"></i>Recent form</div>
                    ${recentHTML}
                </div>
            </div>
            <div class="col-md-6">
                <div class="profile-card p-3">
                    <div class="section-title mb-2"><i class="fa-solid fa-chart-line text-danger me-1"></i>Points progression</div>
                    <div id="progression"><span class="text-muted">Loading…</span></div>
                </div>
            </div>
            <div class="col-md-6">
                <div class="profile-card p-3">
                    <div class="section-title mb-2"><i class="fa-solid fa-people-arrows text-danger me-1"></i>Head to head</div>
                    <select class="form-select form-select-sm mb-2" id="h2h-opponent"><option value="">Choose an opponent…</option></select>
                    <div id="h2h-result"><span class="text-muted">Pick someone to compare.</span></div>
                </div>
            </div>
            <div class="col-12">
                <div class="profile-card p-3">
                    <div class="section-title mb-2"><i class="fa-solid fa-wrench text-danger me-1"></i>My upgrades</div>
                    <div class="row g-3">
                        <div class="col-md-7"><h6 class="text-muted">Owned</h6><div id="owned-upgrades"><span class="text-muted">Loading…</span></div></div>
                        <div class="col-md-5"><h6 class="text-muted">Available to buy</h6><div id="available-upgrades"><span class="text-muted">Loading…</span></div></div>
                    </div>
                </div>
            </div>
        </div>
        <div class="version-footer">HEAT Racing Companion &mdash; {{VERSION}}</div>`;

    document.getElementById('logout-btn')!.addEventListener('click', async () => {
        await fetch('/api/me/logout', { method: 'POST' });
        window.location.href = '/me.html';
    });
}

async function loadProgression(): Promise<void> {
    const el = document.getElementById('progression');
    if (!el || !meRacer) return;
    try {
        const res = await fetch(`/api/stats/points-progression?racer_id=${meRacer.id}`);
        const points: PointsProgression[] = res.ok ? await res.json() : [];
        if (!points.length) { el.innerHTML = '<span class="text-muted">No data yet.</span>'; return; }
        const max = Math.max(...points.map(p => p.points), 1);
        el.innerHTML = points.map(p => `
            <div class="d-flex align-items-center gap-2 mb-1">
                <small class="text-muted" style="width:42%">${escapeHtml(p.race_name)}</small>
                <div class="flex-grow-1" style="background:#eee;border-radius:4px"><div style="height:12px;border-radius:4px;background:#d40000;width:${Math.round((p.points / max) * 100)}%"></div></div>
                <small class="fw-bold">${p.points}</small>
            </div>`).join('');
    } catch { el.innerHTML = '<span class="text-muted">Could not load.</span>'; }
}

async function loadHeadToHeadOptions(): Promise<void> {
    const select = document.getElementById('h2h-opponent') as HTMLSelectElement | null;
    if (!select || !meRacer) return;
    try {
        const res = await fetch('/api/racers');
        const racers: RacerOption[] = await res.json();
        racers.filter(r => r.id !== meRacer!.id).forEach(r => {
            const opt = document.createElement('option');
            opt.value = String(r.id);
            opt.textContent = r.name;
            select.appendChild(opt);
        });
        select.addEventListener('change', () => void loadHeadToHead(select.value));
    } catch { /* ignore */ }
}

async function loadHeadToHead(opponentId: string): Promise<void> {
    const el = document.getElementById('h2h-result');
    if (!el || !meRacer || !opponentId) { if (el) el.innerHTML = '<span class="text-muted">Pick someone to compare.</span>'; return; }
    try {
        const res = await fetch(`/api/stats/head-to-head?racer1=${meRacer.id}&racer2=${encodeURIComponent(opponentId)}`);
        if (!res.ok) { el.innerHTML = '<span class="text-muted">No shared races.</span>'; return; }
        const h: HeadToHead = await res.json();
        el.innerHTML = `
            <div class="d-flex justify-content-between"><span>${escapeHtml(h.racer1)}</span><span class="fw-bold">${h.racer1_wins} wins</span></div>
            <div class="d-flex justify-content-between"><span>${escapeHtml(h.racer2)}</span><span class="fw-bold">${h.racer2_wins} wins</span></div>
            <div class="text-muted small mt-1">${h.races} shared races · avg finish ${h.racer1_avg_position.toFixed(1)} vs ${h.racer2_avg_position.toFixed(1)}</div>`;
    } catch { el.innerHTML = '<span class="text-muted">Could not load.</span>'; }
}

async function loadUpgrades(): Promise<void> {
    const ownedEl = document.getElementById('owned-upgrades');
    const availEl = document.getElementById('available-upgrades');
    if (!ownedEl || !availEl) return;
    try {
        const res = await fetch('/api/me/upgrades');
        const data: { owned: PlayerUpgrade[]; available: UpgradeCard[] } = await res.json();

        ownedEl.innerHTML = data.owned.length ? data.owned.map(u => `
            <div class="d-flex justify-content-between align-items-center border-bottom py-1">
                <span>${escapeHtml(u.upgrade?.name || 'Upgrade')} <small class="text-muted">· ${escapeHtml(u.upgrade?.cost ?? 0)}</small></span>
                <button class="btn btn-sm ${u.equipped ? 'btn-success' : 'btn-outline-secondary'} toggle-btn" data-id="${u.id}" data-equipped="${u.equipped}">
                    ${u.equipped ? 'Equipped' : 'Equip'}
                </button>
            </div>`).join('') : '<span class="text-muted">No upgrades yet.</span>';

        availEl.innerHTML = data.available.length ? data.available.map(u => `
            <div class="d-flex justify-content-between align-items-center border-bottom py-1">
                <span>${escapeHtml(u.name)} <small class="text-muted">· ${u.cost}</small></span>
                <button class="btn btn-sm btn-danger buy-btn" data-id="${u.id}">Buy</button>
            </div>`).join('') : '<span class="text-muted">Nothing available.</span>';

        ownedEl.querySelectorAll<HTMLButtonElement>('.toggle-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                await fetch('/api/me/upgrades/toggle', {
                    method: 'PUT', headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ id: Number(btn.dataset.id), equipped: btn.dataset.equipped !== 'true' }),
                });
                void loadUpgrades();
            });
        });
        availEl.querySelectorAll<HTMLButtonElement>('.buy-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                btn.disabled = true;
                await fetch('/api/me/upgrades/buy', {
                    method: 'POST', headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ upgrade_id: Number(btn.dataset.id), season_id: meSeasonID, round: 0 }),
                });
                void loadUpgrades();
            });
        });
    } catch {
        ownedEl.innerHTML = '<span class="text-muted">Could not load.</span>';
        availEl.innerHTML = '';
    }
}

init();
