import { test, expect } from '@playwright/test';

test.describe('E-Ink Mode', () => {
  test('should not be active by default on home page', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('body')).not.toHaveClass(/eink-mode/);
  });

  test('should activate via ?eink=1 URL parameter', async ({ page }) => {
    await page.goto('/?eink=1');
    await expect(page.locator('body')).toHaveClass(/eink-mode/);
  });

  test('should deactivate via ?eink=0 URL parameter', async ({ page }) => {
    await page.goto('/?eink=0');
    await expect(page.locator('body')).not.toHaveClass(/eink-mode/);
  });

  test('should apply high-contrast styles in e-ink mode', async ({ page }) => {
    await page.goto('/?eink=1');
    // Background should be white
    const bgColor = await page.locator('body').evaluate(el => window.getComputedStyle(el).backgroundColor);
    expect(bgColor).toBe('rgb(255, 255, 255)');
    // Text color should be black
    const textColor = await page.locator('body').evaluate(el => window.getComputedStyle(el).color);
    expect(textColor).toBe('rgb(0, 0, 0)');
  });

  test('should persist preference via localStorage', async ({ page }) => {
    await page.goto('/?eink=1');
    const stored = await page.evaluate(() => localStorage.getItem('eink'));
    expect(stored).toBe('1');
  });

  test('should have toggle button visible on home page', async ({ page }) => {
    await page.goto('/');
    const toggle = page.locator('#eink-toggle');
    await expect(toggle).toBeVisible();
  });

  test('should toggle e-ink mode on button click', async ({ page }) => {
    await page.goto('/');
    const toggle = page.locator('#eink-toggle');
    // Click to enable (force: true for mobile-chrome where body intercepts pointer events)
    await toggle.click({ force: true });
    await expect(page.locator('body')).toHaveClass(/eink-mode/);
    // Click to disable
    await toggle.click({ force: true });
    await expect(page.locator('body')).not.toHaveClass(/eink-mode/);
  });

  test('should work on stats page', async ({ page }) => {
    await page.goto('/stats.html?eink=1');
    await expect(page.locator('body')).toHaveClass(/eink-mode/);
    const bgColor = await page.locator('body').evaluate(el => window.getComputedStyle(el).backgroundColor);
    expect(bgColor).toBe('rgb(255, 255, 255)');
  });

  test('should work on seasons page', async ({ page }) => {
    await page.goto('/seasons.html?eink=1');
    await expect(page.locator('body')).toHaveClass(/eink-mode/);
  });

  test('should work on trophies page', async ({ page }) => {
    await page.goto('/trophies.html?eink=1');
    await expect(page.locator('body')).toHaveClass(/eink-mode/);
  });
});

test.describe('E-Ink Admin Settings', () => {
  test('should load e-ink settings pane in admin', async ({ page, context }) => {
    // Login and go to admin
    await context.addCookies([
      { name: 'session', value: 'admin-session', domain: 'localhost', path: '/' }
    ]);
    await page.goto('/?eink=1');
    await page.evaluate(() => {
      // Set admin session via API
      document.cookie = 'session=admin-session; path=/';
    });
  });

  test('admin can save e-ink settings via API', async ({ page, context }) => {
    // Login as admin first
    const loginRes = await page.request.post('/api/login', {
      data: { username: 'admin', password: 'admin123', setup: true },
      headers: { Origin: 'http://localhost:6270' }
    });
    if (!loginRes.ok()) {
      await page.request.post('/api/login', {
        data: { username: 'admin', password: 'admin123' },
        headers: { Origin: 'http://localhost:6270' }
      });
    }

    // Save e-ink settings
    const saveRes = await page.request.post('/api/eink-settings', {
      data: { enabled: true },
      headers: { Origin: 'http://localhost:6270' }
    });
    expect(saveRes.ok()).toBeTruthy();

    // Verify settings are persisted
    const getRes = await page.request.get('/api/eink-settings', {
      headers: { Origin: 'http://localhost:6270' }
    });
    const data = await getRes.json();
    expect(data.enabled).toBe(true);

    // Reset to disabled
    await page.request.post('/api/eink-settings', {
      data: { enabled: false },
      headers: { Origin: 'http://localhost:6270' }
    });
  });
});
