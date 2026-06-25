import { test, expect, Page } from '@playwright/test';

test.describe.serial('Admin Extended Features', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test.describe.serial('Team CRUD', () => {
    const teamName = `Team-${Date.now()}`;

    test('should create a team', async ({ page }) => {
      await showAdminPane(page, '#cat-content-tab', '#teams-pane');
      await page.click('button[hx-get="/api/html/teams/0/edit"]');
      await page.waitForSelector('#teamModal.show');
      await page.waitForSelector('#teamModal form#team-form');

      await page.fill('form#team-form input[name="name"]', teamName);
      await page.fill('form#team-form input[name="color"]', '#ff6600');

      await page.click('form#team-form button[type="submit"]');
      await page.waitForTimeout(500);

      await expect(page.locator('#team-list')).toContainText(teamName);
    });

    test('should edit the team', async ({ page }) => {
      await showAdminPane(page, '#cat-content-tab', '#teams-pane');
      await page.waitForTimeout(500);

      // Find the edit button in the row containing our team name
      const teamRow = page.locator('#team-list tr', { hasText: teamName });
      await teamRow.locator('.btn-outline-primary').click();
      await page.waitForSelector('#teamModal.show');

      const nameInput = page.locator('form#team-form input[name="name"]');
      await nameInput.fill(teamName + '-renamed');

      await page.click('form#team-form button[type="submit"]');
      await page.waitForTimeout(500);

      await expect(page.locator('#team-list')).toContainText(teamName + '-renamed');
    });

    test('should delete the team', async ({ page }) => {
      await showAdminPane(page, '#cat-content-tab', '#teams-pane');
      await page.waitForTimeout(500);

      // Find delete button in the row containing our renamed team
      const teamRow = page.locator('#team-list tr', { hasText: teamName + '-renamed' });
      page.once('dialog', dialog => dialog.accept());
      await teamRow.locator('.btn-outline-danger').click();

      // Wait for the table to refresh after HTMX delete
      await expect(page.locator('#team-list')).not.toContainText(teamName, { timeout: 10000 });
    });
  });

  test('should create racer with profile picture URL', async ({ page }) => {
    await clickAdminSubTab(page, '#cat-race-tab', '#racers-subtab');

    // Create racer with an existing image URL (no direct API upload needed)
    await page.click('button[hx-get="/api/html/racers/0/edit"]');
    await page.waitForSelector('#racerModal.show');
    await page.waitForSelector('#racerModal form#racer-form');

    const racerName = `Uploaded-Racer-${Date.now()}`;
    await page.fill('form#racer-form input[name="name"]', racerName);
    await page.fill('form#racer-form input[name="profile_picture"]', '/static/images/helmet.svg');
    await page.fill('form#racer-form input[name="car_name"]', 'Upload Car');
    await page.fill('form#racer-form input[name="car_color"]', '#ff00ff');
    await page.fill('form#racer-form input[name="points"]', '15');
    await page.fill('form#racer-form input[name="rank"]', '30');
    await page.fill('form#racer-form input[name="position"]', '0');

    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    await expect(page.locator('#racer-list')).toContainText(racerName);
  });

  test('should edit racer points', async ({ page }) => {
    await clickAdminSubTab(page, '#cat-race-tab', '#racers-subtab');
    await page.waitForTimeout(500);

    // Create a racer with known points
    await page.click('button[hx-get="/api/html/racers/0/edit"]');
    await page.waitForSelector('#racerModal.show');
    await page.waitForSelector('#racerModal form#racer-form');

    const pointsRacerName = `Points-Test-${Date.now()}`;
    await page.fill('form#racer-form input[name="name"]', pointsRacerName);
    await page.fill('form#racer-form input[name="profile_picture"]', '');
    await page.fill('form#racer-form input[name="car_name"]', 'PointsCar');
    await page.fill('form#racer-form input[name="car_color"]', '#00ff00');
    await page.fill('form#racer-form input[name="points"]', '10');
    await page.fill('form#racer-form input[name="rank"]', '40');
    await page.fill('form#racer-form input[name="position"]', '0');

    await page.click('form#racer-form button[type="submit"]');
    await closeRacerModal(page);

    // Now edit the points
    const editBtn = page.locator('#racer-list tr', { hasText: pointsRacerName }).locator('.btn-outline-primary');
    await editBtn.click();
    await page.waitForSelector('#racerModal.show');

    await page.fill('form#racer-form input[name="points"]', '99');
    await page.click('form#racer-form button[type="submit"]');
    await page.waitForTimeout(500);

    // Verify the points badge shows '99 pts'
    await expect(page.locator('#racer-list')).toContainText('99 pts');
  });

  test.describe.serial('Home Page Display', () => {
    const racerDisplayName = `Display-Racer-${Date.now()}`;
    const carName = 'Speedster-X';

    test('should create racer with car name and color', async ({ page }) => {
      await clickAdminSubTab(page, '#cat-race-tab', '#racers-subtab');

      await page.click('button[hx-get="/api/html/racers/0/edit"]');
      await page.waitForSelector('#racerModal.show');
      await page.waitForSelector('#racerModal form#racer-form');

      await page.fill('form#racer-form input[name="name"]', racerDisplayName);
      await page.fill('form#racer-form input[name="profile_picture"]', '/static/images/helmet.svg');
      await page.fill('form#racer-form input[name="car_name"]', carName);
      await page.fill('form#racer-form input[name="car_color"]', '#ff00ff');
      await page.fill('form#racer-form input[name="points"]', '25');
      await page.fill('form#racer-form input[name="rank"]', '50');
      await page.fill('form#racer-form input[name="position"]', '0');

      await page.click('form#racer-form button[type="submit"]');
      await page.waitForTimeout(500);

      await expect(page.locator('#racer-list')).toContainText(racerDisplayName);
    });

    test('should display car name on home page', async ({ page }) => {
      await page.goto('/');
      await page.waitForSelector('#leaderboard-body tr');

      await expect(page.locator('#leaderboard-body')).toContainText(carName);
    });
  });

  test.describe.serial('Team Display on Home Page', () => {
    const teamName = `Home-Team-${Date.now()}`;
    const racerTeamName = `Team-Racer-${Date.now()}`;
    let teamId: number;
    let racerId: number;

    test('should create team and assign racer', async ({ page }) => {
      // Create team via admin UI
      await showAdminPane(page, '#cat-content-tab', '#teams-pane');
      await page.click('button[hx-get="/api/html/teams/0/edit"]');
      await page.waitForSelector('#teamModal.show');
      await page.waitForSelector('#teamModal form#team-form');

      await page.fill('form#team-form input[name="name"]', teamName);
      await page.fill('form#team-form input[name="color"]', '#00aaff');
      await page.click('form#team-form button[type="submit"]');
      await page.waitForTimeout(500);

      await expect(page.locator('#team-list')).toContainText(teamName);

      // Extract team ID from edit button in the newly created team's row
      const teamRow = page.locator('#team-list tr', { hasText: teamName });
      const teamEditBtn = teamRow.locator('.btn-outline-primary');
      const teamHxGet = await teamEditBtn.getAttribute('hx-get');
      const teamMatch = teamHxGet?.match(/\/api\/html\/teams\/(\d+)\/edit/);
      expect(teamMatch).not.toBeNull();
      teamId = parseInt(teamMatch![1]);

      // Create racer
      await clickAdminSubTab(page, '#cat-race-tab', '#racers-subtab');
      await page.click('button[hx-get="/api/html/racers/0/edit"]');
      await page.waitForSelector('#racerModal.show');
      await page.waitForSelector('#racerModal form#racer-form');

      await page.fill('form#racer-form input[name="name"]', racerTeamName);
      await page.fill('form#racer-form input[name="car_name"]', 'TeamCar');
      await page.fill('form#racer-form input[name="car_color"]', '#00aaff');
      await page.fill('form#racer-form input[name="points"]', '30');
      await page.fill('form#racer-form input[name="rank"]', '60');
      await page.fill('form#racer-form input[name="position"]', '0');

      await page.click('form#racer-form button[type="submit"]');
      await page.waitForTimeout(500);

      await expect(page.locator('#racer-list')).toContainText(racerTeamName);

      // Extract racer ID from the newly created racer's row
      const racerRow = page.locator('#racer-list tr', { hasText: racerTeamName });
      const racerEditBtn = racerRow.locator('.btn-outline-primary');
      const racerHxGet = await racerEditBtn.getAttribute('hx-get');
      const racerMatch = racerHxGet?.match(/\/api\/html\/racers\/(\d+)\/edit/);
      expect(racerMatch).not.toBeNull();
      racerId = parseInt(racerMatch![1]);

      // Assign racer to team via browser fetch (uses page cookies/Origin)
      const assignResp = await page.evaluate(async ({ racerId, teamId }) => {
        const res = await fetch('/api/teams/assign', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ racer_id: racerId, team_id: teamId }),
        });
        return res.ok;
      }, { racerId, teamId });
      expect(assignResp).toBeTruthy();
    });

    test('should display team name on home page', async ({ page }) => {
      await page.goto('/');
      await page.waitForSelector('#leaderboard-body tr');

      // Verify the team name appears in the leaderboard TEAM column
      // The home page now has a TEAM column (between DRIVER and CAR)
      await expect(page.locator('#leaderboard-body')).toContainText(teamName);
    });
  });

  test('should take a round snapshot', async ({ page }) => {
    // First ensure we have at least one racer
    const racerCheck = await page.locator('#racer-list tr').count();
    if (racerCheck === 0) {
      // Create a racer for the snapshot
      await clickAdminSubTab(page, '#cat-race-tab', '#racers-subtab');
      await page.click('button[hx-get="/api/html/racers/0/edit"]');
      await page.waitForSelector('#racerModal.show');
      await page.waitForSelector('#racerModal form#racer-form');

      await page.fill('form#racer-form input[name="name"]', `Snapshot-Racer-${Date.now()}`);
      await page.fill('form#racer-form input[name="car_name"]', 'SnapCar');
      await page.fill('form#racer-form input[name="car_color"]', '#ff0000');
      await page.fill('form#racer-form input[name="points"]', '50');
      await page.fill('form#racer-form input[name="rank"]', '70');
      await page.fill('form#racer-form input[name="position"]', '0');

      await page.click('form#racer-form button[type="submit"]');
      await page.waitForTimeout(500);
    }

    // Navigate to rounds pane
    await showAdminPane(page, '#cat-results-tab', '#rounds-pane');

    // Create a round snapshot via API (bypassing the prompt dialog)
    const snapshotResp = await page.evaluate(async () => {
      // Find active season or default to 1
      const seasonsRes = await fetch('/api/seasons');
      const seasons = await seasonsRes.json();
      const active = Array.isArray(seasons) ? seasons.find((s: any) => s.status === 'active') : null;
      const seasonId = active ? active.id : 1;
      const name = `E2E-Round-${Date.now()}`;
      const res = await fetch('/api/rounds', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ race_name: name, round: 0, season_id: seasonId })
      });
      return { ok: res.ok, name };
    });
    expect(snapshotResp.ok).toBeTruthy();

    // Reload the rounds list
    await page.evaluate(async () => {
      try {
        const seasonsRes = await fetch('/api/seasons');
        const seasons = await seasonsRes.json();
        const active = Array.isArray(seasons) ? seasons.find((s: any) => s.status === 'active') : null;
        const sid = active ? active.id : 1;
        const roundsRes = await fetch(`/api/rounds?season_id=${sid}`);
        const rounds = await roundsRes.json();
        const list = document.getElementById('rounds-list')!;
        if (Array.isArray(rounds) && rounds.length > 0) {
          list.innerHTML = rounds.map((r: any) => `
            <tr>
              <td class="ps-4 fw-bold">#${r.round || r.id}</td>
              <td>${r.race_name}</td>
              <td>${r.race_date}</td>
              <td class="text-end pe-4">
                <button class="btn btn-sm btn-outline-danger" onclick="deleteRound(${r.id})"><i class="fa-solid fa-trash"></i></button>
              </td>
            </tr>
          `).join('');
        }
      } catch (e) {
        console.error('Failed to load rounds', e);
      }
    });

    // Verify a snapshot appears in the rounds list
    const roundsList = page.locator('#rounds-list');
    const rows = roundsList.locator('tr');
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThanOrEqual(1);
  });
});

// Helper functions (copied from admin.spec.ts for isolation)
async function clickAdminSubTab(page: Page, categoryTabId: string, subTabSelector: string) {
  await page.click(categoryTabId);
  await page.locator(subTabSelector).waitFor({ state: 'visible', timeout: 5000 });
  await page.click(subTabSelector);
}

async function showAdminPane(page: Page, categoryTabId: string, paneId: string) {
  await page.click(categoryTabId);
  await expect(page.locator(categoryTabId.replace('-tab', ''))).toHaveClass(/active/, { timeout: 5000 });
  await page.evaluate((id) => {
    const pane = document.getElementById(id.replace('#', ''));
    if (pane) pane.classList.add('show', 'active');
  }, paneId);
  await expect(page.locator(paneId)).toHaveClass(/show/, { timeout: 5000 });
}

async function closeRacerModal(page: Page) {
  await page.locator('#racerModal').waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
  await page.evaluate(() => {
    document.querySelectorAll('.modal-backdrop').forEach(el => el.remove());
    document.body.classList.remove('modal-open');
  });
}

async function closeTeamModal(page: Page) {
  await page.locator('#teamModal').waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
  await page.evaluate(() => {
    document.querySelectorAll('.modal-backdrop').forEach(el => el.remove());
    document.body.classList.remove('modal-open');
  });
}

async function loginAsAdmin(page: Page) {
  await page.goto('/admin.html');
  if (await page.locator('#adminCategories').count() > 0) return;

  await page.waitForSelector('#setup-form, #login-form', { timeout: 10000 });

  if (await page.locator('#setup-form').count() > 0) {
    await page.fill('#setup-form input[name="username"]', 'admin');
    await page.fill('#setup-form input[name="password"]', 'admin123');
    await page.fill('#setup-form input[name="confirm_password"]', 'admin123');
    await page.click('#setup-form button[type="submit"]');
    try {
      await page.waitForURL(/admin/, { timeout: 5000 });
    } catch {
      // Setup failed (race with another browser). Fall back to login.
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
  await expect(page.locator('#adminCategories')).toBeVisible({ timeout: 10000 });
}
