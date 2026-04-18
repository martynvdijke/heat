import { test, expect } from '@playwright/test';

test.describe('Admin Page', () => {
  test('should load login or setup page', async ({ page }) => {
    await page.goto('/login.html');
    const title = await page.title();
    expect(title).toMatch(/Login|Setup/);
  });
});

test.describe('Admin Interface', () => {
  test('should redirect to login when accessing admin without session', async ({ page }) => {
    await page.goto('/admin.html');
    await expect(page).toHaveURL(/login|setup/, { timeout: 5000 });
  });
});