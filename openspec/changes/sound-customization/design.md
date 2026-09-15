## Context

Sound flow today: controller buttons POST `/api/sound` `{sound: engine|horn|finish|crash}` → `handlers/race_enhancements.go` `PlaySound` → `SoundBroadcast` → `ws.BroadcastSound` sends `models.SoundCommand{Type:"sound", Sound, Data}` to all clients. Only `ts/tv.ts` consumes it: `data.type === 'sound'` → `tvPlaySound(sound)`, which synthesizes tones (engine 150→80 Hz ramp, finish 440/554/659 arpeggio, flag square 880/440, crash noise burst) with hardcoded gain. Flag sounds also play on `type:'flag'` messages. The controller itself plays no audio — it triggers.

`SoundCommand.Data` (`json.RawMessage`) is available for future payloads but unused today.

## Goals / Non-Goals

**Goals:**
- Per-category volume (including mute) applied to every sound played on a device.
- Optional custom audio file per category, persisted per device.
- Accessible settings UI on the TV page.
- Zero backend changes.

**Non-Goals:**
- Server-side sound files/upload endpoints or shared packs across devices.
- Custom sounds for the start lights (its audio stays synthesized; can adopt the module later).
- Playing sounds on the controller page itself.
- Per-race or per-session sound profiles.

## Decisions

### D1: Client-side only, persisted in localStorage
Sounds are synthesized and played per device, so volume and overrides are per-device concerns. Storing settings client-side avoids backend file storage, auth on public pages (TV is unauthenticated), and new tables. Key `heat.soundSettings.v1`; `JSON.parse` failures or schema mismatches fall back to defaults.

### D2: Settings schema and validation
`{ volumes: Record<category, number>, overrides: Record<category, { name: string, dataUrl: string }> }`. Volumes clamped to 0..1. Uploads restricted to `audio/mpeg`, `audio/ogg`, `audio/wav` and ≤ 2 MB (data-URL budget for localStorage, one override per category); unreadable/invalid files are rejected with feedback and never persisted. `localStorage` quota errors are caught and surfaced as a toast.

### D3: `ts/sound.ts` module
- `getVolume(category): number` — stored value or 1.0 default.
- `playCategory(category): void` — override present → `new Audio(dataUrl)` with volume applied; otherwise the current synthesized tone at volume (0 → silent).
- `loadSettings()/saveSettings()/resetSettings()` helpers with defaults merging.
`ts/tv.ts` `tvPlaySound()` becomes a thin call into `playCategory()`, keeping flag-vs-sound triggers identical.

### D4: Settings modal on tv.html
Trigger button with visible label + `aria-haspopup="dialog"`; modal has per-category rows (slider + percentage + upload button + remove override + reset-all). Esc closes, focus returns to the trigger, rows have `<label>`-associated controls. Modal is hidden by default and does not affect the TV layout.

### D5: Tests
- TS unit test (or Playwright `page.evaluate` assertions): clamping, defaults merge, override storage round-trip, quota/validation rejection.
- Playwright e2e: open modal → set engine volume to 0 → trigger engine sound → assert no audio/`volume` persisted; upload a tiny valid audio file → override stored and replayed on reload; reset → defaults restored.

## Risks / Trade-offs

- [localStorage quota (≈5 MB)] → 2 MB cap per override, one per category, plus quota-error handling.
- [Unplayable/broken custom files] → MIME/size validation up front and `onerror` fallback to the synthesized tone.
- [Modal complexity on a live-updating TV page] → Modal is a separate overlay; TV rendering continues underneath; no interaction with ticker/leaderboard code.
