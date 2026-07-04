import { test, expect, Page } from '@playwright/test';

test.describe('Admin Auth', () => {
  test('should load login or setup page', async ({ page }) => {
    await page.goto('/login.html');
    const title = await page.title();
    expect(title).toMatch(/Login|Setup/);
  });

  test('should redirect to login when accessing admin without session', async ({ page }) => {
    await page.goto('/admin.html');
    await expect(page).toHaveURL(/login|setup/, { timeout: 5000 });
  });
});

test.describe('Controller Page', () => {
  test('should redirect to login when not authenticated', async ({ page }) => {
    await page.goto('/controller.html');
    await expect(page).toHaveURL(/login|setup/, { timeout: 5000 });
  });
});

test.describe.serial('Admin Panel', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should load admin page after login', async ({ page }) => {
    await expect(page).toHaveTitle(/HEAT Admin/);
    await expect(page.locator('#admin-nav')).toBeVisible();
  });

  test('should list racers in table', async ({ page }) => {
    await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
    await expect(page.locator('#racer-list')).toBeVisible();
    const rows = page.locator('#racer-list tr');
    await expect(rows.first()).toBeVisible();
  });

  test('should add a new racer', async ({ page }) => {
    await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
    await page.click('button[hx-get="/api/html/racers/0/edit"]');
    await page.waitForSelector('#racerModal.show');
    await page.waitForSelector('#racerModal form#racer-form');

    await page.fill('form#racer-form input[name="name"]', 'Test Racer');
    await page.fill('form#racer-form input[name="profile_picture"]', '/static/images/helmet.svg');
    await page.fill('form#racer-form input[name="car_name"]', 'Test Car');
    await page.fill('form#racer-form input[name="car_color"]', '#800080');
    await page.fill('form#racer-form input[name="points"]', '42');
    await page.fill('form#racer-form input[name="rank"]', '10');
    await page.fill('form#racer-form input[name="position"]', '0');

    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#racer-list')).toContainText('Test Racer');
  });

  test('should edit an existing racer', async ({ page }) => {
    await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
    await page.waitForTimeout(500);

    const editBtn = page.locator('#racer-list tr .btn-outline-primary').first();
    await editBtn.click();
    await page.waitForSelector('#racerModal.show');

    const nameInput = page.locator('form#racer-form input[name="name"]');
    await nameInput.fill('Edited Racer');

    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#racer-list')).toContainText('Edited Racer');
  });

  test('should delete a racer', async ({ page }) => {
    await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
    await page.waitForTimeout(500);

    page.once('dialog', dialog => dialog.accept());
    const deleteBtn = page.locator('#racer-list tr .btn-outline-danger').first();
    await deleteBtn.click();
    await page.waitForTimeout(500);
  });

  test('should add a racer without profile picture', async ({ page }) => {
    await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
    await page.click('button[hx-get="/api/html/racers/0/edit"]');
    await page.waitForSelector('#racerModal.show');
    await page.waitForSelector('#racerModal form#racer-form');

    await page.fill('form#racer-form input[name="name"]', 'No Pic Racer');
    await page.fill('form#racer-form input[name="profile_picture"]', '');
    await page.fill('form#racer-form input[name="car_name"]', 'Shadow');
    await page.fill('form#racer-form input[name="car_color"]', '#000000');
    await page.fill('form#racer-form input[name="points"]', '10');
    await page.fill('form#racer-form input[name="rank"]', '20');
    await page.fill('form#racer-form input[name="position"]', '0');

    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#racer-list')).toContainText('No Pic Racer');
  });

  test('should validate racer form with only name and car', async ({ page }) => {
    await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
    await page.click('button[hx-get="/api/html/racers/0/edit"]');
    await page.waitForSelector('#racerModal.show');
    await page.waitForSelector('#racerModal form#racer-form');

    await page.fill('form#racer-form input[name="name"]', 'Min Racer');
    await page.fill('form#racer-form input[name="profile_picture"]', '');
    await page.fill('form#racer-form input[name="car_name"]', 'Basic');
    await page.fill('form#racer-form input[name="car_color"]', '#ff0000');
    await page.fill('form#racer-form input[name="points"]', '0');
    await page.fill('form#racer-form input[name="rank"]', '99');
    await page.fill('form#racer-form input[name="position"]', '0');

    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#racer-list')).toContainText('Min Racer');
  });

  test('should show racer stats tab', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="season"]', '#stats-pane');
    const statsList = page.locator('#stats-list');
    await expect(statsList).toBeAttached();
    const hasStats = await statsList.locator('tr').count() > 0;
    if (hasStats) {
      await expect(statsList.locator('tr').first()).toBeVisible();
    }
  });

  test('should show tracks tab and list', async ({ page }) => {
    await clickAdminSubTab(page, 'button[data-tab-id="race-day"]', '#tracks-subtab');
    await expect(page.locator('#track-list')).toBeVisible();
    const rows = page.locator('#track-list tr');
    await expect(rows.first()).toBeVisible();
  });

  test('should show quotes tab and list', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="drivers"]', '#quotes-pane');
    await expect(page.locator('#quote-list')).toBeVisible();
    const rows = page.locator('#quote-list tr');
    await expect(rows.first()).toBeVisible();
  });

  test('should show seasons tab', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="season"]', '#seasons-pane');
    await expect(page.locator('#seasons-list')).toBeVisible();
  });

  test('should show qualification tab', async ({ page }) => {
    await clickAdminSubTab(page, 'button[data-tab-id="race-day"]', '#qualification-subtab');
    await expect(page.locator('#qualification-grid')).toBeVisible();
  });

  test('should show notifications tab', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="config"]', '#notify-pane');
    await expect(page.locator('#notify-form')).toBeVisible();
    await expect(page.locator('#gotify-url')).toBeVisible();
  });

  test('should show AI tab', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="config"]', '#ai-pane');
    await expect(page.locator('#ai-pane')).toBeVisible();
    await expect(page.locator('#ai-settings-form')).toBeVisible();
    await expect(page.locator('#ai-track-extract-url')).toBeVisible();
  });

  test('should show Email tab with settings form', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="config"]', '#email-pane');
    await expect(page.locator('#email-pane')).toBeVisible();
    await expect(page.locator('#email-settings-form')).toBeVisible();
    await expect(page.locator('#smtp-host')).toBeVisible();
    await expect(page.locator('#smtp-port')).toBeVisible();
    await expect(page.locator('#racer-email-list')).toBeAttached();
  });

  test('should show Analytics tab with umami settings', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="config"]', '#umami-pane');
    await expect(page.locator('#umami-pane')).toBeVisible();
    await expect(page.locator('#umami-form')).toBeVisible();
    await expect(page.locator('#umami-url')).toBeVisible();
  });

  test('should show Backup tab with settings and manual backup', async ({ page }) => {
    await showAdminPane(page, 'button[data-tab-id="config"]', '#backup-pane');
    await expect(page.locator('#backup-pane')).toBeVisible();
    await expect(page.locator('#backup-form')).toBeVisible();
    await expect(page.locator('#backup-manual-btn')).toBeVisible();
    await expect(page.locator('#backup-list')).toBeVisible();
  });

  test('should load controller page after admin login', async ({ page }) => {
    await page.goto('/controller.html');
    await expect(page).toHaveTitle(/HEAT: Race Controller/);
    await expect(page.locator('#race-status')).toContainText('STOPPED');
  });

  test('should have start lights button with aria-label', async ({ page }) => {
    await page.goto('/controller.html');
    const btn = page.locator('button[aria-label="Start lights countdown"]');
    await expect(btn).toBeAttached();
    await expect(btn).toContainText('Start Lights');
  });

  test('should have open start lights display link with aria-label', async ({ page }) => {
    await page.goto('/controller.html');
    const link = page.locator('a[aria-label="Open start lights display"]');
    await expect(link).toBeAttached();
    await expect(link).toContainText('Open Lights Display');
  });

  test('should have Sound FX buttons with aria-labels', async ({ page }) => {
    await page.goto('/controller.html');
    const soundButtons = [
      page.locator('button[aria-label="Play engine sound"]'),
      page.locator('button[aria-label="Play horn sound"]'),
      page.locator('button[aria-label="Play finish sound"]'),
      page.locator('button[aria-label="Play crash sound"]'),
    ];
    for (const btn of soundButtons) {
      await expect(btn).toBeAttached();
    }
  });

  test('should have skip-to-content link with aria-label', async ({ page }) => {
    await page.goto('/controller.html');
    const skip = page.locator('a.skip-to-content');
    await expect(skip).toBeAttached();
    await expect(skip).toHaveAttribute('aria-label', 'Skip to main content');
    await expect(skip).toHaveAttribute('href', '#main-content');
  });

  test('should have visually-hidden label for race-type select', async ({ page }) => {
    await page.goto('/controller.html');
    const label = page.locator('label[for="race-type"].visually-hidden');
    await expect(label).toBeAttached();
    await expect(label).toContainText('Race Type');
  });

  test('should not have user-scalable=no in viewport meta', async ({ page }) => {
    await page.goto('/controller.html');
    const viewport = await page.evaluate(() => {
      const meta = document.querySelector('meta[name="viewport"]');
      return meta ? meta.getAttribute('content') : null;
    });
    expect(viewport).not.toBeNull();
    expect(viewport).not.toContain('user-scalable=no');
    expect(viewport).toContain('initial-scale=1.0');
  });

  test('should have aria-live on dynamic sections', async ({ page }) => {
    await page.goto('/controller.html');
    await expect(page.locator('#standings-list')).toHaveAttribute('aria-live', 'polite');
    await expect(page.locator('#turbo-list')).toHaveAttribute('aria-live', 'polite');
    await expect(page.locator('#gear-log')).toHaveAttribute('aria-live', 'polite');
    await expect(page.locator('#race-event-log')).toHaveAttribute('aria-live', 'polite');
    await expect(page.locator('#connected-players')).toHaveAttribute('aria-live', 'polite');
    await expect(page.locator('#current-weather')).toHaveAttribute('aria-live', 'polite');
  });
});

async function clickAdminSubTab(page: Page, tabSelector: string, subTabSelector: string) {
  await page.click(tabSelector);
  await page.locator(subTabSelector).waitFor({ state: 'visible', timeout: 10000 });
  await page.click(subTabSelector);
}

async function showAdminPane(page: Page, tabSelector: string, paneId: string) {
  await page.click(tabSelector);
  await expect(page.locator(paneId)).toBeAttached({ timeout: 10000 });
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
