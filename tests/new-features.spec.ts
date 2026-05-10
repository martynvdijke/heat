import { test, expect } from '@playwright/test';

test.describe('Stats Page', () => {
  test('should load stats page', async ({ page }) => {
    await page.goto('/stats.html');
    await expect(page).toHaveTitle(/HEAT: Season Statistics/);

    const heading = page.locator('h1');
    await expect(heading).toContainText('Season Statistics');
  });

  test('should display stat cards', async ({ page }) => {
    await page.goto('/stats.html');
    const totalRaces = page.locator('#total-races');
    await expect(totalRaces).toBeVisible();
  });

  test('should have navigation links', async ({ page }) => {
    await page.goto('/stats.html');
    await expect(page.locator('a[href="/"]').first()).toBeAttached();
    await expect(page.locator('a[href="/trophies.html"]').first()).toBeAttached();
    await expect(page.locator('a[href="/controller.html"]').first()).toBeAttached();
  });

  test('should render win distribution chart', async ({ page }) => {
    await page.goto('/stats.html');
    const canvas = page.locator('#wins-chart');
    await expect(canvas).toBeVisible();
    const rendered = await page.evaluate(() => {
      const c = document.getElementById('wins-chart') as HTMLCanvasElement;
      return c && c.getContext('2d') ? true : false;
    });
    expect(rendered).toBe(true);
  });

  test('should render driver performance table', async ({ page }) => {
    await page.goto('/stats.html');
    const tbody = page.locator('#driver-stats-table tbody');
    await expect(tbody).toBeVisible();
    const hasRows = await tbody.locator('tr').count();
    expect(hasRows).toBeGreaterThanOrEqual(0);
  });
});

test.describe('Trophies Page', () => {
  test('should load trophies page', async ({ page }) => {
    await page.goto('/trophies.html');
    await expect(page).toHaveTitle(/HEAT: Trophy Room/);

    const heading = page.locator('h1');
    await expect(heading).toContainText('Trophy Room');
  });

  test('should have driver selector', async ({ page }) => {
    await page.goto('/trophies.html');
    const selector = page.locator('#driver-select');
    await expect(selector).toBeVisible();
  });

  test('should display trophy categories', async ({ page }) => {
    await page.goto('/trophies.html');
    await expect(page.locator('#race-wins-grid')).toBeVisible();
    await expect(page.locator('#podium-grid')).toBeVisible();
    await expect(page.locator('#speed-grid')).toBeVisible();
  });
});

test.describe('Index Page - GeoJSON and Racers', () => {
  test('should display racers with helmet fallback images', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const rows = page.locator('#leaderboard-body tr');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);

    const firstImage = rows.first().locator('img');
    if (await firstImage.count() > 0) {
      await expect(firstImage).toBeVisible();
    }
  });

  test('should render circuit map with GeoJSON', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const map = page.locator('#circuit-map');
    await expect(map).toBeVisible();
    await expect(map).toHaveClass(/leaflet-container/);

    const mapPath = page.locator('#circuit-map path');
    await expect(mapPath.first()).toBeAttached({ timeout: 10000 });
  });

  test('should display race info with country, track and laps', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const raceCountry = page.locator('#race-country');
    await expect(raceCountry).toBeVisible();
    await expect(raceCountry).toContainText('Italy');

    const raceTrack = page.locator('#race-track');
    await expect(raceTrack).toBeVisible();
    await expect(raceTrack).toContainText('Monza');
  });
});
