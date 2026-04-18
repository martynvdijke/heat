import { test, expect } from '@playwright/test';

test.describe('Admin Interface', () => {
  
  test.beforeEach(async ({ page }) => {
    // We go to login page first. If setup is not done, it redirects to setup.html
    await page.goto('/login.html');
    
    const title = await page.title();
    if (title.includes('Setup')) {
      // Perform initial setup
      await page.fill('input[name="username"]', 'admin');
      await page.fill('input[name="password"]', 'password123');
      await page.fill('input[name="confirm_password"]', 'password123');
      await page.click('button[type="submit"]');
    } else {
      // Perform login
      await page.fill('input[name="username"]', 'admin');
      await page.fill('input[name="password"]', 'password123');
      await page.click('button[type="submit"]');
    }
    
    // Ensure we are on admin page
    await expect(page).toHaveURL(/\/admin\.html/);
    await expect(page.locator('h1')).toContainText('HEAT RACE CONTROL');
  });

  test('should navigate through tabs', async ({ page }) => {
    // Default tab is Race Setup
    await expect(page.locator('#race-pane')).toBeVisible();

    // Switch to Racers tab
    await page.click('#racers-tab');
    await expect(page.locator('#racers-pane')).toBeVisible();

    // Switch to Quotes tab
    await page.click('#quotes-tab');
    await expect(page.locator('#quotes-pane')).toBeVisible();

    // Switch to Archive tab
    await page.click('#archive-tab');
    await expect(page.locator('#archive-pane')).toBeVisible();
  });

  test('should add a new racer', async ({ page }) => {
    await page.click('#racers-tab');
    await page.click('button:has-text("Add Racer")');

    // Fill racer modal
    await page.fill('#racer-form input[name="name"]', 'L. NORRIS');
    await page.fill('#racer-form input[name="profile_picture"]', 'https://via.placeholder.com/150');
    await page.fill('#racer-form input[name="car_name"]', 'MCL38');
    await page.selectOption('#racer-form select[name="car_color"]', 'yellow');
    await page.fill('#racer-form input[name="points"]', '150');
    await page.fill('#racer-form input[name="rank"]', '2');
    await page.fill('#racer-form input[name="position"]', '10');

    await page.click('#racer-form button:has-text("Save Racer")');

    // Verify racer is in list
    await expect(page.locator('#racer-list')).toContainText('L. NORRIS');
    await expect(page.locator('#racer-list')).toContainText('MCL38');
  });

  test('should update current race details', async ({ page }) => {
    await page.click('#race-tab');
    
    // Fill race details
    await page.fill('#race-form input[name="country"]', 'Netherlands');
    await page.fill('#race-form input[name="track"]', 'Zandvoort');
    await page.fill('#race-form input[name="laps"]', '72');
    await page.fill('#race-form input[name="track_id"]', 'zandvoort');

    // Handle alert after submit
    page.on('dialog', dialog => dialog.accept());
    await page.click('#race-form button:has-text("Update Race Data")');

    // Refresh and verify
    await page.reload();
    await expect(page.locator('#race-form input[name="track"]')).toHaveValue('Zandvoort');
  });

  test('should add a new quote', async ({ page }) => {
    await page.click('#quotes-tab');
    await page.click('button:has-text("Add Quote")');

    await page.fill('#quote-form textarea[name="text"]', 'It is a marathon, not a sprint!');
    await page.fill('#quote-form input[name="author"]', 'Lewis Hamilton');

    await page.click('#quote-form button:has-text("Save Quote")');

    // Verify quote in list
    await expect(page.locator('#quote-list')).toContainText('Lewis Hamilton');
    await expect(page.locator('#quote-list')).toContainText('marathon');
  });

  test('should delete a racer', async ({ page }) => {
    await page.click('#racers-tab');
    
    // We need at least one racer to delete. Let's find one.
    const racerCount = await page.locator('#racer-list tr').count();
    if (racerCount > 0) {
      const firstRacerName = await page.locator('#racer-list tr').first().locator('.fw-bold').innerText();
      
      page.on('dialog', dialog => dialog.accept());
      await page.locator('#racer-list tr').first().locator('button.btn-outline-danger').click();
      
      await expect(page.locator('#racer-list')).not.toContainText(firstRacerName);
    }
  });

  test('should delete a quote', async ({ page }) => {
    await page.click('#quotes-tab');
    
    const quoteCount = await page.locator('#quote-list tr').count();
    if (quoteCount > 0) {
      const firstQuoteText = await page.locator('#quote-list tr').first().locator('em').innerText();
      
      page.on('dialog', dialog => dialog.accept());
      await page.locator('#quote-list tr').first().locator('button.btn-outline-danger').click();
      
      await expect(page.locator('#quote-list')).not.toContainText(firstQuoteText.replace(/"/g, ''));
    }
  });

  test('should archive a race with a custom name', async ({ page }) => {
    await page.click('#race-tab');
    
    const archiveName = `Season ${new Date().getFullYear()}`;
    await page.fill('#archive-name', archiveName);
    
    // Handle confirmation and success alerts
    page.on('dialog', async dialog => {
      await dialog.accept();
    });

    await page.click('button:has-text("Archive Results")');

    // Check archive tab
    await page.click('#archive-tab');
    await expect(page.locator('#history-list')).toContainText(archiveName);
  });
});
