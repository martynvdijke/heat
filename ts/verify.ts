import './theme';
export {}; // make this a module

function escapeHtml(text: string | null | undefined): string {
    const div = document.createElement('div');
    div.textContent = text ?? '';
    return div.innerHTML;
}

function showError(title: string, message: string): void {
    const body = document.getElementById('verify-body');
    if (!body) return;
    body.innerHTML = `
        <i class="fa-solid fa-triangle-exclamation fa-3x text-danger mb-3"></i>
        <h5>${escapeHtml(title)}</h5>
        <p class="text-muted mb-0">${escapeHtml(message)}</p>
    `;
}

async function init(): Promise<void> {
    const token = new URLSearchParams(window.location.search).get('token') || '';
    if (!token) {
        showError('Invalid link', 'No verification token was provided. Start a new sign-in from the Telegram bot with /login.');
        return;
    }

    let racerName = '';
    try {
        const res = await fetch('/api/telegram/verify/validate?token=' + encodeURIComponent(token));
        if (!res.ok) {
            showError('Link expired', 'This sign-in link is invalid or has expired. Request a new one from the Telegram bot with /login.');
            return;
        }
        const data = await res.json();
        racerName = data.racer_name || '';
    } catch {
        showError('Something went wrong', 'We could not check this link. Please try again.');
        return;
    }

    const body = document.getElementById('verify-body');
    if (!body) return;
    body.innerHTML = `
        <i class="fa-solid fa-circle-check fa-3x text-success mb-3"></i>
        <h5>Sign in as ${escapeHtml(racerName || 'your racer')}?</h5>
        <p class="text-muted">Confirm to link this Telegram chat and open your personal stats and upgrades.</p>
        <button id="confirm-btn" class="btn btn-danger w-100 mt-2">Confirm sign-in</button>
    `;

    document.getElementById('confirm-btn')?.addEventListener('click', async () => {
        const btn = document.getElementById('confirm-btn') as HTMLButtonElement;
        btn.disabled = true;
        btn.textContent = 'Signing in...';
        try {
            const res = await fetch('/api/telegram/verify', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ token }),
            });
            if (!res.ok) {
                showError('Sign-in failed', 'This link is invalid or has already been used. Request a new one with /login.');
                return;
            }
            window.location.href = '/me.html';
        } catch {
            showError('Sign-in failed', 'We could not complete your sign-in. Please try again.');
        }
    });
}

init();
