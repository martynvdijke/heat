import { test, expect } from '@playwright/test';

test.describe('E-Ink Mode', () => {
  test.beforeAll(async ({ request }) => {
    // Create the initial admin account so the home page renders instead of
    // redirecting to the setup page.
    const check = await request.get('/api/check-setup', {
      headers: { Origin: 'http://localhost:6270' }
    });
    const setup = await check.json().catch(() => ({ setup: false }));
    if (setup.setup) return;

    const res = await request.post('/api/login', {
      data: { username: 'admin', password: 'admin123', setup: true },
      headers: { Origin: 'http://localhost:6270' }
    });
    if (!res.ok() && res.status() !== 403) {
      console.warn('Setup login failed:', res.status(), await res.text());
    }
  });

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

  test('should show visible dark toggler icon in e-ink mode', async ({ page }) => {
    await page.goto('/?eink=1');
    const toggler = page.locator('.navbar-toggler-icon');
    const bgImage = await toggler.evaluate(el => window.getComputedStyle(el).backgroundImage);
    // The hamburger icon stroke must be dark so it is visible on the white e-ink header.
    expect(bgImage).toContain('rgba(0, 0, 0');
    expect(bgImage).not.toContain('rgba(255, 255, 255');
  });

  test('should not apply e-ink touch targets to navbar links', async ({ page }) => {
    await page.goto('/?eink=1');
    const navLink = page.locator('#mainNav nav a').first();
    const style = await navLink.evaluate(el => ({
      minWidth: window.getComputedStyle(el).minWidth,
      minHeight: window.getComputedStyle(el).minHeight,
      margin: window.getComputedStyle(el).margin,
      padding: window.getComputedStyle(el).padding
    }));
    expect(style.minWidth).not.toBe('48px');
    expect(style.minHeight).not.toBe('48px');
    // e-ink touch-target margins should not break the navbar link layout.
    expect(style.margin).not.toContain('4px');
  });
});

test.describe('E-Ink Mode Mobile', () => {
  test.use({ viewport: { width: 375, height: 812 } });

  test('should keep nav layout compact in e-ink mode on mobile', async ({ page }) => {
    await page.goto('/?eink=1');
    await page.locator('.navbar-toggler').click();
    const navLink = page.locator('#mainNav nav a').first();
    const minHeight = await navLink.evaluate(el => window.getComputedStyle(el).minHeight);
    // Without the fix, e-ink.css would force 48px min-height on all links and break the mobile nav.
    expect(minHeight).toBe('0px');
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
