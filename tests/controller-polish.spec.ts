import { test, expect } from '@playwright/test';

test.describe('Controller Polish', () => {
  async function loginAsAdmin(page: import('@playwright/test').Page) {
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

  test('should update the weather chip when weather is set', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/controller.html');
    await expect(page.locator('#weather-chip')).toBeVisible();

    await page.selectOption('#weather-condition', 'wet');
    await page.click('[data-action="setWeather"]');

    await expect(page.locator('#weather-chip-text')).toContainText('Wet');
    await expect(page.locator('#current-weather')).toContainText('Wet');
  });

  test('should show LEAD and +N gaps after recording laps', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/controller.html');
    await expect(page.locator('.driver-row').first()).toBeVisible();

    // Create uniquely-named racers so other specs' data can never collide
    // (duplicate names across specs caused strict-mode violations in CI).
    const suffix = Date.now();
    const leaderName = `Gap Leader ${suffix}`;
    const lappedName = `Gap Lapped ${suffix}`;
    const created = await page.evaluate(async ([leader, lapped]) => {
      for (const name of [leader, lapped]) {
        const res = await fetch('/api/racers', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            id: 0, name, profile_picture: '', car_color: '#ff0000',
            car_name: 'Gap Test Car', points: 0, rank: 0, position: 0
          })
        });
        if (!res.ok) return false;
      }
      return true;
    }, [leaderName, lappedName] as const);
    expect(created).toBeTruthy();

    const racers = await (await page.request.get('/api/racers')).json();
    const leader = racers.find((r: any) => r.name === leaderName);
    const lapped = racers.find((r: any) => r.name === lappedName);
    expect(leader).toBeTruthy();
    expect(lapped).toBeTruthy();

    // Lap the lapped driver: only the leader is recorded on the later lap.
    const recordsAll = [leader.id, lapped.id].map((id: number, i: number) => ({
      racer_id: id, position: i + 1, gear_used: 0, heat_generated: 0, turbo_used: false
    }));
    const recordsLapped = [{ racer_id: leader.id, position: 1, gear_used: 0, heat_generated: 0, turbo_used: false }];

    // Unix seconds are strictly increasing, so these laps always outrank every
    // earlier record (the gap computation picks the latest position-1 record
    // as the race leader) regardless of which project ran before us.
    const lapBase = Math.floor(Date.now() / 1000);

    // The batch endpoint silently skips rows on transient SQLite busy errors,
    // so post, verify via GET, and re-post until both records are durable.
    const expectedLaps: Array<[number, number]> = [
      [leader.id, lapBase + 1],
      [lapped.id, lapBase]
    ];
    await expect(async () => {
      await page.request.post('/api/lap-records/batch', { data: { race_id: 0, lap: lapBase, records: recordsAll } });
      await page.request.post('/api/lap-records/batch', { data: { race_id: 0, lap: lapBase + 1, records: recordsLapped } });
      const records = await (await page.request.get('/api/lap-records?race_id=0')).json();
      for (const [racerId, lap] of expectedLaps) {
        expect(records).toContainEqual(expect.objectContaining({ racer_id: racerId, lap_number: lap }));
      }
    }).toPass({ timeout: 15000 });

    await page.reload();
    await expect(page.locator('.driver-row').first()).toBeVisible();

    // Leader (position 1 at the latest lap) shows LEAD.
    const leaderRow = page.locator('.driver-row', { hasText: leaderName });
    await expect(leaderRow).toHaveCount(1);
    await expect(leaderRow.locator('.gap-cell')).toHaveText('LEAD');

    // Lapped driver shows +1.
    const lappedRow = page.locator('.driver-row', { hasText: lappedName });
    await expect(lappedRow).toHaveCount(1);
    await expect(lappedRow.locator('.gap-cell')).toHaveText('+1');
  });

  test('should render next race countdown when next_race_date is set', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/controller.html');
    await expect(page.locator('#next-race-line')).toBeHidden();

    const info = await (await page.request.get('/api/race-info')).json();
    // Use the browser's own fetch so the session cookie + Origin header are sent
    // (page.request does not reliably carry the session cookie on all browsers).
    const postOk = await page.evaluate(async (body) => {
      const res = await fetch('/api/race-info', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
      return res.ok;
    }, { ...info, next_race_date: '2099-01-01' });
    expect(postOk).toBeTruthy();

    await page.reload();
    await expect(page.locator('#next-race-line')).toBeVisible();
    await expect(page.locator('#next-race-line')).toContainText('Next race: 2099-01-01');
  });

  test('should hide next race countdown when next_race_date is unset', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/controller.html');

    const info = await (await page.request.get('/api/race-info')).json();
    const postOk = await page.evaluate(async (body) => {
      const res = await fetch('/api/race-info', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
      return res.ok;
    }, { ...info, next_race_date: '' });
    expect(postOk).toBeTruthy();

    await page.reload();
    await expect(page.locator('#next-race-line')).toBeHidden();
  });
});
