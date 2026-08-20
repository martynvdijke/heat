// Sound settings modal - wires the TV sound customization UI.
// Imported by tv.ts for side effects; reads/writes via ts/sound.ts.

import {
    SOUND_CATEGORIES,
    SoundCategory,
    loadSettings,
    saveSettings,
    resetSettings,
    getVolume,
    validateUpload,
    readFileAsDataUrl,
    MAX_UPLOAD_BYTES
} from './sound';
import { showToast } from './toast';

const MODAL_ID = 'sound-settings-modal';
const TRIGGER_ID = 'sound-settings-trigger';
const ROWS_ID = 'sound-settings-rows';

let lastFocused: HTMLElement | null = null;

function categoryLabel(cat: SoundCategory): string {
    const labels: Record<SoundCategory, string> = {
        engine: 'Engine',
        horn: 'Horn',
        finish: 'Finish',
        crash: 'Crash'
    };
    return labels[cat];
}

function buildRows(): void {
    const rows = document.getElementById(ROWS_ID);
    if (!rows) return;
    const settings = loadSettings();

    rows.innerHTML = SOUND_CATEGORIES.map((cat) => {
        const vol = Math.round((settings.volumes[cat] ?? 1) * 100);
        const override = settings.overrides[cat];
        const overrideBadge = override
            ? `<span class="badge bg-success ms-1" title="${override.name}">override</span>`
            : '';
        return `
        <div class="sound-setting-row mb-3" data-category="${cat}">
            <div class="d-flex justify-content-between align-items-center mb-1">
                <label class="form-label mb-0" for="sound-vol-${cat}">${categoryLabel(cat)} ${overrideBadge}</label>
                <span class="sound-vol-readout" id="sound-vol-readout-${cat}">${vol}%</span>
            </div>
            <input type="range" class="form-range" id="sound-vol-${cat}" min="0" max="100" value="${vol}"
                aria-label="${categoryLabel(cat)} volume">
            <div class="d-flex gap-2 mt-1">
                <button type="button" class="btn btn-sm btn-outline-light sound-upload-btn" data-category="${cat}">
                    <i aria-hidden="true" class="fa-solid fa-upload me-1"></i>Upload
                </button>
                <button type="button" class="btn btn-sm btn-outline-danger sound-remove-btn" data-category="${cat}"
                    ${override ? '' : 'disabled'}>
                    <i aria-hidden="true" class="fa-solid fa-trash me-1"></i>Remove
                </button>
                <input type="file" class="d-none sound-file-input" data-category="${cat}" accept="audio/mpeg,audio/ogg,audio/wav">
            </div>
        </div>`;
    }).join('');
}

function openModal(): void {
    const modal = document.getElementById(MODAL_ID);
    if (!modal) return;
    lastFocused = document.activeElement as HTMLElement;
    buildRows();
    modal.hidden = false;
    const closeBtn = modal.querySelector('.sound-settings-close') as HTMLButtonElement | null;
    if (closeBtn) closeBtn.focus();
}

function closeModal(): void {
    const modal = document.getElementById(MODAL_ID);
    if (!modal) return;
    modal.hidden = true;
    if (lastFocused) lastFocused.focus();
}

function updateVolume(cat: SoundCategory, value: number): void {
    const settings = loadSettings();
    settings.volumes[cat] = Math.min(1, Math.max(0, value));
    saveSettings(settings);
    const readout = document.getElementById(`sound-vol-readout-${cat}`);
    if (readout) readout.textContent = `${Math.round(value * 100)}%`;
}

async function handleUpload(cat: SoundCategory, file: File): Promise<void> {
    const error = validateUpload(file);
    if (error) {
        showToast(error, 'error');
        return;
    }
    try {
        const dataUrl = await readFileAsDataUrl(file);
        const settings = loadSettings();
        settings.overrides[cat] = { name: file.name, dataUrl };
        if (!saveSettings(settings)) {
            showToast('Could not save: storage quota exceeded', 'error');
            return;
        }
        showToast(`${categoryLabel(cat)} override saved`, 'success');
        buildRows();
    } catch {
        showToast('Could not read file', 'error');
    }
}

function removeOverride(cat: SoundCategory): void {
    const settings = loadSettings();
    delete settings.overrides[cat];
    saveSettings(settings);
    showToast(`${categoryLabel(cat)} override removed`, 'info');
    buildRows();
}

function resetAll(): void {
    resetSettings();
    showToast('Sound settings reset to defaults', 'success');
    buildRows();
}

document.addEventListener('DOMContentLoaded', () => {
    const trigger = document.getElementById(TRIGGER_ID);
    if (!trigger) return;

    trigger.addEventListener('click', openModal);

    document.addEventListener('click', (e: Event) => {
        const target = e.target as HTMLElement;
        const closeBtn = target.closest('.sound-settings-close');
        if (closeBtn) {
            closeModal();
            return;
        }
        const uploadBtn = target.closest('.sound-upload-btn');
        if (uploadBtn) {
            const cat = uploadBtn.getAttribute('data-category') as SoundCategory;
            const input = document.querySelector(`.sound-file-input[data-category="${cat}"]`) as HTMLInputElement | null;
            if (input) input.click();
            return;
        }
        const removeBtn = target.closest('.sound-remove-btn');
        if (removeBtn) {
            const cat = removeBtn.getAttribute('data-category') as SoundCategory;
            removeOverride(cat);
            return;
        }
        const resetBtn = target.closest('.sound-settings-reset');
        if (resetBtn) {
            resetAll();
            return;
        }
    });

    document.addEventListener('change', (e: Event) => {
        const target = e.target as HTMLInputElement;
        if (target.classList.contains('sound-file-input')) {
            const cat = target.getAttribute('data-category') as SoundCategory;
            const file = target.files?.[0];
            if (file) handleUpload(cat, file);
            target.value = '';
        }
    });

    document.addEventListener('input', (e: Event) => {
        const target = e.target as HTMLInputElement;
        if (target.id && target.id.startsWith('sound-vol-')) {
            const cat = target.id.replace('sound-vol-', '') as SoundCategory;
            updateVolume(cat, parseInt(target.value, 10) / 100);
        }
    });

    document.addEventListener('keydown', (e: KeyboardEvent) => {
        if (e.key === 'Escape') {
            const modal = document.getElementById(MODAL_ID);
            if (modal && !modal.hidden) {
                closeModal();
            }
        }
    });

    // Expose for tests
    (window as any).soundSettings = {
        loadSettings,
        saveSettings,
        resetSettings,
        getVolume,
        validateUpload,
        readFileAsDataUrl,
        MAX_UPLOAD_BYTES
    };
});