import { test, expect, Page } from '@playwright/test';

test.describe('Car Color Rendering', () => {
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
      // A seeded racer (e.g. rank 1) should have car_color "red"
      await page.goto('/');
      await page.waitForSelector('#leaderboard-body tr');

      // The first row's color indicator should have background matching #ff4444
      const firstColorDot = page.locator('#leaderboard-body .color-indicator').first();
      const bg = await firstColorDot.evaluate(el => getComputedStyle(el).backgroundColor);

      // #ff4444 in rgb
      expect(bg).toBe('rgb(255, 68, 68)');
    });
  });
});

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
  await page.locator(subTabSelector).waitFor({ state: 'visible', timeout: 10000 });
  await page.click(subTabSelector);
}
