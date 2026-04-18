import { test, expect } from '@playwright/test';

test.describe('Index Page', () => {
  test('should load the index page without console errors', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') consoleErrors.push(msg.text());
    });

    await page.goto('/');

    // Check for title
    await expect(page).toHaveTitle(/HEAT: Pedal to the Metal/);

    // Check if the map is initialized (it should have leaflet classes)
    const map = page.locator('#circuit-map');
    await expect(map).toHaveClass(/leaflet-container/);

    // Check if quotes are loaded
    const quote = page.locator('#random-quote');
    await expect(quote).not.toContainText('Loading commentary...');

    // Verify no console errors occurred during load
    expect(consoleErrors).toEqual([]);
  });

  test('should display leaderboard', async ({ page }) => {
    await page.goto('/');
    const rows = page.locator('#leaderboard-body tr');
    // Seed data has at least one racer
    await expect(rows.first()).toBeVisible();
  });

  test('should switch circuits', async ({ page }) => {
    await page.goto('/');
    const selector = page.locator('#track-selector');
    
    // Select Spa
    await selector.selectOption('spa');
    
    // Check if race details updated (Monza is 5km, Spa is 7km)
    await expect(page.locator('#race-length')).toHaveText('7');
    await expect(page.locator('#race-track')).toHaveText(/Spa/);
  });
});
