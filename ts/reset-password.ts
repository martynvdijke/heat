import './theme';
export {}; // make this a module

function showInvalid(): void {
    (document.getElementById('reset-form-wrap') as HTMLElement).style.display = 'none';
    (document.getElementById('reset-invalid') as HTMLElement).classList.remove('d-none');
}

function showError(message: string): void {
    const err = document.getElementById('reset-error')!;
    err.classList.remove('d-none');
    err.textContent = message;
}

async function validateToken(token: string): Promise<boolean> {
    try {
        const res = await fetch('/api/reset-password/validate?token=' + encodeURIComponent(token));
        const data = await res.json();
        return !!data.valid;
    } catch (err) {
        return false;
    }
}

async function init(): Promise<void> {
    const params = new URLSearchParams(window.location.search);
    const token = params.get('token') || '';

    if (!token || !(await validateToken(token))) {
        showInvalid();
        return;
    }

    document.getElementById('reset-password-form')?.addEventListener('submit', async (e: Event) => {
        e.preventDefault();
        const password = (document.getElementById('password') as HTMLInputElement).value;
        const confirm = (document.getElementById('confirm-password') as HTMLInputElement).value;

        if (password.length < 8) {
            showError('Password must be at least 8 characters');
            return;
        }
        if (password !== confirm) {
            showError('Passwords do not match!');
            return;
        }

        const res = await fetch('/api/reset-password', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token, password })
        });

        if (res.ok) {
            window.location.href = '/admin.html';
        } else {
            const data = await res.json().catch(() => ({}));
            showError(data.error || 'Failed to reset password. The link may be invalid or expired.');
        }
    });
}

init();
