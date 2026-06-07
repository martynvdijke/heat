// Shared toast notification system for HEAT
// Provides non-blocking feedback with auto-dismiss

type ToastType = 'success' | 'error' | 'info' | 'warning';

interface ToastOptions {
    message: string;
    type: ToastType;
    duration?: number;   // ms, default 3000
    container?: HTMLElement;
}

const MAX_VISIBLE = 3;
const DEFAULT_DURATION = 3000;
const SHORT_DURATION = 2000;

let toastContainer: HTMLElement | null = null;
let activeToasts: HTMLElement[] = [];

function ensureContainer(container?: HTMLElement): HTMLElement {
    if (container) return container;
    if (toastContainer) return toastContainer;

    toastContainer = document.getElementById('toast-container') as HTMLElement;
    if (!toastContainer) {
        toastContainer = document.createElement('div');
        toastContainer.id = 'toast-container';
        toastContainer.setAttribute('aria-live', 'polite');
        toastContainer.setAttribute('aria-relevant', 'additions');
        document.body.appendChild(toastContainer);
    }
    return toastContainer;
}

function dismiss(toast: HTMLElement): void {
    toast.classList.add('toast-dismissing');
    setTimeout(() => {
        toast.remove();
        activeToasts = activeToasts.filter(t => t !== toast);
    }, 300);
}

function showToast(message: string, type: ToastType = 'info', duration?: number): void {
    const container = ensureContainer();
    const actualDuration = duration ?? (type === 'success' ? DEFAULT_DURATION : type === 'error' ? DEFAULT_DURATION + 1000 : DEFAULT_DURATION);

    // Trim from oldest if over max
    while (activeToasts.length >= MAX_VISIBLE) {
        const oldest = activeToasts.shift();
        if (oldest) {
            oldest.remove();
        }
    }

    const toast = document.createElement('div');
    toast.className = `toast-notification toast-${type}`;
    toast.setAttribute('role', 'alert');

    const iconMap: Record<ToastType, string> = {
        success: 'fa-circle-check',
        error: 'fa-circle-xmark',
        info: 'fa-circle-info',
        warning: 'fa-triangle-exclamation',
    };

    toast.innerHTML = `
        <i class="fa-solid ${iconMap[type] || 'fa-circle-info'}" aria-hidden="true"></i>
        <span class="toast-message">${escapeHtml(message)}</span>
        <button class="toast-close" onclick="dismissToast(this.parentElement)" aria-label="Dismiss notification">&times;</button>
    `;

    container.appendChild(toast);
    activeToasts.push(toast);

    // Trigger animation
    requestAnimationFrame(() => {
        toast.classList.add('toast-visible');
    });

    // Auto-dismiss
    if (actualDuration > 0) {
        setTimeout(() => dismiss(toast), actualDuration);
    }
}

// Expose dismiss for inline onclick
function dismissToast(el: HTMLElement | null): void {
    if (el) dismiss(el);
}

function escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Expose globally for inline onclick handlers
(window as any).showToast = showToast;
(window as any).dismissToast = dismissToast;
