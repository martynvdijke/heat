export interface WeatherEntry {
    id: number;
    race_id: number;
    condition: string;
    lap_start: number;
    lap_end: number;
    grip_modifier: number;
}

export const weatherIcons: Record<string, string> = { dry: '☀️', damp: '🌦️', wet: '🌧️', torrential: '⛈️' };
export const weatherNames: Record<string, string> = { dry: 'Dry', damp: 'Damp', wet: 'Wet', torrential: 'Torrential' };

export function weatherLabel(condition: string): string {
    return weatherNames[condition] || (condition ? condition.charAt(0).toUpperCase() + condition.slice(1) : 'Dry');
}

export function weatherIcon(condition: string): string {
    return weatherIcons[condition] || '☀️';
}

export function formatGrip(grip: number): string {
    return `${Math.round(grip * 100)}% grip`;
}

/**
 * Returns the active weather entry for the given lap.
 * Active means lap_start <= lap < lap_end (lap_end 999 = open-ended).
 * When lap is unknown (<= 0) the latest entry by lap_start wins.
 * Entries are expected to be ordered by lap_start (as returned by GET /api/weather).
 */
export function getActiveWeather(entries: WeatherEntry[], lap: number): WeatherEntry | null {
    if (!entries || entries.length === 0) return null;
    if (lap > 0) {
        for (const e of entries) {
            if (e.lap_start <= lap && (e.lap_end === 999 || lap < e.lap_end)) return e;
        }
    }
    return entries[entries.length - 1];
}

/**
 * Returns the next scheduled weather change after the given lap, or null.
 */
export function getForecast(entries: WeatherEntry[], lap: number): WeatherEntry | null {
    if (!entries || entries.length === 0 || lap <= 0) return null;
    for (const e of entries) {
        if (e.lap_start > lap) return e;
    }
    return null;
}
