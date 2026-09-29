import { test, expect, Page } from '@playwright/test';

async function loginAsAdmin(page: Page) {
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
      // Setup failed (race with another browser). Fall back to login.
      await page.goto('/login.html');
    }
  }

  if (!page.url().includes('/admin')) {
    await page.waitForSelector('#login-form', { timeout: 10000 });
    await page.fill('#login-form input[name="username"]', 'admin');
    await page.fill('#login-form input[name="password"]', 'admin123');
    await page.click('#login-form button[type="submit"]');
  }

  await page.waitForURL(/admin/, { timeout: 20000 });
  await expect(page.locator('#admin-nav')).toBeVisible({ timeout: 10000 });
}

test.describe.serial('Admin Telegram Settings', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('saves Telegram settings from the dynamically loaded config tab', async ({ page }) => {
    // The config tab is injected by HTMX after admin.js has loaded, so the form
    // must be handled via event delegation (regression: direct listeners never bind).
    await page.click('button[data-tab-id="config"]');
    await expect(page.locator('#telegram-pane')).toBeAttached({ timeout: 15000 });

    await page.click('button[data-bs-target="#telegram-pane"]');
    await expect(page.locator('#telegram-pane')).toHaveClass(/active/, { timeout: 5000 });

    const requestPromise = page.waitForRequest(
      (req) => req.url().includes('/api/telegram-settings') && req.method() === 'POST'
    );

    await page.fill('#telegram-token', '123456789:FAKE-test-token');
    await page.fill('#telegram-chat-id', '-1001234567890');
    await page.click('#telegram-form button[type="submit"]');

    const request = await requestPromise;
    const payload = request.postDataJSON() as Record<string, unknown>;
    expect(payload.bot_token).toBe('123456789:FAKE-test-token');
    expect(payload.default_chat_id).toBe('-1001234567890');

    await expect(page.locator('.toast-notification', { hasText: 'Telegram settings saved!' })).toBeVisible();
  });
});
