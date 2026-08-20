// Sound customization - client-side volume + override settings for TV sound FX.
// Stored in localStorage under `heat.soundSettings.v1`.

export type SoundCategory = 'engine' | 'horn' | 'finish' | 'crash';

export interface SoundOverride {
    name: string;
    dataUrl: string;
}

export interface SoundSettings {
    volumes: Record<SoundCategory, number>;
    overrides: Partial<Record<SoundCategory, SoundOverride>>;
}

export const SOUND_CATEGORIES: SoundCategory[] = ['engine', 'horn', 'finish', 'crash'];

export const STORAGE_KEY = 'heat.soundSettings.v1';
export const MAX_UPLOAD_BYTES = 2 * 1024 * 1024; // 2MB
export const ALLOWED_MIME = new Set(['audio/mpeg', 'audio/ogg', 'audio/wav']);

export function defaultSettings(): SoundSettings {
    return {
        volumes: { engine: 1, horn: 1, finish: 1, crash: 1 },
        overrides: {}
    };
}

export function clampVolume(v: number): number {
    if (typeof v !== 'number' || Number.isNaN(v)) return 1;
    return Math.min(1, Math.max(0, v));
}

function isValidOverride(o: unknown): o is SoundOverride {
    if (!o || typeof o !== 'object') return false;
    const rec = o as Record<string, unknown>;
    return typeof rec.name === 'string' && typeof rec.dataUrl === 'string';
}

export function loadSettings(): SoundSettings {
    const defaults = defaultSettings();
    try {
        const raw = localStorage.getItem(STORAGE_KEY);
        if (!raw) return defaults;
        const parsed = JSON.parse(raw);
        if (!parsed || typeof parsed !== 'object') return defaults;

        const volumes = { ...defaults.volumes };
        const parsedVolumes = (parsed as Record<string, unknown>).volumes;
        if (parsedVolumes && typeof parsedVolumes === 'object') {
            for (const cat of SOUND_CATEGORIES) {
                const v = (parsedVolumes as Record<string, unknown>)[cat];
                if (typeof v === 'number') volumes[cat] = clampVolume(v);
            }
        }

        const overrides: Partial<Record<SoundCategory, SoundOverride>> = {};
        const parsedOverrides = (parsed as Record<string, unknown>).overrides;
        if (parsedOverrides && typeof parsedOverrides === 'object') {
            for (const cat of SOUND_CATEGORIES) {
                const o = (parsedOverrides as Record<string, unknown>)[cat];
                if (isValidOverride(o)) overrides[cat] = o;
            }
        }

        return { volumes, overrides };
    } catch {
        return defaults;
    }
}

export function saveSettings(settings: SoundSettings): boolean {
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
        return true;
    } catch {
        return false;
    }
}

export function resetSettings(): SoundSettings {
    const defaults = defaultSettings();
    saveSettings(defaults);
    return defaults;
}

export function getVolume(category: SoundCategory): number {
    const settings = loadSettings();
    return settings.volumes[category] ?? 1;
}

export function validateUpload(file: File): string | null {
    if (!ALLOWED_MIME.has(file.type)) {
        return 'Unsupported file type. Use MP3, OGG or WAV.';
    }
    if (file.size > MAX_UPLOAD_BYTES) {
        return 'File too large. Maximum size is 2MB.';
    }
    return null;
}

export function readFileAsDataUrl(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result));
        reader.onerror = () => reject(new Error('Could not read file'));
        reader.readAsDataURL(file);
    });
}

let audioCtx: AudioContext | null = null;

function getAudioContext(): AudioContext {
    if (!audioCtx) {
        audioCtx = new (window.AudioContext || (window as any).webkitAudioContext)();
    }
    return audioCtx;
}

// Synthesized fallback tones, scaled by volume. Mirrors the original tv.ts
// synthesis plus a horn case (layered sawtooth, same as start lights).
export function playSynthesized(category: SoundCategory, volume: number): void {
    if (volume <= 0) return;
    try {
        const ctx = getAudioContext();
        const t = ctx.currentTime;

        if (category === 'engine') {
            const osc = ctx.createOscillator();
            const gain = ctx.createGain();
            osc.type = 'sawtooth';
            osc.frequency.setValueAtTime(150, t);
            osc.frequency.exponentialRampToValueAtTime(80, t + 0.3);
            gain.gain.setValueAtTime(0.1 * volume, t);
            gain.gain.exponentialRampToValueAtTime(0.01, t + 0.3);
            osc.connect(gain);
            gain.connect(ctx.destination);
            osc.start(t);
            osc.stop(t + 0.3);
        } else if (category === 'finish') {
            [440, 554, 659].forEach((freq, i) => {
                const osc = ctx.createOscillator();
                const gain = ctx.createGain();
                osc.type = 'square';
                osc.frequency.setValueAtTime(freq, t + i * 0.15);
                gain.gain.setValueAtTime(0.1 * volume, t + i * 0.15);
                gain.gain.exponentialRampToValueAtTime(0.01, t + 0.5 + i * 0.15);
                osc.connect(gain);
                gain.connect(ctx.destination);
                osc.start(t + i * 0.15);
                osc.stop(t + 0.5 + i * 0.15);
            });
        } else if (category === 'horn') {
            [220, 330, 440].forEach((freq, i) => {
                const osc = ctx.createOscillator();
                const gain = ctx.createGain();
                osc.type = 'sawtooth';
                osc.frequency.setValueAtTime(freq, t + i * 0.05);
                gain.gain.setValueAtTime(0.15 * volume, t + i * 0.05);
                gain.gain.exponentialRampToValueAtTime(0.01, t + 1.5 + i * 0.05);
                osc.connect(gain);
                gain.connect(ctx.destination);
                osc.start(t + i * 0.05);
                osc.stop(t + 1.5 + i * 0.05);
            });
        } else if (category === 'crash') {
            const bufferSize = ctx.sampleRate * 0.3;
            const buffer = ctx.createBuffer(1, bufferSize, ctx.sampleRate);
            const data = buffer.getChannelData(0);
            for (let i = 0; i < bufferSize; i++) {
                data[i] = (Math.random() * 2 - 1) * (1 - i / bufferSize);
            }
            const src = ctx.createBufferSource();
            const gain = ctx.createGain();
            src.buffer = buffer;
            gain.gain.setValueAtTime(0.1 * volume, t);
            gain.gain.exponentialRampToValueAtTime(0.01, t + 0.3);
            src.connect(gain);
            gain.connect(ctx.destination);
            src.start(t);
            src.stop(t + 0.3);
        }
    } catch {
        // Audio not available
    }
}

// Play a category: override audio if present, else synthesized tone.
// 'flag' maps to the horn category volume (flags use the horn sound).
export function playCategory(category: SoundCategory | 'flag'): void {
    const cat: SoundCategory = category === 'flag' ? 'horn' : category;
    const volume = getVolume(cat);
    if (volume <= 0) return;

    const settings = loadSettings();
    const override = settings.overrides[cat];
    if (override && override.dataUrl) {
        const audio = new Audio(override.dataUrl);
        audio.volume = volume;
        audio.onerror = () => playSynthesized(cat, volume);
        audio.play().catch(() => playSynthesized(cat, volume));
        return;
    }

    playSynthesized(cat, volume);
}