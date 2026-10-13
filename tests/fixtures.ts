// Shared Playwright fixtures.
//
// The pages under test load styles/scripts/fonts from public CDNs. Chromium
// keeps the document `load` event pending while an in-use web font request is
// outstanding, so a stalled CDN fetch (seen in release CI with the cdnjs
// FontAwesome webfont) makes Playwright's default `waitUntil: 'load'`
// navigation time out even though the page is fully rendered.
// `domcontentloaded` is not blocked by subresources, and Playwright's
// auto-waiting locators cover everything after that. Callers can still pass an
// explicit `waitUntil` to `page.goto()` to opt out.
import { test as base, expect } from '@playwright/test';

export { expect };
export type { Page } from '@playwright/test';

export const test = base.extend({
  page: async ({ page }, use) => {
    const originalGoto = page.goto.bind(page);
    page.goto = (url, options) => originalGoto(url, { waitUntil: 'domcontentloaded', ...options });
    await use(page);
  },
});
