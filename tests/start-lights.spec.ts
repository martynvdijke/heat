import { test, expect } from '@playwright/test';

test.describe('Start Lights Page', () => {
  test('should load start lights page with correct title', async ({ page }) => {
    await page.goto('/static/startlights.html');
    await expect(page).toHaveTitle(/HEAT: Start Lights/);
  });

  test('should display HEAT START LIGHTS logo', async ({ page }) => {
    await page.goto('/static/startlights.html');
    await expect(page.locator('.logo')).toContainText('HEAT START LIGHTS');
  });

  test('should have a close button with aria-label', async ({ page }) => {
    await page.goto('/static/startlights.html');
    const closeBtn = page.locator('button[aria-label="Close"]');
    await expect(closeBtn).toBeVisible();
  });

  test('should have status bar showing READY', async ({ page }) => {
    await page.goto('/static/startlights.html');
    const statusBar = page.locator('#start-status-bar');
    await expect(statusBar).toBeVisible();
    await expect(statusBar).toContainText('READY');
  });

  test('should have message area elements', async ({ page }) => {
    await page.goto('/static/startlights.html');
    await expect(page.locator('#start-message')).toBeVisible();
    await expect(page.locator('#start-submessage')).toBeVisible();
  });

  test('should load compiled startlights JS', async ({ page }) => {
    await page.goto('/static/startlights.html');
    // Verify the script tag exists
    const script = page.locator('script[src="/static/js/startlights.js"]');
    await expect(script).toBeAttached();
  });

  test('should not have user-scalable=no in viewport', async ({ page }) => {
    await page.goto('/static/startlights.html');
    const viewport = await page.evaluate(() => {
      const meta = document.querySelector('meta[name="viewport"]');
      return meta ? meta.getAttribute('content') : null;
    });
    expect(viewport).not.toBeNull();
    expect(viewport).not.toContain('user-scalable=no');
    expect(viewport).not.toContain('maximum-scale=1.0');
  });

  test('should have prefers-reduced-motion styles', async ({ page }) => {
    await page.goto('/static/startlights.html');
    const hasReducedMotion = await page.evaluate(() => {
      for (const sheet of document.styleSheets) {
        try {
          for (const rule of sheet.cssRules) {
            if (rule instanceof CSSMediaRule && rule.media?.mediaText?.includes('prefers-reduced-motion')) {
              return true;
            }
          }
        } catch (_) { /* skip cross-origin */ }
      }
      return false;
    });
    expect(hasReducedMotion).toBe(true);
  });
});

test.describe('Start Lights API', () => {
  test('should trigger start lights sequence via API', async ({ page }) => {
    const res = await page.request.post('/api/flags', {
      data: { type: 'flag', flag: 'startlights', state: 'trigger' }
    });
    expect(res.ok()).toBeTruthy();
  });

  test('should abort start lights via API', async ({ page }) => {
    const res = await page.request.post('/api/flags', {
      data: { type: 'flag', flag: 'startlights', state: 'abort' }
    });
    expect(res.ok()).toBeTruthy();
  });

  test('should reset start lights via API', async ({ page }) => {
    const res = await page.request.post('/api/flags', {
      data: { type: 'flag', flag: 'startlights', state: 'reset' }
    });
    expect(res.ok()).toBeTruthy();
  });
});

test.describe('Web Design — Player Page', () => {
  test('should load player page', async ({ page }) => {
    await page.goto('/player.html');
    await expect(page).toHaveTitle(/HEAT: Player Dashboard/);
  });

  test('should not have user-scalable=no in viewport', async ({ page }) => {
    await page.goto('/player.html');
    const viewport = await page.evaluate(() => {
      const meta = document.querySelector('meta[name="viewport"]');
      return meta ? meta.getAttribute('content') : null;
    });
    expect(viewport).not.toBeNull();
    expect(viewport).not.toContain('user-scalable=no');
    expect(viewport).toContain('initial-scale=1.0');
  });

  test('should have visually-hidden label for player-select', async ({ page }) => {
    await page.goto('/player.html');
    const label = page.locator('label[for="player-select"].visually-hidden');
    await expect(label).toBeAttached();
    await expect(label).toContainText('Select your driver');
  });

  test('should have visually-hidden label for device-name', async ({ page }) => {
    await page.goto('/player.html');
    const label = page.locator('label[for="device-name"].visually-hidden');
    await expect(label).toBeAttached();
    await expect(label).toContainText('Device Name');
  });

  test('should have aria-label on logout button', async ({ page }) => {
    await page.goto('/player.html');
    await expect(page.locator('button[aria-label="Logout"]')).toBeAttached();
  });

  test('should have aria-label on turbo button', async ({ page }) => {
    await page.goto('/player.html');
    await expect(page.locator('button[aria-label="Use turbo"]')).toBeAttached();
  });
});

test.describe('Web Design — Spectator Page', () => {
  test('should load spectator page', async ({ page }) => {
    await page.goto('/spectator.html');
    await expect(page).toHaveTitle(/HEAT: Spectator View/);
  });

  test('should have aria-label on home link', async ({ page }) => {
    await page.goto('/spectator.html');
    await expect(page.locator('a[aria-label="Home"]')).toBeAttached();
  });

  test('should have aria-live on grid container', async ({ page }) => {
    await page.goto('/spectator.html');
    await expect(page.locator('#spec-grid')).toHaveAttribute('aria-live', 'polite');
  });

  test('should have aria-live on events container', async ({ page }) => {
    await page.goto('/spectator.html');
    await expect(page.locator('#spec-events')).toHaveAttribute('aria-live', 'polite');
  });

  test('should not have user-scalable=no in viewport', async ({ page }) => {
    await page.goto('/spectator.html');
    const viewport = await page.evaluate(() => {
      const meta = document.querySelector('meta[name="viewport"]');
      return meta ? meta.getAttribute('content') : null;
    });
    expect(viewport).not.toBeNull();
    expect(viewport).not.toContain('user-scalable=no');
  });
});

test.describe('Controller Inline Start Lights Widget', () => {
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
    await page.waitForURL(/admin/, { timeout: 20000 });
    await expect(page.locator('#admin-nav')).toBeVisible({ timeout: 10000 });
  }

  async function waitForControllerWs(page: import('@playwright/test').Page) {
    await page.waitForFunction(() => (window as any).__controllerWsConnected === true, undefined, { timeout: 5000 });
  }

  // The broadcast can race with server-side WS registration, so retry the
  // trigger until the first bulb actually lights.
  async function triggerSequenceUntilLit(page: import('@playwright/test').Page) {
    await expect(async () => {
      await page.request.post('/api/flags', {
        data: { type: 'flag', flag: 'startlights', state: 'sequence' }
      });
      await page.waitForTimeout(500);
      const lit = await page.locator('#start-light-1 .start-light-bulb.red').count();
      expect(lit).toBeGreaterThan(0);
    }).toPass({ timeout: 8000 });
  }

  test('should render the inline widget with five bulbs and READY status', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/controller.html');
    const widget = page.locator('#controller-start-lights');
    await expect(widget).toBeVisible();
    await expect(widget.locator('.start-light--inline')).toHaveCount(5);
    await expect(page.locator('#start-status-bar')).toContainText('READY');
    await expect(page.locator('#abort-start-lights')).toBeDisabled();
  });

  test('should light bulbs sequentially and reach green when triggered', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/controller.html');
    await expect(page.locator('#controller-start-lights')).toBeVisible();
    await waitForControllerWs(page);

    // Single trigger, then wait for green. The engine ignores re-triggers while
    // counting, so retrying the whole cycle here would just restart the
    // sequence; a single trigger + generous wait is reliable (workers:1 means
    // no cross-project abort broadcasts can reset it).
    await triggerSequenceUntilLit(page);
    await expect(page.locator('#start-light-5 .start-light-bulb.green')).toBeVisible({ timeout: 15000 });
    await expect(page.locator('#start-status-bar')).toContainText('GREEN FLAG');
  });

  test('should abort mid-sequence and reset bulbs', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/controller.html');
    await expect(page.locator('#controller-start-lights')).toBeVisible();
    await waitForControllerWs(page);

    await triggerSequenceUntilLit(page);

    // Retry the abort until the bulbs actually go dark.
    await expect(async () => {
      await page.request.post('/api/flags', {
        data: { type: 'flag', flag: 'startlights', state: 'abort' }
      });
      await page.waitForTimeout(300);
      const red = await page.locator('#start-light-1 .start-light-bulb.red').count();
      expect(red).toBe(0);
    }).toPass({ timeout: 8000 });

    await expect(page.locator('#start-status-bar')).toContainText('ABORTED');
    await expect(page.locator('#abort-start-lights')).toBeDisabled();
  });
});
