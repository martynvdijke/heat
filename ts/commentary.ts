import { escapeHtml } from './toast';

export interface CommentaryEntry {
    id: number;
    race_id: number;
    lap: number;
    racer_id?: number;
    racer_name?: string;
    message: string;
    template_key?: string;
    created_at?: string;
}

const FADE_AFTER_MS = 30000;
const FADE_TRANSITION_MS = 1000;
const POLL_INTERVAL_MS = 5000;
const MAX_ITEMS = 20;

/**
 * CommentaryTicker renders a live commentary feed.
 *
 * Entries arrive either over an existing WebSocket (type === 'commentary')
 * or via polling GET /api/commentary?race_id=0&since=<lastId> every 5s.
 * Entries older than 30s fade out (CSS transition) and are removed.
 * Hovering pauses polling so the operator can read the feed.
 */
export class CommentaryTicker {
    private container: HTMLElement;
    private lastId = 0;
    private pollTimer: number | null = null;
    private ws: WebSocket | null = null;
    private paused = false;

    constructor(container: HTMLElement) {
        this.container = container;
        this.container.setAttribute('aria-live', 'polite');
        this.container.addEventListener('mouseenter', () => { this.paused = true; });
        this.container.addEventListener('mouseleave', () => { this.paused = false; });
    }

    start(): void {
        this.poll();
        this.pollTimer = window.setInterval(() => this.poll(), POLL_INTERVAL_MS);
    }

    stop(): void {
        if (this.pollTimer !== null) {
            window.clearInterval(this.pollTimer);
            this.pollTimer = null;
        }
        if (this.ws) {
            this.ws.removeEventListener('message', this.onWsMessage);
            this.ws = null;
        }
    }

    connect(ws: WebSocket): void {
        this.ws = ws;
        ws.addEventListener('message', this.onWsMessage);
    }

    private onWsMessage = (event: MessageEvent): void => {
        let data: any;
        try {
            data = JSON.parse(event.data);
        } catch {
            return;
        }
        if (data && data.type === 'commentary') {
            this.append(data as CommentaryEntry);
        }
    };

    private async poll(): Promise<void> {
        if (this.paused) return;
        try {
            const res = await fetch(`/api/commentary?race_id=0&since=${this.lastId}`);
            if (!res.ok) return;
            const entries: CommentaryEntry[] = await res.json();
            // API returns newest-first; append in reverse to keep chronological order.
            for (let i = entries.length - 1; i >= 0; i--) {
                this.append(entries[i]);
            }
        } catch {
            // Ignore transient network errors; next poll will retry.
        }
    }

    append(entry: CommentaryEntry): void {
        if (entry.id > this.lastId) this.lastId = entry.id;
        if (this.container.querySelector(`[data-commentary-id="${entry.id}"]`)) return;

        const el = document.createElement('div');
        el.className = 'commentary-item';
        el.dataset.commentaryId = String(entry.id);
        const lap = entry.lap > 0 ? `<span class="commentary-lap">L${entry.lap}</span>` : '';
        const name = entry.racer_name ? `<span class="commentary-driver">${escapeHtml(entry.racer_name)}</span>` : '';
        el.innerHTML = `${lap}${name}<span class="commentary-message">${escapeHtml(entry.message)}</span>`;
        this.container.appendChild(el);

        window.setTimeout(() => {
            el.classList.add('commentary-fade');
            window.setTimeout(() => el.remove(), FADE_TRANSITION_MS);
        }, FADE_AFTER_MS);

        while (this.container.children.length > MAX_ITEMS) {
            this.container.firstElementChild?.remove();
        }
    }
}
