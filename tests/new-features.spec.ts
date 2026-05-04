import { test, expect, Page } from '@playwright/test';

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
    await expect(page.locator('a[href="/"]')).toBeVisible();
    await expect(page.locator('a[href="/trophies.html"]')).toBeVisible();
    await expect(page.locator('a[href="/controller.html"]')).toBeVisible();
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

test.describe('Controller Page', () => {
  test('should redirect to login when not authenticated', async ({ page }) => {
    await page.goto('/controller.html');
    await expect(page).toHaveURL(/login|setup/, { timeout: 5000 });
  });
});

test.describe('One-Off Races Feature', () => {
  test('controller requires authentication', async ({ page }) => {
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
    await page.selectOption('form#racer-form select[name="car_color"]', 'purple');
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

  test('should show archive tab', async ({ page }) => {
    await page.click('#archive-tab');
    await expect(page.locator('#archive-name')).toBeVisible();
    await expect(page.locator('#archive-date')).toBeVisible();
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

  test('should add a racer without profile picture', async ({ page }) => {
    await page.click('#racers-tab');
    await page.click('[onclick="openRacerModal()"]');
    await page.waitForSelector('#racerModal.show');

    await page.fill('form#racer-form input[name="name"]', 'No Pic Racer');
    await page.fill('form#racer-form input[name="profile_picture"]', '');
    await page.fill('form#racer-form input[name="car_name"]', 'Shadow');
    await page.selectOption('form#racer-form select[name="car_color"]', 'black');
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
    await page.selectOption('form#racer-form select[name="car_color"]', 'red');
    await page.fill('form#racer-form input[name="points"]', '0');
    await page.fill('form#racer-form input[name="rank"]', '99');
    await page.fill('form#racer-form input[name="position"]', '0');

    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#racer-list')).toContainText('Min Racer');
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

async function loginAsAdmin(page: Page) {
  await page.goto('/login.html');
  await page.waitForLoadState('domcontentloaded');

  const url = page.url();
  const isSetup = url.includes('/setup') || await page.locator('#setup-form').count() > 0;

  if (isSetup) {
    await page.fill('#setup-form input[name="username"]', 'admin');
    await page.fill('#setup-form input[name="password"]', 'admin123');
    await page.fill('#setup-form input[name="confirm_password"]', 'admin123');
    await page.click('#setup-form button[type="submit"]');
  } else {
    await page.fill('#login-form input[name="username"]', 'admin');
    await page.fill('#login-form input[name="password"]', 'admin123');
    await page.click('#login-form button[type="submit"]');
  }

  await page.waitForURL(/admin/, { timeout: 10000 });
  await expect(page.locator('#adminTabs')).toBeVisible({ timeout: 5000 });
}
