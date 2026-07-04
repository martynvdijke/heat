import { test, expect, Page } from '@playwright/test';

test.describe.serial('Admin Performance Trace', () => {

  async function loginAsAdmin(page: Page) {
    await page.goto('/admin.html');
    if (await page.locator('#adminCategories').count() > 0) return;
    await page.waitForSelector('#setup-form, #login-form', { timeout: 10000 });
    if (await page.locator('#setup-form').count() > 0) {
      await page.fill('#setup-form input[name="username"]', 'admin');
      await page.fill('#setup-form input[name="password"]', 'admin123');
      await page.fill('#setup-form input[name="confirm_password"]', 'admin123');
      await page.click('#setup-form button[type="submit"]');
      try { await page.waitForURL(/admin/, { timeout: 5000 }); } catch {}
    }
    if (!page.url().includes('/admin')) {
      await page.waitForSelector('#login-form', { timeout: 10000 });
      await page.fill('#login-form input[name="username"]', 'admin');
      await page.fill('#login-form input[name="password"]', 'admin123');
      await page.click('#login-form button[type="submit"]');
    }
    await page.waitForURL(/admin/, { timeout: 20000 });
    await expect(page.locator('#adminCategories')).toBeVisible({ timeout: 10000 });
  }

  test('capture performance timeline', async ({ page }) => {
    await page.coverage.startJSCoverage();
    await page.coverage.startCSSCoverage();
    await page.goto('/login.html');
    await page.goto('/admin.html');
    if (await page.locator('#setup-form').count() > 0) {
      await page.fill('#setup-form input[name="username"]', 'admin');
      await page.fill('#setup-form input[name="password"]', 'admin123');
      await page.fill('#setup-form input[name="confirm_password"]', 'admin123');
      await page.click('#setup-form button[type="submit"]');
    }
    await page.waitForURL(/admin/, { timeout: 20000 });
    await page.waitForSelector('#adminCategories', { timeout: 10000 });

    // Capture timing via Performance API
    const perfData = await page.evaluate(() => {
      const entries = performance.getEntriesByType('resource');
      return entries.map(e => ({
        name: e.name.split('/').slice(-3).join('/'),
        duration: Math.round(e.duration),
        initiatorType: e.initiatorType,
      })).sort((a, b) => b.duration - a.duration);
    });

    console.log(`\n=== Resource Load Times (slowest first) ===`);
    const thresholds: Record<string, number> = {};
    for (const r of perfData) {
      console.log(`  ${r.duration.toString().padStart(6)}ms  [${r.initiatorType}] ${r.name}`);
      if (r.duration > 1000) {
        const key = r.initiatorType;
        thresholds[key] = (thresholds[key] || 0) + 1;
      }
    }

    console.log(`\n=== Slow Resources (>1000ms) ===`);
    for (const [type, count] of Object.entries(thresholds)) {
      console.log(`  ${type}: ${count} resources > 1s`);
    }

    const paintEntries = await page.evaluate(() => {
      return performance.getEntriesByType('paint').map(e => ({
        name: e.name,
        startTime: Math.round(e.startTime),
      }));
    });

    console.log(`\n=== Paint Timings ===`);
    for (const p of paintEntries) {
      console.log(`  ${p.name}: ${p.startTime}ms`);
    }

    const longTasks = await page.evaluate(() => {
      return performance.getEntriesByType('measure').filter(m => m.duration > 50).length;
    });
    console.log(`\nLong (>50ms) tasks: ${longTasks}`);
    console.log(`Total resources: ${perfData.length}`);
  });
});
