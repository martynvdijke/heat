import { test, expect, Page } from '@playwright/test';

test.describe.serial('Admin Extended Features', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test.describe.serial('Team CRUD', () => {
    const teamName = `Team-${Date.now()}`;

    test('should create a team', async ({ page }) => {
      await showAdminPane(page, 'button[data-tab-id="drivers"]', '#teams-pane');
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
      await showAdminPane(page, 'button[data-tab-id="drivers"]', '#teams-pane');
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
      await showAdminPane(page, 'button[data-tab-id="drivers"]', '#teams-pane');
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
    await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');

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
    await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
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
      await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');

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
      await showAdminPane(page, 'button[data-tab-id="drivers"]', '#teams-pane');
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
      await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
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

  test.describe.serial('Round Editing Flow', () => {
    let roundId = 0;
    let seasonId = 0;

    test('should create a round snapshot (draft)', async ({ page }) => {
      // Ensure we have at least one racer
      await clickAdminSubTab(page, 'button[data-tab-id="drivers"]', '#racers-subtab');
      const racerCount = await page.locator('#racer-list tr').count();
      if (racerCount === 0) {
        await page.click('button[hx-get="/api/html/racers/0/edit"]');
        await page.waitForSelector('#racerModal.show');
        await page.fill('form#racer-form input[name="name"]', `Round-Racer-${Date.now()}`);
        await page.fill('form#racer-form input[name="car_name"]', 'RoundCar');
        await page.fill('form#racer-form input[name="car_color"]', '#ff0000');
        await page.fill('form#racer-form input[name="points"]', '10');
        await page.fill('form#racer-form input[name="rank"]', '1');
        await page.fill('form#racer-form input[name="position"]', '0');
        await page.click('form#racer-form button[type="submit"]');
        await page.waitForTimeout(500);
      }

      // Navigate to rounds pane via direct sub-tab click
      await page.click('button[data-tab-id="season"]');
      await page.waitForTimeout(300);
      await page.click('button[data-bs-target="#rounds-pane"]');
      await page.waitForTimeout(500);

      // Create round snapshot via API (bypass prompt dialog)
      const result = await page.evaluate(async () => {
        // Create a fresh season so we don't rely on SeedSeason's season 1
        const seasonRes = await fetch('/api/seasons', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: `E2E-Season-${Date.now()}` })
        });
        const season = await seasonRes.json();
        const sid = season.id;
        const res = await fetch('/api/rounds', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ race_name: `E2E-Round-${Date.now()}`, round: 0, season_id: sid })
        });
        const data = await res.json();
        return { ok: res.ok, id: data.id, seasonId: sid };
      });
      expect(result.ok).toBeTruthy();
      roundId = result.id;
      seasonId = result.seasonId;
    });

    test('should edit round scores', async ({ page }) => {
      expect(roundId).toBeGreaterThan(0);

      // Navigate to rounds pane
      await page.click('button[data-tab-id="season"]');
      await page.waitForTimeout(300);
      await page.click('button[data-bs-target="#rounds-pane"]');
      await page.waitForTimeout(500);

      // Fetch round scores via API
      const scores = await page.evaluate(async (id) => {
        const res = await fetch(`/api/rounds?id=${id}`);
        const round = await res.json();
        return round.scores || [];
      }, roundId);
      expect(scores.length).toBeGreaterThanOrEqual(1);

      // Modify scores via API
      const updated = scores.map((s: any) => ({
        id: s.id,
        points: s.points + 5,
        position: s.position,
        dnf: false,
        dns: false,
        spins: 1
      }));
      const saveRes = await page.evaluate(async (data) => {
        const res = await fetch(`/api/rounds?id=${data.roundId}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(data.scores)
        });
        return res.ok;
      }, { roundId, scores: updated });
      expect(saveRes).toBeTruthy();

      // Verify scores were updated
      const verifyScores = await page.evaluate(async (id) => {
        const res = await fetch(`/api/rounds?id=${id}`);
        const round = await res.json();
        return (round.scores || []).map((s: any) => ({ points: s.points, spins: s.spins }));
      }, roundId);
      expect(verifyScores[0].points).toBe(scores[0].points + 5);
      expect(verifyScores[0].spins).toBe(1);
    });

    test('should finalize round and verify driver points updated', async ({ page }) => {
      expect(roundId).toBeGreaterThan(0);

      // Get racer points before finalizing
      const beforePoints = await page.evaluate(async () => {
        const res = await fetch('/api/racers');
        const racers = await res.json();
        return (racers || []).map((r: any) => ({ id: r.id, name: r.name, points: r.points }));
      });

      // Finalize round
      const finalRes = await page.evaluate(async (id) => {
        const res = await fetch(`/api/rounds/finalize?id=${id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' }
        });
        return res.ok;
      }, roundId);
      expect(finalRes).toBeTruthy();

      // Verify round is now final
      const roundStatus = await page.evaluate(async (id) => {
        const res = await fetch(`/api/rounds?id=${id}`);
        const round = await res.json();
        return round.status;
      }, roundId);
      expect(roundStatus).toBe('final');

      // Verify racer points increased
      const afterPoints = await page.evaluate(async () => {
        const res = await fetch('/api/racers');
        const racers = await res.json();
        return (racers || []).reduce((acc: any, r: any) => { acc[r.id] = r.points; return acc; }, {});
      });

      // Each racer in the round should have gained points
      const roundScores = await page.evaluate(async (id) => {
        const res = await fetch(`/api/rounds?id=${id}`);
        const round = await res.json();
        return round.scores || [];
      }, roundId);

      for (const score of roundScores) {
        const before = beforePoints.find((b: any) => b.id === score.racer_id);
        if (before) {
          expect(afterPoints[score.racer_id]).toBe(before.points + score.points);
        }
      }
    });

    test('should reject editing after finalization', async ({ page }) => {
      // Try to edit the already-finalized round from test #3
      const editResult = await page.evaluate(async () => {
        const roundsRes = await fetch('/api/rounds');
        const rounds = await roundsRes.json();
        // Find a finalized round
        const finalRound = Array.isArray(rounds) ? rounds.find((r: any) => r.status === 'final') : null;
        if (!finalRound) return { error: 'no final round', rounds: JSON.stringify(rounds) };

        // Try to update it
        const scoresRes = await fetch(`/api/rounds?id=${finalRound.id}`);
        const roundDetail = await scoresRes.json();
        const scores = roundDetail.scores || [];
        if (scores.length === 0) return { error: 'no scores' };

        const res = await fetch(`/api/rounds?id=${finalRound.id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify([{ id: scores[0].id, points: 999, position: 1, dnf: false, dns: false, spins: 0 }])
        });
        return { ok: res.ok, status: res.status, errorText: res.statusText };
      });
      expect(editResult.ok).toBeFalsy();
      expect(editResult.status).toBe(409);
    });
  });

  test.describe.serial('Archive Lock - Round Protection', () => {
    let roundId = 0;
    let seasonId = 0;

    test('should create a round in active season', async ({ page }) => {
      // Create a round via API
      const result = await page.evaluate(async () => {
        // Create a fresh season so we don't rely on SeedSeason's season 1
        const seasonRes = await fetch('/api/seasons', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: `Archive-Season-${Date.now()}` })
        });
        const season = await seasonRes.json();
        const sid = season.id;
        const res = await fetch('/api/rounds', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ race_name: `Archive-Round-${Date.now()}`, round: 0, season_id: sid })
        });
        const data = await res.json();
        return { ok: res.ok, id: data.id, seasonId: sid };
      });
      expect(result.ok).toBeTruthy();
      roundId = result.id;
      seasonId = result.seasonId;
    });

    test('should archive the season', async ({ page }) => {
      const result = await page.evaluate(async (sid) => {
        const res = await fetch(`/api/seasons/archive?id=${sid}`, { method: 'POST' });
        return res.ok;
      }, seasonId);
      expect(result).toBeTruthy();
    });

    test('should reject creating rounds in archived season', async ({ page }) => {
      const result = await page.evaluate(async (sid) => {
        const res = await fetch('/api/rounds', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ race_name: 'Should Fail', round: 0, season_id: sid })
        });
        return { status: res.status };
      }, seasonId);
      expect(result.status).toBe(409);
    });

    test('should reject editing rounds in archived season', async ({ page }) => {
      const result = await page.evaluate(async (rid) => {
        const scoresRes = await fetch(`/api/rounds?id=${rid}`);
        const round = await scoresRes.json();
        const scores = round.scores || [];
        if (scores.length === 0) return { status: -1, error: 'no scores' };

        const res = await fetch(`/api/rounds?id=${rid}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(scores.slice(0, 1).map((s: any) => ({ ...s, points: 999 })))
        });
        return { status: res.status };
      }, roundId);
      expect(result.status).toBe(409);
    });

    test('should reject finalizing rounds in archived season', async ({ page }) => {
      const result = await page.evaluate(async (rid) => {
        const res = await fetch(`/api/rounds/finalize?id=${rid}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' }
        });
        return { status: res.status };
      }, roundId);
      expect(result.status).toBe(409);
    });

    test('should reject deleting rounds in archived season', async ({ page }) => {
      const result = await page.evaluate(async (rid) => {
        const res = await fetch(`/api/rounds?id=${rid}`, { method: 'DELETE' });
        return { status: res.status };
      }, roundId);
      expect(result.status).toBe(409);
    });

    test('should verify round still exists after delete attempt', async ({ page }) => {
      const result = await page.evaluate(async (rid) => {
        const res = await fetch(`/api/rounds?id=${rid}`);
        return { status: res.status, ok: res.ok };
      }, roundId);
      expect(result.ok).toBeTruthy();
    });
  });
});

// Helper functions (copied from admin.spec.ts for isolation)
async function clickAdminSubTab(page: Page, tabSelector: string, subTabSelector: string) {
  await page.click(tabSelector);
  await page.locator(subTabSelector).waitFor({ state: 'visible', timeout: 10000 });
  await page.click(subTabSelector);
}

async function showAdminPane(page: Page, tabSelector: string, paneId: string) {
  await page.click(tabSelector);
  await expect(page.locator(paneId)).toBeAttached({ timeout: 10000 });
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
  await expect(page.locator('#admin-nav')).toBeVisible({ timeout: 10000 });
}
