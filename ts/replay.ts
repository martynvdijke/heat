import './theme';
interface ReplayRace {
    id: number; name: string; race_date: string; track: string; country: string;
    total_laps: number; results: ReplayResult[];
}

interface ReplayResult {
    racer_id: number; racer_name: string; position: number; points: number;
    fastest_lap: boolean; finished: boolean;
}

let replayRaces: ReplayRace[] = [];
let currentReplayRace: ReplayRace | null = null;
let currentReplayLap = 1;
let replayLaps: Map<number, any[]> = new Map();
let replayPlaying = false;
let replayInterval: ReturnType<typeof setInterval> | null = null;

async function loadReplayRaces(): Promise<void> {
    const res = await fetch('/api/race-history');
    replayRaces = await res.json();
    const select = document.getElementById('replay-race-select') as HTMLSelectElement;
    replayRaces.forEach(r => {
        const opt = document.createElement('option');
        opt.value = r.id.toString();
        opt.textContent = `${r.race_date || '?'} - ${r.track || r.name}`;
        select.appendChild(opt);
    });
}

async function selectReplayRace(): Promise<void> {
    const select = document.getElementById('replay-race-select') as HTMLSelectElement;
    const id = parseInt(select.value);
    if (!id) return;

    currentReplayRace = replayRaces.find(r => r.id === id) || null;
    if (!currentReplayRace) return;

    // Load lap records
    const lapRes = await fetch(`/api/lap-records?race_id=${id}`);
    const records = await lapRes.json();

    replayLaps.clear();
    records.forEach((r: any) => {
        if (!replayLaps.has(r.lap_number)) {
            replayLaps.set(r.lap_number, []);
        }
        replayLaps.get(r.lap_number)!.push(r);
    });

    currentReplayLap = 1;
    renderTimeline();
    renderReplayLap(currentReplayLap);
    renderReplayStats();
}

function renderTimeline(): void {
    const totalLaps = currentReplayRace?.total_laps || replayLaps.size || 10;
    const timeline = document.getElementById('replay-timeline')!;
    timeline.innerHTML = '';
    for (let i = 1; i <= totalLaps; i++) {
        const div = document.createElement('div');
        div.className = `replay-lap ${i === currentReplayLap ? 'active' : ''}`;
        div.textContent = i.toString();
        div.onclick = () => { currentReplayLap = i; renderReplayLap(i); renderTimeline(); };
        timeline.appendChild(div);
    }
    document.getElementById('replay-lap-display')!.textContent = `Lap ${currentReplayLap} / ${totalLaps}`;
}

function renderReplayLap(lap: number): void {
    const container = document.getElementById('replay-positions')!;
    const lapData = replayLaps.get(lap);
    if (!lapData || !currentReplayRace) {
        container.innerHTML = '<p class="text-center opacity-50">No data for this lap</p>';
        return;
    }

    const sorted = [...lapData].sort((a: any, b: any) => a.position - b.position);
    container.innerHTML = sorted.map((r: any) => {
        const name = currentReplayRace!.results.find((rr: any) => rr.racer_id === r.racer_id)?.racer_name || `#${r.racer_id}`;
        const gearStr = r.gear_used ? `<span class="replay-gear ms-2">G${r.gear_used}</span>` : '';
        const heatStr = r.heat_generated ? `<span class="replay-heat ms-2"><i class="fa-solid fa-fire"></i> ${r.heat_generated}</span>` : '';
        const turboStr = r.turbo_used ? `<span class="replay-turbo ms-2"><i class="fa-solid fa-bolt"></i></span>` : '';
        return `<div class="replay-row">
            <div class="replay-pos">P${r.position}</div>
            <div class="replay-name">${name}</div>
            ${gearStr}${heatStr}${turboStr}
        </div>`;
    }).join('');

    // Load events for this lap
    fetch(`/api/race-events?race_id=${currentReplayRace.id}`)
        .then(r => r.json())
        .then(events => {
            const lapEvents = events.filter((e: any) => e.lap === lap);
            const el = document.getElementById('replay-events')!;
            const noEvents = el.querySelector('.opacity-50');
            if (noEvents) noEvents.remove();
            if (lapEvents.length === 0) {
                if (!el.querySelector('.opacity-50')) {
                    // keep previous
                }
                return;
            }
            lapEvents.forEach((e: any) => {
                const existing = Array.from(el.children).find(c => c.textContent?.includes(`Lap ${e.lap} ${e.event_type}`));
                if (!existing) {
                    const div = document.createElement('div');
                    div.className = 'py-1 border-bottom border-secondary border-opacity-25 small';
                    div.innerHTML = `<span class="opacity-75">Lap ${e.lap}</span> ${e.event_type} - ${e.note || ''}`;
                    el.prepend(div);
                }
            });
        });
}

function renderReplayStats(): void {
    if (!currentReplayRace) return;
    const results = currentReplayRace.results || [];
    if (results.length === 0) return;

    const avgPos = results.reduce((s, r) => s + r.position, 0) / results.length;
    document.getElementById('replay-avg-pos')!.textContent = avgPos.toFixed(1);

    const mostOvertakes = results.sort((a, b) => b.points - a.points)[0]?.racer_name || '-';
    document.getElementById('replay-most-overtakes')!.textContent = mostOvertakes;

    // Count turbo and heat from lap records
    let totalTurbo = 0;
    let totalHeat = 0;
    for (const [, laps] of replayLaps) {
        laps.forEach((r: any) => {
            if (r.turbo_used) totalTurbo++;
            totalHeat += r.heat_generated || 0;
        });
    }
    document.getElementById('replay-total-turbo')!.textContent = totalTurbo.toString();
    document.getElementById('replay-total-heat')!.textContent = totalHeat.toString();
}

function replayNextLap(): void {
    const totalLaps = currentReplayRace?.total_laps || replayLaps.size || 10;
    if (currentReplayLap < totalLaps) {
        currentReplayLap++;
        renderReplayLap(currentReplayLap);
        renderTimeline();
    }
}

function replayPrevLap(): void {
    if (currentReplayLap > 1) {
        currentReplayLap--;
        renderReplayLap(currentReplayLap);
        renderTimeline();
    }
}

function toggleReplayPlay(): void {
    const btn = document.getElementById('replay-play-btn')!;
    if (replayPlaying) {
        replayPlaying = false;
        clearInterval(replayInterval!);
        btn.innerHTML = '<i class="fa-solid fa-play"></i>';
        btn.className = 'btn btn-success';
    } else {
        replayPlaying = true;
        btn.innerHTML = '<i class="fa-solid fa-pause"></i>';
        btn.className = 'btn btn-warning';
        replayInterval = setInterval(() => {
            replayNextLap();
            const totalLaps = currentReplayRace?.total_laps || replayLaps.size || 10;
            if (currentReplayLap >= totalLaps) {
                toggleReplayPlay();
            }
        }, 1000);
    }
}

function replayReset(): void {
    currentReplayLap = 1;
    if (replayPlaying) toggleReplayPlay();
    renderReplayLap(1);
    renderTimeline();
}

// Event listener for race select
document.getElementById('replay-race-select')?.addEventListener('change', selectReplayRace);

loadReplayRaces();
