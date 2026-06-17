/**
 * E-Ink Mode Toggle Module
 *
 * Handles activation/deactivation of e-ink display mode.
 * Activated via:
 * 1. URL parameter ?eink=1 (highest priority)
 * 2. localStorage cookie (persistent per-user)
 * 3. Admin-enforced server-side setting (checked via data attribute)
 */

const EINK_STORAGE_KEY = 'eink';
const EINK_CLASS = 'eink-mode';

function getBody(): HTMLElement {
    return document.body;
}

function isEInkFromUrl(): boolean | null {
    const params = new URLSearchParams(window.location.search);
    const val = params.get('eink');
    if (val === '1') return true;
    if (val === '0') return false;
    return null;
}

function isEInkFromStorage(): boolean {
    return localStorage.getItem(EINK_STORAGE_KEY) === '1';
}

function isEInkAdminForced(): boolean {
    // Check for server-enforced e-ink mode via data attribute on <html> or <body>
    return document.documentElement.getAttribute('data-eink') === '1'
        || document.body.getAttribute('data-eink') === '1';
}

function setEInk(active: boolean): void {
    if (active) {
        document.body.classList.add(EINK_CLASS);
        localStorage.setItem(EINK_STORAGE_KEY, '1');
    } else {
        document.body.classList.remove(EINK_CLASS);
        localStorage.setItem(EINK_STORAGE_KEY, '0');
    }
    updateEInkToggleIcon(active);
}

function updateEInkToggleIcon(active: boolean): void {
    const toggle = document.getElementById('eink-toggle');
    if (!toggle) return;
    const icon = toggle.querySelector('i');
    if (icon) {
        icon.className = active
            ? 'fa-solid fa-file-lines'
            : 'fa-solid fa-file-pen';
    }
    toggle.setAttribute('aria-label', active ? 'Disable E-Ink Mode' : 'Enable E-Ink Mode');
    toggle.setAttribute('title', active ? 'Disable E-Ink Mode' : 'Enable E-Ink Mode');
}

function handleToggleClick(): void {
    const isActive = document.body.classList.contains(EINK_CLASS);
    setEInk(!isActive);
}

function initEInkMode(): void {
    // Admin-enforced setting takes priority
    if (isEInkAdminForced()) {
        setEInk(true);
        return;
    }

    // URL parameter overrides storage
    const urlVal = isEInkFromUrl();
    if (urlVal !== null) {
        setEInk(urlVal);
        return;
    }

    // Check localStorage
    if (isEInkFromStorage()) {
        setEInk(true);
        return;
    }

    // Ensure icon reflects initial state (off)
    updateEInkToggleIcon(false);
}

// Auto-initialize immediately. The script is loaded with `defer`, so the DOM
// is already parsed and ready when this IIFE executes.
initEInkMode();

// Wire up toggle button
const toggle = document.getElementById('eink-toggle');
if (toggle) {
    toggle.addEventListener('click', handleToggleClick);
}

// Export for manual initialization if needed
export { initEInkMode, setEInk, isEInkFromUrl, isEInkAdminForced };
