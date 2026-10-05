import { test, expect } from '@playwright/test';

test.describe('Index Page', () => {
  test('should load the index page without critical console errors', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') consoleErrors.push(msg.text());
    });

    await page.goto('/');

    await expect(page).toHaveTitle(/HEAT: Pedal to the Metal/);

    const map = page.locator('#circuit-map');
    await expect(map).toHaveClass(/leaflet-container/);

    const quote = page.locator('#random-quote');
    await expect(quote).not.toContainText('Loading commentary...');

    const criticalErrors = consoleErrors.filter(e => 
      !e.includes('404') && !e.includes('500') && !e.includes('Failed to load')
    );
    expect(criticalErrors).toEqual([]);
  });

  test('should display leaderboard with racers', async ({ page }) => {
    await page.goto('/');
    const rows = page.locator('#leaderboard-body tr');
    await expect(rows.first()).toBeVisible();
  });

  test('should display version in footer', async ({ page }) => {
    await page.goto('/');
    const versionElement = page.locator('#version-display');
    await expect(versionElement).toBeVisible();
    await expect(versionElement).toHaveText(/v\d+\.\d+\.\d+/);
  });

  test('should display standings container', async ({ page }) => {
    await page.goto('/');
    const standings = page.locator('#standings-container');
    await expect(standings).toBeVisible();
  });

  test('should display race info', async ({ page }) => {
    await page.goto('/');
    const raceCountry = page.locator('#race-country');
    await expect(raceCountry).toBeVisible();
    const track = page.locator('#race-track');
    await expect(track).toBeVisible();
    const laps = page.locator('#race-laps');
    await expect(laps).toBeVisible();
  });

  test('should display stats cards', async ({ page }) => {
    await page.goto('/');
    const totalSeasons = page.locator('#total-seasons');
    await expect(totalSeasons).toBeVisible();
    const totalDrivers = page.locator('#total-drivers');
    await expect(totalDrivers).toBeVisible();
    const totalTracks = page.locator('#total-tracks');
    await expect(totalTracks).toBeVisible();
  });

  test('should have link to admin in navigation', async ({ page }) => {
    await page.goto('/');
    const adminLink = page.locator('a[href="/login.html"]');
    await expect(adminLink.first()).toBeAttached();
  });

  test('should have link to API docs in navigation', async ({ page }) => {
    await page.goto('/');
    const docsLink = page.locator('a[href="/docs"]');
    await expect(docsLink.first()).toBeAttached();
  });

  test('should display circuit map', async ({ page }) => {
    await page.goto('/');
    const map = page.locator('#circuit-map');
    await expect(map).toBeVisible();
  });

  test('should sort leaderboard by different columns', async ({ page }) => {
    await page.goto('/');
    
    // force: true needed for mobile-chrome where Playwright's actionability
    // check fails on mobile emulation (isMobile: true) due to viewport mismatch
    await page.click('th[data-sort="name"]', { force: true });
    let nameHeader = page.locator('th[data-sort="name"] i');
    await expect(nameHeader).toHaveClass(/fa-sort-up|fa-sort-down/);

    await page.click('th[data-sort="points"]', { force: true });
    let pointsHeader = page.locator('th[data-sort="points"] i');
    await expect(pointsHeader).toHaveClass(/fa-sort-up|fa-sort-down/);
  });

  test('should show driver stats modal on click', async ({ page }) => {
    await page.goto('/');
    
    if (await page.locator('#leaderboard-body tr').count() > 0) {
      // force: true needed for mobile-chrome (same reason as sort test)
      await page.locator('#leaderboard-body tr').first().click({ force: true });
      const modal = page.locator('#statsModal');
      await expect(modal).toHaveClass(/show/);
    }
  });

  test('compiled JS should not contain export statement', async ({ page, request }) => {
    const resp = await request.get('/static/js/index.js');
    const body = await resp.text();
    expect(body).not.toMatch(/\bexport\b/);
  });

  test('avatar-sm CSS class should constrain standings images', async ({ page }) => {
    await page.goto('/');
    const hasAvatarSmStyle = await page.evaluate(() => {
      for (const sheet of document.styleSheets) {
        try {
          for (const rule of sheet.cssRules) {
            if (rule instanceof CSSStyleRule && rule.selectorText.includes('avatar-sm')) {
              return rule.style.width === '32px' && rule.style.height === '32px';
            }
          }
        } catch (e) { /* cross-origin stylesheets may throw */ }
      }
      const testEl = document.createElement('div');
      testEl.className = 'avatar-sm';
      document.body.appendChild(testEl);
      const computed = getComputedStyle(testEl);
      document.body.removeChild(testEl);
      return computed.width === '32px' && computed.height === '32px';
    });
    expect(hasAvatarSmStyle).toBe(true);
  });

  test('standings container should not be oversized', async ({ page }) => {
    await page.goto('/');
    const box = await page.locator('#standings-container').boundingBox();
    expect(box).not.toBeNull();
    if (box) {
      expect(box.width).toBeLessThan(600);
    }
  });

  test.describe('upcoming race badge', () => {
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

    async function setNextRaceDate(
      page: import('@playwright/test').Page,
      info: Record<string, unknown>,
      nextRaceDate: string
    ): Promise<boolean> {
      // Use the browser's own fetch so the session cookie + Origin header are sent
      // (page.request does not reliably carry the session cookie on all browsers).
      return page.evaluate(async (body) => {
        const res = await fetch('/api/race-info', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body)
        });
        return res.ok;
      }, { ...info, next_race_date: nextRaceDate });
    }

    test('should show the upcoming race date and countdown', async ({ page }) => {
      await loginAsAdmin(page);
      const info = await (await page.request.get('/api/race-info')).json();
      const original = info.next_race_date || '';
      try {
        expect(await setNextRaceDate(page, info, '2099-01-01')).toBeTruthy();
        await page.goto('/');
        const badge = page.locator('#next-race-badge');
        await expect(badge).toBeVisible();
        await expect(badge).toContainText('2099-01-01');
        await expect(badge).toContainText(/in \d+ days/);
      } finally {
        await setNextRaceDate(page, info, original);
      }
    });

    test('should hide the upcoming race badge when no date is set', async ({ page }) => {
      await loginAsAdmin(page);
      const info = await (await page.request.get('/api/race-info')).json();
      try {
        expect(await setNextRaceDate(page, info, '')).toBeTruthy();
        await page.goto('/');
        await expect(page.locator('#next-race-badge')).toBeHidden();
      } finally {
        await setNextRaceDate(page, info, info.next_race_date || '');
      }
    });
  });
});