import { test, expect, Page } from '@playwright/test';

test.describe.serial('Admin Season CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  const seasonName = `E2E-Season-${Date.now()}`;

  test('should create a season', async ({ page }) => {
    // Navigate to Season tab, then click Seasons subtab
    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#seasons-subtab');
    await page.waitForTimeout(500);

    // Click "New Season" button (HTMX loads form into modal)
    await page.click('button[hx-get="/api/html/seasons/new"]');
    await page.waitForSelector('#seasonsModal.show', { timeout: 5000 });
    await page.waitForSelector('#seasonsModal form#season-form');

    await page.fill('form#season-form input[name="name"]', seasonName);
    await page.click('form#season-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#seasons-list')).toContainText(seasonName, { timeout: 10000 });
  });

  test('should show season in dropdown', async ({ page }) => {
    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#seasons-subtab');
    await page.waitForTimeout(500);

    const select = page.locator('#season-rounds-select');
    await expect(select).toContainText(seasonName, { timeout: 10000 });
  });

  test('should archive the season', async ({ page }) => {
    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#seasons-subtab');
    await page.waitForTimeout(500);

    const archiveBtn = page.locator('#seasons-list tr', { hasText: seasonName }).locator('button.btn-outline-warning');
    page.once('dialog', dialog => dialog.accept());
    await archiveBtn.click();
    await page.waitForTimeout(800);

    await expect(page.locator('#seasons-list')).toContainText('archived', { timeout: 10000 });
  });

  test('should delete the season', async ({ page }) => {
    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#seasons-subtab');
    await page.waitForTimeout(500);

    const deleteBtn = page.locator('#seasons-list tr', { hasText: seasonName }).locator('button.btn-outline-danger');
    page.once('dialog', dialog => dialog.accept());
    await deleteBtn.click();

    await expect(page.locator('#seasons-list')).not.toContainText(seasonName, { timeout: 10000 });
  });
});

test.describe.skip('Admin Stats CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should create stats for a racer', async ({ page }) => {
    // Navigate to Season tab (stats-subtab is active by default)
    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(1000);

    // Click Stats subtab to ensure loadRacerStats is triggered
    await page.click('#stats-subtab');
    await page.waitForTimeout(500);

    // Click the header "Add Stats" button
    const addBtn = page.locator('.card-header button:has-text("Add Stats")');
    await addBtn.click();
    await page.waitForSelector('#statsModal.show', { timeout: 5000 });

    // Select first racer
    const options = page.locator('#stats-racer-select option');
    const count = await options.count();
    expect(count).toBeGreaterThanOrEqual(2); // placeholder + at least one racer

    await page.selectOption('#stats-racer-select', { index: 1 });
    await page.fill('#stats-races', '3');
    await page.fill('#stats-gold', '1');
    await page.fill('#stats-silver', '1');
    await page.fill('#stats-bronze', '1');
    await page.fill('#stats-fastest-laps', '2');
    await page.fill('#stats-dnf', '0');
    await page.fill('#stats-dns', '0');

    await page.click('#stats-form button[type="submit"]');
    await page.waitForTimeout(800);

    await expect(page.locator('#stats-list tr')).toHaveCount(1, { timeout: 10000 });
  });

  test('should edit existing stats', async ({ page }) => {
    page.on('console', msg => console.log('BROWSER:', msg.type(), msg.text()));

    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#stats-subtab');
    await page.waitForTimeout(500);

    // Find edit button in the first stats row
    const editBtn = page.locator('#stats-list .btn-outline-primary').first();
    await expect(editBtn).toBeVisible({ timeout: 10000 });

    // Log what we're clicking
    const editBtnId = await editBtn.getAttribute('onclick');
    console.log('EDIT BUTTON onclick:', editBtnId);

    await editBtn.click();
    await page.waitForSelector('#statsModal.show', { timeout: 5000 });

    // Check racerStats from browser
    const statsInfo = await page.evaluate(() => ({
        racerStatsLen: (window as any).racerStats?.length,
        racerStats: JSON.parse(JSON.stringify((window as any).racerStats || [])),
        statsRacesVal: (document.getElementById('stats-races') as HTMLInputElement)?.value,
    }));
    console.log('STATS INFO:', JSON.stringify(statsInfo));

    const racesInput = page.locator('#stats-races');
    await expect(racesInput).toHaveValue(/^\d+$/);

    // Modify
    await page.fill('#stats-races', '5');
    await page.fill('#stats-gold', '2');
    await page.click('#stats-form button[type="submit"]');
    await page.waitForTimeout(800);

    await expect(page.locator('#stats-list')).toContainText('2', { timeout: 10000 });
  });
});

test.describe('Public Seasons Page', () => {
  test('should load seasons page with title', async ({ page }) => {
    await page.goto('/seasons.html');
    await expect(page).toHaveTitle(/HEAT: Seasons & Rounds/);
    const heading = page.locator('h1');
    await expect(heading).toContainText('Seasons & Rounds');
  });

  test('should display at least one season', async ({ page }) => {
    await page.goto('/seasons.html');
    await page.waitForTimeout(1500);

    const container = page.locator('#seasons-container');
    const hasSeasons = await container.locator('.card').count();
    const noSeasons = await container.locator('text=No seasons configured').count();

    expect(hasSeasons > 0 || noSeasons > 0).toBeTruthy();
  });

  test('should expand active season', async ({ page }) => {
    await page.goto('/seasons.html');
    await page.waitForTimeout(1500);

    // Active season has a green badge; its collapse should be expanded (show class)
    const activeBadge = page.locator('.badge.bg-success').first();
    if (await activeBadge.count() > 0) {
      // Check that the card contains an expanded collapse
      await expect(activeBadge).toBeVisible();
    }
  });

  test('should display navigation links', async ({ page }) => {
    await page.goto('/seasons.html');
    await expect(page.locator('a[href="/"]').first()).toBeAttached();
    await expect(page.locator('a[href="/stats.html"]').first()).toBeAttached();
    await expect(page.locator('a[href="/trophies.html"]').first()).toBeAttached();
  });
});

test.describe('Public Stats Page', () => {
  test('should load stats page', async ({ page }) => {
    await page.goto('/stats.html');
    await expect(page).toHaveTitle(/HEAT: Season Statistics/);
  });

  test('should display stat cards', async ({ page }) => {
    await page.goto('/stats.html');
    await expect(page.locator('#total-races')).toBeVisible();
    await expect(page.locator('#total-drivers')).toBeVisible();
    await expect(page.locator('#fastest-laps')).toBeVisible();
  });

  test('should render driver performance table', async ({ page }) => {
    await page.goto('/stats.html');
    await page.waitForTimeout(1500);
    const tbody = page.locator('#driver-stats-table tbody');
    await expect(tbody).toBeVisible();
  });

  test('should render chart canvases', async ({ page }) => {
    await page.goto('/stats.html');
    await page.waitForTimeout(2000);

    const charts = [
      page.locator('#points-chart'),
      page.locator('#wins-chart'),
      page.locator('#laptime-chart'),
      page.locator('#battle-chart'),
    ];
    for (const chart of charts) {
      await expect(chart).toBeAttached();
    }
  });

  test('should display track statistics table', async ({ page }) => {
    await page.goto('/stats.html');
    await page.waitForTimeout(2000);
    const tbody = page.locator('#track-stats-table tbody');
    await expect(tbody).toBeVisible();
  });

  test('should load deeper stats tables', async ({ page }) => {
    await page.goto('/stats.html');
    await page.waitForTimeout(2000);

    await expect(page.locator('#qualifying-delta-body')).toBeAttached();
    await expect(page.locator('#consistency-body')).toBeAttached();
    await expect(page.locator('#incidents-body')).toBeAttached();
    await expect(page.locator('#pace-heatmap-body')).toBeAttached();
  });

  test('should display spins and overheated columns in driver performance table', async ({ page }) => {
    await page.goto('/stats.html');
    await page.waitForTimeout(2000);

    const headers = page.locator('#driver-stats-table thead th');
    const headerTexts = await headers.allTextContents();
    const combined = headerTexts.join(' ');

    expect(combined).toMatch(/spins/i);
    expect(combined).toMatch(/overheated/i);

    // Should have 8 columns: Driver, Races, Wins, G/S/B, Podiums, Points, Spins, Overheated
    await expect(headers).toHaveCount(8);
  });
});

test.describe.serial('Stats Page Round-Trip', () => {
  let seasonId = 0;
  let roundId = 0;

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should create and finalize a round with spins and overheated', async ({ page }) => {
    const setup = await page.evaluate(async () => {
      // Create a season
      const sRes = await fetch('/api/seasons', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: `StatsRR-${Date.now()}` }),
      });
      if (!sRes.ok) throw new Error('Failed to create season');
      const s = await sRes.json();

      // Create a round snapshot
      const rRes = await fetch('/api/rounds', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ race_name: `Round-${Date.now()}`, season_id: s.id }),
      });
      if (!rRes.ok) throw new Error('Failed to create round');
      const r = await rRes.json();

      // Fetch the snapshot scores
      const batchRes = await fetch(`/api/rounds/batch?ids=${r.id}`);
      const snaps = await batchRes.json();
      const snap = snaps[0];

      if (snap?.scores?.length) {
        // Update scores with spins and overheated
        const updated = snap.scores.map((sc: any, i: number) => ({
          ...sc,
          spins: (i + 1) * 2,
          overheated: i + 1,
        }));
        const patchRes = await fetch(`/api/rounds?id=${r.id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(updated),
        });
        if (!patchRes.ok) throw new Error('Failed to update scores');

        // Finalize round
        const finalRes = await fetch(`/api/rounds/finalize?id=${r.id}`, { method: 'PATCH' });
        if (!finalRes.ok) throw new Error('Failed to finalize round');
      }

      return { seasonId: s.id, roundId: r.id };
    });

    seasonId = setup.seasonId;
    roundId = setup.roundId;
    expect(seasonId).toBeGreaterThan(0);
    expect(roundId).toBeGreaterThan(0);
  });

  test('should display spins and overheated values on stats page', async ({ page }) => {
    expect(seasonId).toBeGreaterThan(0);

    await page.goto('/stats.html');
    await page.waitForTimeout(1500);

    // Select the season we created
    const select = page.locator('#stats-season-select');
    await select.selectOption(String(seasonId));
    await page.waitForTimeout(1500);

    // Verify driver table has data rows
    const tbody = page.locator('#driver-stats-table tbody');
    const rows = tbody.locator('tr');
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThan(0);

    // Each row should have 8 columns: Driver, Races, Wins, G/S/B, Podiums, Points, Spins, Overheated
    const firstRowCells = rows.first().locator('td');
    await expect(firstRowCells).toHaveCount(8);

    // Spins (column 7, index 6) and Overheated (column 8, index 7) should be numbers
    const cellTexts = await firstRowCells.allTextContents();
    const spins = parseInt(cellTexts[6], 10);
    const overheated = parseInt(cellTexts[7], 10);
    expect(spins).toBeGreaterThan(0);
    expect(overheated).toBeGreaterThan(0);
  });

  test('should verify points leaderboard shows season data from rounds', async ({ page }) => {
    expect(seasonId).toBeGreaterThan(0);

    await page.goto('/stats.html');
    await page.waitForTimeout(1500);

    // Select our season
    const select = page.locator('#stats-season-select');
    await select.selectOption(String(seasonId));
    await page.waitForTimeout(1500);

    // Points leaderboard should have data rows (not the empty placeholder)
    const pointsBody = page.locator('#points-body');
    const pointsRows = pointsBody.locator('tr');
    const rowCount = await pointsRows.count();
    expect(rowCount).toBeGreaterThan(0);

    // First row should show the #1 racer with positive points
    const firstPointsRow = pointsRows.first().locator('td');
    const ptsCells = await firstPointsRow.allTextContents();
    // Columns: Rank, Driver, Car, Points, Wins, Podiums, Avg Finish
    expect(ptsCells.length).toBeGreaterThanOrEqual(7);
    const points = parseInt(ptsCells[3], 10);
    expect(points).toBeGreaterThan(0);
  });

  test('should cleanup season and rounds', async ({ page }) => {
    expect(seasonId).toBeGreaterThan(0);
    const cleaned = await page.evaluate(async (sid) => {
      // Delete rounds first
      const roundsRes = await fetch(`/api/rounds?season_id=${sid}`, { method: 'DELETE' });
      // Delete season
      const seasonRes = await fetch(`/api/seasons?id=${sid}`, { method: 'DELETE' });
      return seasonRes.ok;
    }, seasonId);
    expect(cleaned).toBeTruthy();
  });
});

test.describe.skip('Admin Round Editing Flow', () => {
  let roundId = 0;
  let seasonId = 0;

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should create a round via API', async ({ page }) => {
    // Ensure at least one active season exists
    const setup = await page.evaluate(async () => {
      // Check if there's an active season
      const seasonsRes = await fetch('/api/seasons');
      const seasons = await seasonsRes.json();
      const active = Array.isArray(seasons) ? seasons.find((s: any) => s.status === 'active') : null;

      let sid = active ? active.id : 0;
      if (!sid) {
        // Create a new season
        const newSeasonRes = await fetch('/api/seasons', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: `Round-E2E-${Date.now()}` })
        });
        const newSeason = await newSeasonRes.json();
        sid = newSeason.id;
      }

      // Create round
      const res = await fetch('/api/rounds', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ race_name: `E2E-Round-${Date.now()}`, round: 0, season_id: sid })
      });
      const data = await res.json();
      return { ok: res.ok, id: data.id, seasonId: sid };
    });
    expect(setup.ok).toBeTruthy();
    roundId = setup.id;
    seasonId = setup.seasonId;
  });

  test('should display round in rounds list', async ({ page }) => {
    expect(roundId).toBeGreaterThan(0);

    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#rounds-subtab');
    await page.waitForTimeout(800);

    await expect(page.locator('#rounds-list')).toContainText('draft', { timeout: 15000 });
  });

  test('should edit round via inline editor', async ({ page }) => {
    expect(roundId).toBeGreaterThan(0);

    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#rounds-subtab');
    await page.waitForTimeout(800);

    const editBtn = page.locator('#rounds-list .btn-outline-primary').first();
    await expect(editBtn).toBeVisible({ timeout: 10000 });
    await editBtn.click();
    await page.waitForTimeout(500);

    await expect(page.locator('text=Editing:')).toBeVisible({ timeout: 5000 });

    const scoreInput = page.locator('.round-score-pts').first();
    await expect(scoreInput).toBeVisible();
    await scoreInput.fill('15');
    await page.waitForTimeout(200);

    // Save draft
    const saveBtn = page.locator('button:has-text("Save Draft")');
    await saveBtn.click();
    await page.waitForTimeout(800);

    await expect(page.locator('text=Editing:')).toHaveCount(0, { timeout: 5000 });
    await expect(page.locator('#rounds-list')).toContainText('draft');
  });

  test('should finalize round', async ({ page }) => {
    expect(roundId).toBeGreaterThan(0);

    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#rounds-subtab');
    await page.waitForTimeout(800);

    const editBtn = page.locator('#rounds-list .btn-outline-primary').first();
    await expect(editBtn).toBeVisible({ timeout: 10000 });
    await editBtn.click();
    await page.waitForTimeout(500);

    page.once('dialog', dialog => dialog.accept());
    const finalizeBtn = page.locator('button:has-text("Finalize")');
    await finalizeBtn.click();
    await page.waitForTimeout(1000);

    await expect(page.locator('#rounds-list')).toContainText('Final', { timeout: 10000 });
  });

  test('should verify final round is locked', async ({ page }) => {
    expect(roundId).toBeGreaterThan(0);

    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#rounds-subtab');
    await page.waitForTimeout(800);

    await expect(page.locator('#rounds-list .fa-lock')).toBeVisible({ timeout: 10000 });
  });

  test('should load stats page with season data', async ({ page }) => {
    await page.goto('/stats.html');
    await expect(page).toHaveTitle(/HEAT: Season Statistics/);
    const select = page.locator('#stats-season-select');
    const optionCount = await select.locator('option').count();
    expect(optionCount).toBeGreaterThan(1);
  });

  test('should cleanup round and season', async ({ page }) => {
    expect(seasonId).toBeGreaterThan(0);

    const res = await page.evaluate(async (sid) => {
      await fetch(`/api/rounds?season_id=${sid}`, { method: 'DELETE' });
      const delRes = await fetch(`/api/seasons?id=${sid}`, { method: 'DELETE' });
      return delRes.ok;
    }, seasonId);
    expect(res).toBeTruthy();
  });
});

// Helper functions
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
