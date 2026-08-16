import './theme';
export {}; // make this a module

document.getElementById('forgot-password-form')?.addEventListener('submit', async (e: Event) => {
    e.preventDefault();
    const email = (document.getElementById('email') as HTMLInputElement).value;

    let ok = true;
    try {
        const res = await fetch('/api/forgot-password', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email })
        });
        ok = res.ok;
    } catch (err) {
        ok = false;
    }

    const form = document.getElementById('forgot-password-form')!;
    form.style.display = 'none';

    const msg = document.getElementById('forgot-message')!;
    msg.classList.remove('d-none');
    if (ok) {
        msg.className = 'alert alert-success';
        msg.textContent = 'If that email is registered, a reset link has been sent. Check your inbox.';
    } else {
        msg.className = 'alert alert-danger';
        msg.textContent = 'Something went wrong sending the reset link. Please try again later.';
    }
});
