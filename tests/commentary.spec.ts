import { test, expect } from '@playwright/test';

async function loginAsAdmin(page: import('@playwright/test').Page) {
    await page.goto('/admin.html');
    if (await page.locator('#admin-nav').count() > 0) return;
    await page.waitForSelector('#setup-form, #login-form', { timeout: 10000 });
    if (await page.locator('#setup-form').count() > 0) {
        await page.fill('#setup-form input[name="username"]', 'admin');
        await page.fill('#setup-form input[name="password"]', 'admin123');
        await page.fill('#setup-form input[name="confirm_password"]', 'admin123');
        await page.click('#setup-form button[type="submit"]');
        try {
            await page.waitForURL(/admin/, { timeout: 5000 });
        } catch {
            await page.goto('/login.html');
        }
    }
    if (!page.url().includes('/admin')) {
        await page.waitForSelector('#login-form', { timeout: 10000 });
        await page.fill('#login-form input[name="username"]', 'admin');
        await page.fill('#login-form input[name="password"]', 'admin123');
        await page.click('#login-form button[type="submit"]');
    }
    await expect(page.locator('#admin-nav')).toBeVisible({ timeout: 10000 });
}

test.describe('Commentary Feed', () => {
    test('controller manual entry appears on TV ticker', async ({ page }) => {
        await loginAsAdmin(page);
        await page.goto('/controller.html');
        await expect(page.locator('#commentary-feed')).toBeVisible();

        const message = `Manual entry ${Date.now()}`;
        await page.fill('#commentary-message', message);
        await page.click('[data-action="sendCommentary"]');

        // The entry should appear in the controller's own feed (via WS broadcast).
        await expect(page.locator('#commentary-feed .commentary-item').filter({ hasText: message })).toBeVisible({ timeout: 10000 });

        // Open the TV page; the ticker picks the entry up via WS or polling.
        await page.goto('/static/tv.html');
        await expect(page.locator('#tv-commentary .commentary-item').filter({ hasText: message })).toBeVisible({ timeout: 15000 });
    });

    test('commentary item renders lap, driver and message', async ({ page }) => {
        await loginAsAdmin(page);
        await page.goto('/static/tv.html');

        // Create a dedicated racer so the driver name is guaranteed to resolve
        // (a hardcoded racer_id may not exist in a fresh database).
        const racerName = `Commentary Driver ${Date.now()}`;
        const created = await page.evaluate(async (name) => {
            const res = await fetch('/api/racers', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    id: 0, name, profile_picture: '', car_color: '#00ff00',
                    car_name: 'Commentary Car', points: 0, rank: 0, position: 0
                })
            });
            return res.ok;
        }, racerName);
        expect(created).toBeTruthy();
        const racers = await (await page.request.get('/api/racers')).json();
        const racer = racers.find((r: any) => r.name === racerName);
        expect(racer).toBeTruthy();

        const message = `Structured entry ${Date.now()}`;
        const res = await page.request.post('/api/commentary', {
            data: { race_id: 0, lap: 7, racer_id: racer.id, message }
        });
        expect(res.ok()).toBeTruthy();

        const item = page.locator('#tv-commentary .commentary-item').filter({ hasText: message });
        await expect(item).toBeVisible({ timeout: 15000 });
        await expect(item.locator('.commentary-lap')).toHaveText('L7');
        await expect(item.locator('.commentary-driver')).toHaveText(racerName);
        await expect(item.locator('.commentary-message')).toHaveText(message);
    });

    test('old entries fade out after 30 seconds', async ({ page }) => {
        await page.clock.install();
        await page.goto('/static/tv.html');

        const message = `Fading entry ${Date.now()}`;
        const res = await page.request.post('/api/commentary', {
            data: { race_id: 0, lap: 1, message }
        });
        expect(res.ok()).toBeTruthy();

        const item = page.locator('#tv-commentary .commentary-item').filter({ hasText: message });
        await expect(item).toBeVisible({ timeout: 15000 });
        await expect(item).not.toHaveClass(/commentary-fade/);

        await page.clock.fastForward(30500);
        await expect(item).toHaveClass(/commentary-fade/);
    });

    test('spectator page shows commentary section', async ({ page }) => {
        await page.goto('/static/spectator.html');
        await expect(page.locator('#spec-commentary')).toBeVisible();

        const message = `Spectator entry ${Date.now()}`;
        const res = await page.request.post('/api/commentary', {
            data: { race_id: 0, lap: 2, message }
        });
        expect(res.ok()).toBeTruthy();

        await expect(page.locator('#spec-commentary .commentary-item').filter({ hasText: message })).toBeVisible({ timeout: 15000 });
    });
});
