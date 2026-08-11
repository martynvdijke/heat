import { test, expect, Page } from '@playwright/test';

// Extension ownership e2e: the admin Extensions tab exposes an "Owned" toggle
// per extension, and all content-selection surfaces (race-day track select,
// controller track select) only show content from owned extensions.

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

async function openExtensionsTab(page: Page) {
  const tabBtn = page.locator('button[data-tab-id="extensions"]');
  // The click can be lost if it lands before htmx finishes booting (the tab is
  // loaded via hx-get); retry until the tab is actually mounted.
  for (let attempt = 0; attempt < 3; attempt++) {
    await tabBtn.click();
    try {
      await expect(tabBtn).toHaveClass(/active/, { timeout: 5000 });
      await expect(page.locator('#extensions-list input.owned-ext').first()).toBeAttached({ timeout: 10000 });
      return;
    } catch (err) {
      if (attempt === 2) throw err;
      await page.waitForTimeout(500);
    }
  }
}

function ownedRow(page: Page, name: string) {
  return page.locator('#extensions-list tr', { hasText: name }).locator('input.owned-ext');
}

function putOwnedResponse(page: Page) {
  return page.waitForResponse(
    (r) => r.url().includes('/api/extensions/owned') && r.request().method() === 'PUT',
  );
}

// Moves the seeded base track Monza to the Heavy Rain extension (id 2) and back,
// as a proxy for "content the group does/does not own". Returns the fetch result.
async function moveMonza(page: Page, extensionId: number) {
  return page.evaluate(async (extId) => {
    const res = await fetch('/api/content/extension', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content_type: 'track', content_id: 'monza', extension_id: extId }),
    });
    return res.ok;
  }, extensionId);
}

// The Race Day tab is the default initial panel, but it is not hx-loaded: once
// another tab has replaced #admin-tab-container, navigating back via a hash-only
// URL change is a same-document navigation and never restores the race-day DOM.
// A real reload guarantees the fresh, default race-day layout.
async function gotoRaceDay(page: Page) {
  await page.goto('/admin.html#race-day');
  await page.reload();
}

test.describe.serial('Extension ownership', () => {
  test('base game always owned, heavy rain unowned by default', async ({ page }) => {
    await loginAsAdmin(page);
    await openExtensionsTab(page);

    const baseBox = ownedRow(page, 'Base Game');
    await expect(baseBox).toBeChecked();
    await expect(baseBox).toBeDisabled();

    const rainBox = ownedRow(page, 'Heavy Rain');
    await expect(rainBox).not.toBeChecked();
    await expect(rainBox).toBeEnabled();
  });

  test('owning heavy rain persists through the owned API', async ({ page }) => {
    await loginAsAdmin(page);
    await openExtensionsTab(page);

    const put = putOwnedResponse(page);
    await ownedRow(page, 'Heavy Rain').check();
    expect((await put).ok()).toBeTruthy();

    // saveOwned re-renders the table; the checkbox must stay checked.
    await expect(ownedRow(page, 'Heavy Rain')).toBeChecked({ timeout: 10000 });

    const restore = putOwnedResponse(page);
    await ownedRow(page, 'Heavy Rain').uncheck();
    expect((await restore).ok()).toBeTruthy();
    await expect(ownedRow(page, 'Heavy Rain')).not.toBeChecked({ timeout: 10000 });
  });

  test('race day track select excludes unowned extension tracks', async ({ page }) => {
    await loginAsAdmin(page);
    expect(await moveMonza(page, 2)).toBeTruthy();

    await gotoRaceDay(page);
    // Wait for the select to be populated from the owned list.
    await expect(page.locator('#track-select option', { hasText: 'Interlagos' })).toHaveCount(1, { timeout: 10000 });
    await expect(page.locator('#track-select option', { hasText: 'Monza' })).toHaveCount(0);

    expect(await moveMonza(page, 1)).toBeTruthy();
  });

  test('owning the extension restores tracks in the race day select', async ({ page }) => {
    await loginAsAdmin(page);
    expect(await moveMonza(page, 2)).toBeTruthy();

    await openExtensionsTab(page);
    const put = putOwnedResponse(page);
    await ownedRow(page, 'Heavy Rain').check();
    expect((await put).ok()).toBeTruthy();
    await expect(ownedRow(page, 'Heavy Rain')).toBeChecked({ timeout: 10000 });

    await gotoRaceDay(page);
    await expect(page.locator('#track-select option', { hasText: 'Monza' })).toHaveCount(1, { timeout: 10000 });

    // Cleanup: un-own Heavy Rain and move Monza back to the Base Game.
    const restore = await page.evaluate(async () => {
      const res = await fetch('/api/extensions/owned', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ owned_ids: [1] }),
      });
      return res.ok;
    });
    expect(restore).toBeTruthy();
    expect(await moveMonza(page, 1)).toBeTruthy();
  });

  test('controller track select excludes unowned extension tracks', async ({ page }) => {
    await loginAsAdmin(page);
    expect(await moveMonza(page, 2)).toBeTruthy();

    await page.goto('/controller.html');
    await expect(page.locator('#track-select option', { hasText: 'Interlagos' })).toHaveCount(1, { timeout: 10000 });
    await expect(page.locator('#track-select option', { hasText: 'Monza' })).toHaveCount(0);

    expect(await moveMonza(page, 1)).toBeTruthy();
  });
});
