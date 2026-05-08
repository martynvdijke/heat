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
    
    await page.click('th[data-sort="name"]');
    let nameHeader = page.locator('th[data-sort="name"] i');
    await expect(nameHeader).toHaveClass(/fa-sort-up|fa-sort-down/);

    await page.click('th[data-sort="points"]');
    let pointsHeader = page.locator('th[data-sort="points"] i');
    await expect(pointsHeader).toHaveClass(/fa-sort-up|fa-sort-down/);
  });

  test('should show driver stats modal on click', async ({ page }) => {
    await page.goto('/');
    
    if (await page.locator('#leaderboard-body tr').count() > 0) {
      await page.locator('#leaderboard-body tr').first().click();
      const modal = page.locator('#statsModal');
      await expect(modal).toHaveClass(/show/);
    }
  });

  test('compiled JS should not contain export keyword', async ({ page, request }) => {
    const resp = await request.get('/static/js/index.js');
    const body = await resp.text();
    expect(body).not.toContain('export');
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
});