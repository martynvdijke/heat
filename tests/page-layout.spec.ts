import { test, expect } from '@playwright/test';

test.describe('Page Layout Regression — CSS body cascade fix', () => {
  // Previously, page-specific body styles in style.css used bare selectors.
  // The last declaration (login: display:flex; align-items:center; justify-content:center;)
  // won the CSS cascade and flex-centered every page, pushing headers/navbars to center.
  // Fix: scope with body[data-page="..."] so styles only apply to their page.

  // Pages loading style.css — body should be default block, not flex-centered.
  // Login and driver pages have intentional inline flex (they don't load style.css).
  // Controller page is excluded: it requires auth and redirects to login.html (flex body).
  const PAGES = [
    { path: '/',              desc: 'Index' },
    { path: '/tv.html',       desc: 'TV' },
    { path: '/pitboard.html', desc: 'Pitboard' },
    { path: '/spectator.html',desc: 'Spectator' },
    { path: '/player.html',   desc: 'Player' },
    { path: '/replay.html',   desc: 'Replay' },
  ];

  for (const { path, desc } of PAGES) {
    test(`${desc} page body should not be flex-centered`, async ({ page }) => {
      await page.goto(path);
      const display = await page.evaluate(() => getComputedStyle(document.body).display);
      expect(display).toBe('block');
    });
  }

  test('Login page body should be flex-centered (intentional, no style.css)', async ({ page }) => {
    await page.goto('/login.html');
    const display = await page.evaluate(() => getComputedStyle(document.body).display);
    expect(display).toBe('flex');
  });

  test('TV page header should be at top of viewport', async ({ page }) => {
    await page.goto('/tv.html');
    const header = page.locator('.tv-header');
    const box = await header.boundingBox();
    expect(box).not.toBeNull();
    // .tv-header is position: fixed; top: 0;
    expect(box!.y).toBe(0);
  });
});
