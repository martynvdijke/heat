## Why

Sound FX are hardcoded synthesized tones: the operator triggers engine/horn/finish/crash from the controller (`POST /api/sound` → `SoundBroadcast`), and `ts/tv.ts` `tvPlaySound()` synthesizes each sound with WebAudio at a fixed volume. There is no volume control, no way to mute, and no way to use custom sounds — a room full of players or a loud environment has no recourse but to ignore the sounds entirely. `SoundCommand.Data` exists in the model but is unused.

## What Changes

- **New `ts/sound.ts` module**: shared playback for all sound categories (`engine`, `horn`, `finish`, `crash`, `flag`). Defaults reproduce the current synthesized tones exactly; supports per-category volume and optional custom audio overrides.
- **Per-device settings**, persisted in `localStorage` under a versioned key (`heat.soundSettings.v1`): `{ volumes: { engine, horn, finish, crash, flag: 0..1 }, overrides: { engine?: { name, dataUrl }, ... } }`.
- **Settings modal on the TV page** (a11y-labeled trigger button): volume slider per category with percentage readout, custom sound upload per category (audio file → data URL, validated type + size cap, ≤ 2 MB), remove-override and reset-to-defaults actions. Keyboard dismiss (Esc), focus returns to the trigger.
- **`ts/tv.ts` refactored** to play through the module (no behavior change when settings are untouched).
- **No backend changes** — sounds are synthesized/played per device, so customization is a per-device client concern.
- **Tests**: Playwright e2e for the modal (persist across reload, override active, reset) plus a small TS unit test for the settings helpers.

## Capabilities

### New Capabilities
- `sound-customization`: Per-device sound volume and custom sound overrides for the race sound FX, configured from a settings modal on the TV page.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `ts/sound.ts` (new): volume-aware playback, default tones, override loading, settings persistence helpers.
- `ts/tv.ts`: `tvPlaySound()` delegates to the module.
- `static/tv.html`: settings modal markup + trigger button.
- `ts/tv.ts` (or a small `ts/sound-settings.ts`): modal wiring.
- Tests: Playwright + TS unit. No Go changes.
