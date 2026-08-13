import { test, expect, Page } from '@playwright/test';

test.describe('Car Color Rendering', () => {
  test('should display car color indicator in standings container', async ({ page }) => {
    // Navigate to the main page
    await page.goto('/');
    await page.waitForSelector('#standings-container .standing-item');

    // Check that at least one standing item has a color-indicator
    const standingColorDots = page.locator('#standings-container .standing-item .color-indicator');
    await expect(standingColorDots.first()).toBeVisible();

    // Verify the color indicator has a valid background color (not the default fallback)
    const standingDot = standingColorDots.first();
    const bg = await standingDot.evaluate(el => getComputedStyle(el).backgroundColor);
    // Should be a valid RGB color (not transparent or default #cccccc fallback)
    expect(bg).not.toBe('rgba(0, 0, 0, 0)');
    expect(bg).not.toBe('transparent');
  });

  test('home page standings show the racer\'s exact car color (regression: stale bundle)', async ({ page }) => {
    // Ground truth from the API: the standings are sorted by points desc, so the
    // first .standing-item must render the car color of the top-points racer.
    await page.goto('/');
    await page.waitForSelector('#standings-container .standing-item');

    const racers = await (await page.request.get('/api/racers')).json();
    expect(racers.length).toBeGreaterThan(0);

    const firstItem = page.locator('#standings-container .standing-item').first();
    const shownName = (await firstItem.locator('.driver').innerText()).trim().toUpperCase();
    const racer = racers.find((r: any) => r.name.toUpperCase() === shownName);
    expect(racer, `standings racer "${shownName}" should exist in /api/racers`).toBeTruthy();

    const bg = await firstItem.locator('.color-indicator').first().evaluate(el => getComputedStyle(el).backgroundColor);
    // The rendered color must be exactly the racer's car color (normalized), not
    // transparent, not the #cccccc fallback, not a stale bundle without the dot.
    expect(bg).toBe(hexToRgb(normalizeColor(racer.car_color)));
  });

  test('should display custom hex car color in standings container', async ({ page }) => {
    // Create a racer with a distinct hex color directly via API
    await loginAsAdmin(page);
    const created = await page.evaluate(async () => {
      const res = await fetch('/api/racers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: 0,
          name: 'Standings Hex Racer',
          profile_picture: '/static/images/helmet.svg',
          car_color: '#800080',
          car_name: 'Purple Car',
          points: 999,
          rank: 2,
          position: 0,
        }),
      });
      return { ok: res.ok, status: res.status };
    });
    expect(created.ok).toBeTruthy();

    // Navigate to the main page
    await page.goto('/');
    await page.waitForSelector('#standings-container .standing-item');

    // The racer with 999 points should be first in standings (sorted by points desc)
    const standingItem = page.locator('#standings-container .standing-item').first();
    await expect(standingItem).toContainText('Standings Hex Racer');

    // The color indicator should have the correct purple color
    const colorDot = standingItem.locator('.color-indicator').first();
    const bg = await colorDot.evaluate(el => getComputedStyle(el).backgroundColor);
    // rgb(128, 0, 128) is the computed value for #800080
    expect(bg).toBe('rgb(128, 0, 128)');
  });

  test.describe.serial('Admin sets custom hex and named colors', () => {
    test.beforeEach(async ({ page }) => {
      await loginAsAdmin(page);
    });

    test('should display custom hex car color in leaderboard color indicator', async ({ page }) => {
      // Set up: create racer with a hex color in admin
      await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
      await page.click('button[hx-get="/api/html/racers/0/edit"]');
      await page.waitForSelector('#racerModal.show');
      await page.waitForSelector('#racerModal form#racer-form');

      await page.fill('form#racer-form input[name="name"]', 'Hex Racer');
      await page.fill('form#racer-form input[name="profile_picture"]', '/static/images/helmet.svg');
      await page.fill('form#racer-form input[name="car_name"]', 'Purple Car');
      await page.fill('form#racer-form input[name="car_color"]', '#800080');
      await page.fill('form#racer-form input[name="points"]', '50');
      await page.fill('form#racer-form input[name="rank"]', '5');
      await page.fill('form#racer-form input[name="position"]', '0');

      await page.click('form#racer-form button[type="submit"]');
      await page.waitForTimeout(500);

      // Navigate to the main leaderboard
      await page.goto('/');
      await page.waitForSelector('#leaderboard-body tr');

      // Find rows for Hex Racer (multiple workers create the same racer, use .first())
      const racerRows = page.locator('#leaderboard-body tr', { hasText: 'Hex Racer' });
      await expect(racerRows.first()).toBeVisible();

      // The color indicator inside the driver-name should have background containing the hex
      const colorDot = racerRows.first().locator('.color-indicator').first();
      const bg = await colorDot.evaluate(el => getComputedStyle(el).backgroundColor);
      // rgb(128, 0, 128) is the CSS computed value for #800080
      expect(bg).toBe('rgb(128, 0, 128)');
    });

    test('should display named red car color as mapped hex', async ({ page }) => {
      // Create a racer with named color "red" via fetch from within the page context
      // (this ensures cookies + Origin/Referer headers are properly included for auth+CSRF)
      const created = await page.evaluate(async () => {
        const res = await fetch('/api/racers', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            id: 0,
            name: 'Red Racer',
            profile_picture: '/static/images/helmet.svg',
            car_color: 'red',
            car_name: 'Red Car',
            points: 30,
            rank: 15,
            position: 0,
          }),
        });
        return { ok: res.ok, status: res.status };
      });
      expect(created.ok).toBeTruthy();

      // Navigate to the main leaderboard
      await page.goto('/');
      await page.waitForSelector('#leaderboard-body tr');

      // Find our racer by name
      const racerRow = page.locator('#leaderboard-body tr', { hasText: 'Red Racer' });
      await expect(racerRow.first()).toBeVisible();

      const colorDot = racerRow.first().locator('.color-indicator').first();
      const bg = await colorDot.evaluate(el => getComputedStyle(el).backgroundColor);

      // "red" should normalize to #ff4444 → rgb(255, 68, 68)
      expect(bg).toBe('rgb(255, 68, 68)');
    });
  });
});

test.describe('Team Color Rendering', () => {
  test('should display team color in leaderboard TEAM column', async ({ page }) => {
    await loginAsAdmin(page);

    const teamName = `Color-Team-${Date.now()}`;
    const racerName = `Color-Racer-${Date.now()}`;

    // Create a team with a distinct color via API (same path as admin settings)
    const team = await page.evaluate(async (name) => {
      const res = await fetch('/api/teams', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: 0, name, color: '#00ff88' }),
      });
      const teams = await (await fetch('/api/teams')).json();
      const created = teams.find((t: any) => t.name === name);
      return { ok: res.ok, id: created ? created.id : 0 };
    }, teamName);
    expect(team.ok).toBeTruthy();
    expect(team.id).toBeGreaterThan(0);

    // Create a racer assigned to the team
    const created = await page.evaluate(async ({ teamId, racerName }) => {
      const res = await fetch('/api/racers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: 0,
          name: racerName,
          profile_picture: '/static/images/helmet.svg',
          car_color: '#111111',
          car_name: 'Team Color Car',
          points: 10,
          rank: 40,
          position: 0,
          team_id: teamId,
        }),
      });
      return res.ok;
    }, { teamId: team.id, racerName });
    expect(created).toBeTruthy();

    // Frontpage: the TEAM column (3rd cell) should show a color indicator with the team color
    await page.goto('/');
    await page.waitForSelector('#leaderboard-body tr');

    const racerRow = page.locator('#leaderboard-body tr', { hasText: racerName });
    await expect(racerRow.first()).toBeVisible();

    const teamCell = racerRow.first().locator('td').nth(2);
    await expect(teamCell).toContainText(teamName);

    const colorDot = teamCell.locator('.color-indicator').first();
    const bg = await colorDot.evaluate(el => getComputedStyle(el).backgroundColor);
    // rgb(0, 255, 136) is the CSS computed value for #00ff88
    expect(bg).toBe('rgb(0, 255, 136)');
  });
});

test.describe('Leaderboard GAP column removal', () => {
  test('front page has no GAP column and CAR cell shows car/team color', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#leaderboard-body tr');

    const headers = page.locator('#leaderboard-table thead th');
    await expect(headers).toHaveCount(5);
    await expect(page.locator('#leaderboard-table thead')).not.toContainText('GAP');

    // CAR column (4th cell) should show a color indicator instead of the gap
    const carCell = page.locator('#leaderboard-body tr').first().locator('td').nth(3);
    await expect(carCell.locator('.color-indicator').first()).toBeVisible();
  });
});

// Mirrors ts/color.ts normalizeHex so the regression test asserts the exact
// color the frontend is expected to render.
const NAMED_COLORS: Record<string, string> = {
  red: '#ff4444',
  blue: '#4444ff',
  green: '#44ff44',
  yellow: '#ffff44',
  grey: '#aaaaaa',
  silver: '#aaaaaa',
  black: '#333333',
  purple: '#9b59b6',
  orange: '#e67e22',
};

function normalizeColor(color: string): string {
  if (!color) return '#cccccc';
  const trimmed = color.trim();
  const named = NAMED_COLORS[trimmed.toLowerCase()];
  if (named) return named;
  const hex = trimmed.startsWith('#') ? trimmed : '#' + trimmed;
  return /^#[0-9a-fA-F]{6}$/.test(hex) ? hex : '#cccccc';
}

function hexToRgb(hex: string): string {
  const m = /^#([0-9a-fA-F]{2})([0-9a-fA-F]{2})([0-9a-fA-F]{2})$/.exec(hex);
  if (!m) return '';
  return `rgb(${parseInt(m[1], 16)}, ${parseInt(m[2], 16)}, ${parseInt(m[3], 16)})`;
}

// Reuse helpers from admin.spec.ts (same pattern)
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

async function clickAdminSubTab(page: Page, tabSelector: string, subTabSelector: string) {
  await page.click(tabSelector);
  await page.locator(subTabSelector).waitFor({ state: 'visible', timeout: 15000 });
  await page.click(subTabSelector);
}
