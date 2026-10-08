import { test, expect } from '@playwright/test';

// The full email-link round trip needs a token that is only ever delivered by
// email or the Telegram bot, so these tests verify the public frontend wiring
// (unauthenticated states) plus the session boundary of the racer API.
test.describe('Racer identity', () => {
  test('verify page rejects a missing token', async ({ page }) => {
    await page.goto('/verify.html');
    await expect(page.locator('#verify-body')).toContainText('Invalid link');
    await expect(page.locator('#verify-body')).toContainText('No verification token');
  });

  test('verify page rejects an invalid token', async ({ page }) => {
    await page.goto('/verify.html?token=not-a-real-token');
    await expect(page.locator('#verify-body')).toContainText('Link expired');
  });

  test('me page prompts for an email when signed out', async ({ page }) => {
    await page.goto('/me.html');
    await expect(page.locator('#link-form')).toBeVisible();
    await page.fill('#link-email', 'someone@example.com');
    await page.click('#link-form button[type="submit"]');
    await expect(page.locator('#link-msg')).toContainText('sign-in link is on its way');
  });

  test('me API requires a racer session', async ({ request }) => {
    const res = await request.get('/api/me');
    expect(res.status()).toBe(401);
  });
});
