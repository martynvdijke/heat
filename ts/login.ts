export {}; // make this a module

async function init(): Promise<void> {
    const res = await fetch('/api/check-setup');
    const data = await res.json();
    if (!data.setup) {
        window.location.href = '/setup';
    }
}

document.getElementById('login-form')?.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const fd = new FormData(e.target as HTMLFormElement);

    const res = await fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            username: fd.get('username'),
            password: fd.get('password')
        })
    });

    if (res.ok) {
        window.location.href = '/admin.html';
    } else {
        const existing = document.getElementById('login-error');
        if (existing) {
            existing.textContent = 'Invalid credentials';
        } else {
            const d = document.createElement('div');
            d.id = 'login-error';
            d.className = 'alert alert-danger mt-3';
            d.role = 'alert';
            d.textContent = 'Invalid credentials';
            document.getElementById('login-form')!.appendChild(d);
        }
    }
});

init();
