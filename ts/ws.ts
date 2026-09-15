// Shared WebSocket lifecycle helper for all Heat pages.
//
// Owns connect/reconnect with exponential backoff + full jitter, topic
// subscription, sequence-number gap detection, and resync requests. The server
// sends every broadcast as a sequenced envelope
// ({type, topic, seq, payload}); `hello`/`resync` carry a `snapshot`.
//
// Backoff: delay = random(0, min(cap, base * 2^attempt)); the attempt counter
// resets on `onopen`.

export interface HeatEnvelope<T = any> {
  type: string;
  topic?: string;
  seq?: number;
  payload?: T;
  snapshot?: HeatSnapshot;
  [key: string]: any;
}

export interface HeatSnapshot {
  flags?: any[];
  weather?: any;
  racers?: any[];
  race_state?: any;
  standings?: any[];
  [key: string]: any;
}

export interface ConnectOptions {
  /** Topics to subscribe to on open (also used for resync requests). */
  topics?: string[];
  /** WebSocket subprotocols (players pass the token here). */
  protocols?: string[];
  onMessage?: (msg: HeatEnvelope) => void;
  onOpen?: (ws: WebSocket) => void;
  onStatusChange?: (status: 'open' | 'closed') => void;
}

export interface HeatSocket {
  send(data: unknown): boolean;
  close(): void;
  socket(): WebSocket | null;
}

const DEFAULT_BASE = 500;
const DEFAULT_CAP = 30000;

/** Full-jitter exponential backoff delay for the given attempt (0-based). */
export function backoffDelay(
  attempt: number,
  base: number = DEFAULT_BASE,
  cap: number = DEFAULT_CAP,
  rand: () => number = Math.random
): number {
  const window = Math.min(cap, base * Math.pow(2, attempt));
  return Math.floor(rand() * window);
}

/** True when `seq` skipped ahead of `lastSeq` (messages were missed). */
export function detectGap(lastSeq: number | undefined, seq: number | undefined): boolean {
  if (typeof seq !== 'number' || typeof lastSeq !== 'number') return false;
  return seq > lastSeq + 1;
}

/** Merge append-only streams by stable id; existing entries win. */
export function mergeById<T extends { id: number | string }>(existing: T[], incoming: T[]): T[] {
  const seen = new Set(existing.map((e) => e.id));
  const merged = existing.slice();
  for (const item of incoming) {
    if (!seen.has(item.id)) {
      merged.push(item);
      seen.add(item.id);
    }
  }
  return merged;
}

export function connectWithRetry(url: string, opts: ConnectOptions = {}): HeatSocket {
  let ws: WebSocket | null = null;
  let attempt = 0;
  let closedByCaller = false;
  let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
  let lastSeq: number | undefined;
  let hasOpened = false;

  const send = (data: unknown): boolean => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(data));
      return true;
    }
    return false;
  };

  const subscribe = (): void => {
    if (opts.topics && opts.topics.length) send({ type: 'subscribe', topics: opts.topics });
  };

  const requestResync = (): void => {
    send({ type: 'resync', topics: opts.topics || [] });
  };

  const scheduleReconnect = (): void => {
    if (closedByCaller) return;
    const delay = backoffDelay(attempt);
    attempt += 1;
    reconnectTimer = setTimeout(connect, delay);
  };

  function connect(): void {
    ws = opts.protocols ? new WebSocket(url, opts.protocols) : new WebSocket(url);

    ws.onopen = (): void => {
      attempt = 0;
      opts.onStatusChange?.('open');
      subscribe();
      // A reconnect may have missed messages; ask for a fresh snapshot.
      if (hasOpened) requestResync();
      hasOpened = true;
      opts.onOpen?.(ws as WebSocket);
    };

    ws.onmessage = (ev: MessageEvent): void => {
      let msg: HeatEnvelope;
      try {
        msg = JSON.parse(ev.data);
      } catch {
        return;
      }
      if (msg.type === 'hello') {
        // A server restart resets seq; treat hello as the new baseline.
        if (typeof msg.seq === 'number') lastSeq = msg.seq;
        opts.onMessage?.(msg);
        return;
      }
      if (typeof msg.seq === 'number') {
        if (detectGap(lastSeq, msg.seq)) requestResync();
        lastSeq = msg.seq;
      }
      opts.onMessage?.(msg);
    };

    // onerror always trails into onclose; keep the handler so the socket does
    // not surface an unhandled error.
    ws.onerror = (): void => {
      /* handled by onclose */
    };

    ws.onclose = (): void => {
      opts.onStatusChange?.('closed');
      ws = null;
      scheduleReconnect();
    };
  }

  connect();

  return {
    send,
    close(): void {
      closedByCaller = true;
      if (reconnectTimer !== undefined) clearTimeout(reconnectTimer);
      ws?.close();
    },
    socket(): WebSocket | null {
      return ws;
    },
  };
}
