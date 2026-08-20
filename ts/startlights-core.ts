// Start Light System - F1-style 5-light countdown (shared engine)
// Extracted from startlights.ts so both the standalone page and the
// controller's inline widget can drive the same state machine.

export interface StartLightsState {
    phase: 'idle' | 'counting' | 'green' | 'done';
    currentLight: number; // 0-5 (5 = all lit, ready for green)
}

export interface StartLightsHooks {
    setLightState(lightNum: number, state: 'off' | 'red' | 'green'): void;
    showMessage(text: string, subtext?: string): void;
    showStatusBar(text: string): void;
}

// Sequence timing (ms)
export const LIGHT_INTERVAL = 1000;
export const HOLD_ON_LIGHTS = 1000;
export const GREEN_DURATION = 3000;

let audioCtx: AudioContext | null = null;

export function getAudioContext(): AudioContext {
    if (!audioCtx) {
        audioCtx = new (window.AudioContext || (window as any).webkitAudioContext)();
    }
    return audioCtx;
}

export function playBeep(frequency: number, duration: number, volume = 0.3): void {
    try {
        const ctx = getAudioContext();
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = 'square';
        osc.frequency.setValueAtTime(frequency, ctx.currentTime);
        gain.gain.setValueAtTime(volume, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + duration);
        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.start();
        osc.stop(ctx.currentTime + duration);
    } catch {
        // Audio not available
    }
}

export function playHorn(): void {
    try {
        const ctx = getAudioContext();
        // Layered frequencies for a rich horn sound
        [220, 330, 440].forEach((freq, i) => {
            const osc = ctx.createOscillator();
            const gain = ctx.createGain();
            osc.type = 'sawtooth';
            osc.frequency.setValueAtTime(freq, ctx.currentTime + i * 0.05);
            gain.gain.setValueAtTime(0.15, ctx.currentTime + i * 0.05);
            gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 1.5 + i * 0.05);
            osc.connect(gain);
            gain.connect(ctx.destination);
            osc.start(ctx.currentTime + i * 0.05);
            osc.stop(ctx.currentTime + 1.5 + i * 0.05);
        });
    } catch {
        // Audio not available
    }
}

export class StartLightsEngine {
    private state: StartLightsState = { phase: 'idle', currentLight: 0 };
    private timer: ReturnType<typeof setTimeout> | null = null;
    private hooks: StartLightsHooks;

    constructor(hooks: StartLightsHooks) {
        this.hooks = hooks;
    }

    get phase(): StartLightsState['phase'] {
        return this.state.phase;
    }

    get isRunning(): boolean {
        return this.state.phase === 'counting' || this.state.phase === 'green';
    }

    resetAllLights(): void {
        for (let i = 1; i <= 5; i++) {
            this.hooks.setLightState(i, 'off');
        }
    }

    runSequence(): void {
        if (this.state.phase !== 'idle') return;

        this.state = { phase: 'counting', currentLight: 0 };
        this.resetAllLights();
        this.hooks.showMessage('');
        this.hooks.showStatusBar('Sequence started');
        this.hooks.showMessage('', '5 lights');
        this.hooks.showStatusBar('START LIGHTS • SEQUENCE');

        let lightIndex = 0;

        const lightNext = (): void => {
            if (this.state.phase !== 'counting') return;

            lightIndex++;
            if (lightIndex > 5) {
                // All lights on - hold then green
                this.state.currentLight = 5;
                this.hooks.showMessage('', 'All lights on');
                playBeep(880, 0.3, 0.5);

                this.timer = setTimeout(() => {
                    if (this.state.phase !== 'counting') return;
                    // ALL GREEN!
                    this.state.phase = 'green';
                    for (let i = 1; i <= 5; i++) {
                        this.hooks.setLightState(i, 'green');
                    }
                    this.hooks.showMessage('GO! GO! GO!', '');
                    this.hooks.showStatusBar('🟢 GREEN FLAG • RACE START!');
                    playHorn();

                    this.timer = setTimeout(() => {
                        this.state.phase = 'done';
                        this.hooks.showMessage('Race Started', '');
                        this.hooks.showStatusBar('RACE IS ON');
                        this.timer = setTimeout(() => {
                            this.state.phase = 'idle';
                            this.state.currentLight = 0;
                            this.resetAllLights();
                            this.hooks.showMessage('Start Lights', 'Ready');
                            this.hooks.showStatusBar('START LIGHTS • READY');
                        }, 2000);
                    }, GREEN_DURATION);
                }, HOLD_ON_LIGHTS);
                return;
            }

            this.state.currentLight = lightIndex;
            this.hooks.setLightState(lightIndex, 'red');
            this.hooks.showMessage('', `${lightIndex}/5 lights`);
            playBeep(440 + lightIndex * 60, 0.2, 0.4);

            this.timer = setTimeout(lightNext, LIGHT_INTERVAL);
        };

        // Start the sequence
        this.timer = setTimeout(lightNext, LIGHT_INTERVAL);
    }

    abortSequence(): void {
        if (this.timer) {
            clearTimeout(this.timer);
            this.timer = null;
        }
        this.state = { phase: 'idle', currentLight: 0 };
        this.resetAllLights();
        this.hooks.showMessage('Aborted', '');
        this.hooks.showStatusBar('START LIGHTS • ABORTED');
    }

    reset(): void {
        this.abortSequence();
        this.hooks.showMessage('Start Lights', 'Ready');
        this.hooks.showStatusBar('START LIGHTS • READY');
    }

    handleCommand(cmd: any): void {
        if (cmd.state === 'sequence') {
            this.runSequence();
        } else if (cmd.state === 'abort') {
            this.abortSequence();
        } else if (cmd.state === 'reset') {
            this.reset();
        }
    }
}