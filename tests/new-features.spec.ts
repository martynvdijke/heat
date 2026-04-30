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
    await expect(page.locator('a[href="/chat.html"]')).toBeVisible();
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
  test('should load controller page', async ({ page }) => {
    await page.goto('/controller.html');
    await expect(page).toHaveTitle(/HEAT: Race Controller/);
    
    const heading = page.locator('h5').first();
    await expect(heading).toContainText('Race Control');
  });

  test('should have race type selector', async ({ page }) => {
    await page.goto('/controller.html');
    const raceType = page.locator('#race-type');
    await expect(raceType).toBeVisible();
    await expect(raceType).toHaveValue('season');
  });

  test('should have quick action buttons', async ({ page }) => {
    await page.goto('/controller.html');
    await expect(page.locator('button:has-text("Shuffle Grid")')).toBeVisible();
    await expect(page.locator('button:has-text("Save to History")')).toBeVisible();
  });

  test('should have track selector', async ({ page }) => {
    await page.goto('/controller.html');
    const trackSelect = page.locator('#track-select');
    await expect(trackSelect).toBeVisible();
  });
});

test.describe('Chat Page', () => {
  test('should load chat page', async ({ page }) => {
    await page.goto('/chat.html');
    await expect(page).toHaveTitle(/HEAT: Live Chat/);
    
    const heading = page.locator('h5');
    await expect(heading).toContainText('Live Commentary');
  });

  test('should have chat input', async ({ page }) => {
    await page.goto('/chat.html');
    const chatInput = page.locator('#chat-input');
    await expect(chatInput).toBeVisible();
  });

  test('should display viewer count', async ({ page }) => {
    await page.goto('/chat.html');
    const viewerCount = page.locator('#viewer-count');
    await expect(viewerCount).toBeVisible();
  });

  test('should have navigation links', async ({ page }) => {
    await page.goto('/chat.html');
    await expect(page.locator('a[href="/"]')).toBeVisible();
    await expect(page.locator('a[href="/stats.html"]')).toBeVisible();
    await expect(page.locator('a[href="/controller.html"]')).toBeVisible();
  });
});

test.describe('One-Off Races Feature', () => {
  test('should allow selecting one-off race type in controller', async ({ page }) => {
    await page.goto('/controller.html');
    const raceType = page.locator('#race-type');
    await raceType.selectOption('oneoff');
    await expect(raceType).toHaveValue('oneoff');
  });
});