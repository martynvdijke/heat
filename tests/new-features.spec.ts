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

async function loginAsAdmin(page) {
  await page.goto('/login.html');
  await page.waitForLoadState('networkidle');
  
  const title = await page.title();
  const isSetup = title.includes('Setup') || (await page.locator('input[name="confirm_password"]').count()) > 0;
  
  if (isSetup) {
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.fill('input[name="confirm_password"]', 'admin123');
  } else {
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
  }
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
}

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