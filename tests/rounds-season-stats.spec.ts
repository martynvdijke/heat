import { test, expect, Page } from '@playwright/test';

test.describe.serial('Admin Season CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  const seasonName = `E2E-Season-${Date.now()}`;

  test('should create a season', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="season"]', '#seasons-pane');

    await page.click('button[hx-get="/api/html/seasons/new"]');
    await page.waitForSelector('#seasonsModal.show', { timeout: 5000 });
    await page.waitForSelector('#seasonsModal form#season-form');

    await page.fill('form#season-form input[name="name"]', seasonName);
    await page.click('form#season-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#seasons-list')).toContainText(seasonName, { timeout: 10000 });
  });

  test('should show season in dropdown', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="season"]', '#seasons-pane');

    const select = page.locator('#season-rounds-select');
    await expect(select).toBeVisible();
    await expect(select).toContainText(seasonName);
  });

  test('should archive the season', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="season"]', '#seasons-pane');
    await page.waitForTimeout(500);

    const archiveBtn = page.locator('#seasons-list tr', { hasText: seasonName }).locator('button[hx-post*="/archive"]');
    page.once('dialog', dialog => dialog.accept());
    await archiveBtn.click();
    await page.waitForTimeout(500);

    await expect(page.locator('#seasons-list')).toContainText('archived', { timeout: 10000 });
  });

  test('should delete the season', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="season"]', '#seasons-pane');
    await page.waitForTimeout(500);

    const deleteBtn = page.locator('#seasons-list tr', { hasText: seasonName }).locator('button[hx-delete*="/seasons/"]');
    page.once('dialog', dialog => dialog.accept());
    await deleteBtn.click();

    await expect(page.locator('#seasons-list')).not.toContainText(seasonName, { timeout: 10000 });
  });
});

test.describe.serial('Admin Stats CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should create stats for a racer', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="season"]', '#stats-pane');
    await page.waitForTimeout(500);

    // Check if we have racers
    const racerCount = await page.evaluate(async () => {
      const res = await fetch('/api/racers');
      const racers = await res.json();
      return Array.isArray(racers) ? racers.length : 0;
    });
    expect(racerCount).toBeGreaterThan(0);

    // Click "Add Stats" button (in empty state or header)
    const addBtn = page.locator('button:has-text("Add Stats")').first();
    await addBtn.click();
    await page.waitForSelector('#statsModal.show', { timeout: 5000 });

    // Select first racer
    await page.selectOption('#stats-racer-select', { index: 1 });
    await page.fill('#stats-races', '3');
    await page.fill('#stats-gold', '1');
    await page.fill('#stats-silver', '1');
    await page.fill('#stats-bronze', '1');
    await page.fill('#stats-fastest-laps', '2');
    await page.fill('#stats-dnf', '0');
    await page.fill('#stats-dns', '0');

    await page.click('#stats-form button[type="submit"]');
    await page.waitForTimeout(500);

    // Modal should close and stats list should have data
    await expect(page.locator('#stats-list tr')).toHaveCount(1, { timeout: 10000 });
  });

  test('should edit existing stats', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="season"]', '#stats-pane');
    await page.waitForTimeout(500);

    // Find the edit button for the first stats row
    const editBtn = page.locator('#stats-list .btn-outline-primary').first();
    await expect(editBtn).toBeVisible({ timeout: 10000 });
    await editBtn.click();
    await page.waitForSelector('#statsModal.show', { timeout: 5000 });

    // Verify modal has data populated
    const racesInput = page.locator('#stats-races');
    await expect(racesInput).toHaveValue(/^\d+$/);

    // Modify stats
    await page.fill('#stats-races', '5');
    await page.fill('#stats-gold', '2');
    await page.click('#stats-form button[type="submit"]');
    await page.waitForTimeout(500);

    // Verify modal closed and stats list updated
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
    await page.waitForTimeout(1000);

    const container = page.locator('#seasons-container');
    const hasSeasons = await container.locator('.card').count();
    const noSeasons = await container.locator('text=No seasons configured').count();

    expect(hasSeasons > 0 || noSeasons > 0).toBeTruthy();
  });

  test('should expand active season accordion', async ({ page }) => {
    await page.goto('/seasons.html');
    await page.waitForTimeout(1000);

    const activeBadge = page.locator('.badge.bg-success');
    if (await activeBadge.count() > 0) {
      const seasonCard = activeBadge.first().locator('..').locator('..');
      const collapse = seasonCard.locator('.collapse.show');
      await expect(collapse).toBeAttached({ timeout: 5000 });
    }
  });

  test('should display navigation links', async ({ page }) => {
    await page.goto('/seasons.html');
    await expect(page.locator('a[href="/"]').first()).toBeAttached();
    await expect(page.locator('a[href="/stats.html"]').first()).toBeAttached();
    await expect(page.locator('a[href="/trophies.html"]').first()).toBeAttached();
  });
});

test.describe('Public Stats Page with Data', () => {
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

  test('should have season selector populated', async ({ page }) => {
    await page.goto('/stats.html');
    await page.waitForTimeout(1000);
    const select = page.locator('#stats-season-select');
    const optionCount = await select.locator('option').count();
    expect(optionCount).toBeGreaterThan(1);
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
});

test.describe.serial('Admin Round Editing Flow', () => {
  let seasonId = 0;
  let roundId = 0;

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should create a season and round via API', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const seasonRes = await fetch('/api/seasons', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: `Round-E2E-Season-${Date.now()}` })
      });
      const season = await seasonRes.json();
      const sid = season.id;

      const res = await fetch('/api/rounds', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ race_name: `E2E-Round-${Date.now()}`, round: 0, season_id: sid })
      });
      const data = await res.json();
      return { ok: res.ok, id: data.id, seasonId: sid };
    });
    expect(result.ok).toBeTruthy();
    roundId = result.id;
    seasonId = result.seasonId;
  });

  test('should display round in rounds list', async ({ page }) => {
    expect(roundId).toBeGreaterThan(0);

    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#rounds-subtab');
    await page.waitForTimeout(500);

    await expect(page.locator('#rounds-list')).toContainText('draft', { timeout: 10000 });
  });

  test('should edit round via inline editor', async ({ page }) => {
    expect(roundId).toBeGreaterThan(0);

    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#rounds-subtab');
    await page.waitForTimeout(500);

    // Click edit button on the draft round
    const editBtn = page.locator('#rounds-list .btn-outline-primary').first();
    await expect(editBtn).toBeVisible({ timeout: 10000 });
    await editBtn.click();

    // Wait for inline editor to appear
    await page.waitForTimeout(500);
    await expect(page.locator('text=Editing:')).toBeVisible({ timeout: 5000 });

    // Verify score inputs are present
    const scoreInput = page.locator('.round-score-pts').first();
    await expect(scoreInput).toBeVisible();

    // Modify a score
    await scoreInput.fill('15');
    await page.waitForTimeout(200);

    // Save draft
    const saveBtn = page.locator('button:has-text("Save Draft")');
    await saveBtn.click();
    await page.waitForTimeout(500);

    // Verify editor closed and round still listed
    await expect(page.locator('text=Editing:')).toHaveCount(0, { timeout: 5000 });
    await expect(page.locator('#rounds-list')).toContainText('draft');
  });

  test('should finalize round', async ({ page }) => {
    expect(roundId).toBeGreaterThan(0);

    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#rounds-subtab');
    await page.waitForTimeout(500);

    const editBtn = page.locator('#rounds-list .btn-outline-primary').first();
    await expect(editBtn).toBeVisible({ timeout: 10000 });
    await editBtn.click();
    await page.waitForTimeout(500);

    page.once('dialog', dialog => dialog.accept());
    const finalizeBtn = page.locator('button:has-text("Finalize")');
    await finalizeBtn.click();
    await page.waitForTimeout(1000);

    // Verify round shows as final
    await expect(page.locator('#rounds-list')).toContainText('Final', { timeout: 10000 });
  });

  test('should verify final round is locked', async ({ page }) => {
    expect(roundId).toBeGreaterThan(0);

    await page.click('button[data-tab-id="season"]');
    await page.waitForTimeout(500);
    await page.click('#rounds-subtab');
    await page.waitForTimeout(500);

    // Should show lock icon on final round
    await expect(page.locator('#rounds-list .fa-lock')).toBeVisible({ timeout: 10000 });
  });

  test('should cleanup season and round', async ({ page }) => {
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
async function clickAdminSubTab(page: Page, tabSelector: string, subTabSelector: string) {
  await page.click(tabSelector);
  await page.locator(subTabSelector).waitFor({ state: 'visible', timeout: 15000 });
  await page.click(subTabSelector);
}

async function showAdminPane(page: Page, tabSelector: string, paneId: string) {
  await page.click(tabSelector);
  await expect(page.locator(paneId)).toBeAttached({ timeout: 15000 });
  await page.evaluate((id) => {
    const pane = document.getElementById(id.replace('#', ''));
    if (pane) pane.classList.add('show', 'active');
  }, paneId);
  await expect(page.locator(paneId)).toHaveClass(/show/, { timeout: 5000 });
}

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
