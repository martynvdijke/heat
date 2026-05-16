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
    await expect(page.locator('#adminTabs')).toBeVisible();
  });

  test('should list racers in table', async ({ page }) => {
    await page.click('#racers-tab');
    await expect(page.locator('#racer-list')).toBeVisible();
    const rows = page.locator('#racer-list tr');
    await expect(rows.first()).toBeVisible();
  });

  test('should add a new racer', async ({ page }) => {
    await page.click('#racers-tab');
    await page.click('[onclick="openRacerModal()"]');
    await page.waitForSelector('#racerModal.show');

    await page.fill('form#racer-form input[name="name"]', 'Test Racer');
    await page.fill('form#racer-form input[name="profile_picture"]', '/static/images/helmet.svg');
    await page.fill('form#racer-form input[name="car_name"]', 'Test Car');
    await page.fill('form#racer-form input[name="car_color_text"]', '#800080');
    await page.fill('form#racer-form input[name="points"]', '42');
    await page.fill('form#racer-form input[name="rank"]', '10');
    await page.fill('form#racer-form input[name="position"]', '0');

    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#racer-list')).toContainText('Test Racer');
  });

  test('should edit an existing racer', async ({ page }) => {
    await page.click('#racers-tab');
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
    await page.click('#racers-tab');
    await page.waitForTimeout(500);

    page.once('dialog', dialog => dialog.accept());
    const deleteBtn = page.locator('#racer-list tr .btn-outline-danger').first();
    await deleteBtn.click();
    await page.waitForTimeout(500);
  });

  test('should add a racer without profile picture', async ({ page }) => {
    await page.click('#racers-tab');
    await page.click('[onclick="openRacerModal()"]');
    await page.waitForSelector('#racerModal.show');

    await page.fill('form#racer-form input[name="name"]', 'No Pic Racer');
    await page.fill('form#racer-form input[name="profile_picture"]', '');
    await page.fill('form#racer-form input[name="car_name"]', 'Shadow');
    await page.fill('form#racer-form input[name="car_color_text"]', '#000000');
    await page.fill('form#racer-form input[name="points"]', '10');
    await page.fill('form#racer-form input[name="rank"]', '20');
    await page.fill('form#racer-form input[name="position"]', '0');

    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#racer-list')).toContainText('No Pic Racer');
  });

  test('should validate racer form with only name and car', async ({ page }) => {
    await page.click('#racers-tab');
    await page.click('[onclick="openRacerModal()"]');
    await page.waitForSelector('#racerModal.show');

    await page.fill('form#racer-form input[name="name"]', 'Min Racer');
    await page.fill('form#racer-form input[name="profile_picture"]', '');
    await page.fill('form#racer-form input[name="car_name"]', 'Basic');
    await page.fill('form#racer-form input[name="car_color_text"]', '#ff0000');
    await page.fill('form#racer-form input[name="points"]', '0');
    await page.fill('form#racer-form input[name="rank"]', '99');
    await page.fill('form#racer-form input[name="position"]', '0');

    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#racer-list')).toContainText('Min Racer');
  });

  test('should show racer stats tab', async ({ page }) => {
    await page.click('#stats-tab');
    await expect(page.locator('#stats-tab')).toHaveAttribute('aria-selected', 'true');
    const statsList = page.locator('#stats-list');
    await expect(statsList).toBeAttached();
    const hasStats = await statsList.locator('tr').count() > 0;
    if (hasStats) {
      await expect(statsList.locator('tr').first()).toBeVisible();
    }
  });

  test('should show tracks tab and list', async ({ page }) => {
    await page.click('#tracks-tab');
    await expect(page.locator('#track-list')).toBeVisible();
    const rows = page.locator('#track-list tr');
    await expect(rows.first()).toBeVisible();
  });

  test('should show quotes tab and list', async ({ page }) => {
    await page.click('#quotes-tab');
    await expect(page.locator('#quote-list')).toBeVisible();
    const rows = page.locator('#quote-list tr');
    await expect(rows.first()).toBeVisible();
  });

  test('should show seasons tab', async ({ page }) => {
    await page.click('#seasons-tab');
    await expect(page.locator('#seasons-list')).toBeVisible();
  });

  test('should show qualification tab', async ({ page }) => {
    await page.click('#qualification-tab');
    await expect(page.locator('#qualification-grid')).toBeVisible();
  });

  test('should show notifications tab', async ({ page }) => {
    await page.click('#notify-tab');
    await expect(page.locator('#notify-form')).toBeVisible();
    await expect(page.locator('#gotify-url')).toBeVisible();
  });

  test('should show AI tab', async ({ page }) => {
    await page.click('#ai-tab');
    await expect(page.locator('#ai-pane')).toBeVisible();
    await expect(page.locator('#ai-settings-form')).toBeVisible();
    await expect(page.locator('#ai-track-extract-url')).toBeVisible();
  });

  test('should show Email tab with settings form', async ({ page }) => {
    await page.click('#email-tab');
    await expect(page.locator('#email-pane')).toBeVisible();
    await expect(page.locator('#email-settings-form')).toBeVisible();
    await expect(page.locator('#smtp-host')).toBeVisible();
    await expect(page.locator('#smtp-port')).toBeVisible();
    await expect(page.locator('#racer-email-list')).toBeAttached();
  });

  test('should show Analytics tab with umami settings', async ({ page }) => {
    await page.click('#umami-tab');
    await expect(page.locator('#umami-pane')).toBeVisible();
    await expect(page.locator('#umami-form')).toBeVisible();
    await expect(page.locator('#umami-url')).toBeVisible();
  });

  test('should show Backup tab with settings and manual backup', async ({ page }) => {
    await page.click('#backup-tab');
    await expect(page.locator('#backup-pane')).toBeVisible();
    await expect(page.locator('#backup-form')).toBeVisible();
    await expect(page.locator('#backup-manual-btn')).toBeVisible();
    await expect(page.locator('#backup-list')).toBeVisible();
  });
});

async function loginAsAdmin(page: Page) {
  await page.goto('/admin.html');
  if (await page.locator('#adminTabs').count() > 0) return;

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
  await expect(page.locator('#adminTabs')).toBeVisible({ timeout: 10000 });
}
