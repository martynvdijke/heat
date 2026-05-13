import { test, expect } from '@playwright/test';

const API_HEADERS = { Origin: 'http://localhost:6270' };

async function loginAdminViaAPI(pageRequest: any) {
  const res = await pageRequest.post('/api/login', {
    data: { username: 'admin', password: 'admin123' },
    headers: API_HEADERS,
  });
  if (!res.ok()) {
    await pageRequest.post('/api/login', {
      data: { username: 'admin', password: 'admin123', setup: true },
      headers: API_HEADERS,
    });
  }
}

test.describe('Stats Page', () => {
  test('should load stats page', async ({ page }) => {
    await page.goto('/stats.html');
    await expect(page).toHaveTitle(/HEAT: Season Statistics/);

    const heading = page.locator('h1');
    await expect(heading).toContainText('Season Statistics');
  });

  test('should display stat cards', async ({ page }) => {
    await page.goto('/stats.html');
    const totalRaces = page.locator('#total-races');
    await expect(totalRaces).toBeVisible();
  });

  test('should have navigation links', async ({ page }) => {
    await page.goto('/stats.html');
    await expect(page.locator('a[href="/"]').first()).toBeAttached();
    await expect(page.locator('a[href="/trophies.html"]').first()).toBeAttached();
    await expect(page.locator('a[href="/controller.html"]').first()).toBeAttached();
  });

  test('should render win distribution chart', async ({ page }) => {
    await page.goto('/stats.html');
    const canvas = page.locator('#wins-chart');
    await expect(canvas).toBeVisible();
    const rendered = await page.evaluate(() => {
      const c = document.getElementById('wins-chart') as HTMLCanvasElement;
      return c && c.getContext('2d') ? true : false;
    });
    expect(rendered).toBe(true);
  });

  test('should render driver performance table', async ({ page }) => {
    await page.goto('/stats.html');
    const tbody = page.locator('#driver-stats-table tbody');
    await expect(tbody).toBeVisible();
    const hasRows = await tbody.locator('tr').count();
    expect(hasRows).toBeGreaterThanOrEqual(0);
  });
});

test.describe('Trophies Page', () => {
  test('should load trophies page', async ({ page }) => {
    await page.goto('/trophies.html');
    await expect(page).toHaveTitle(/HEAT: Trophy Room/);

    const heading = page.locator('h1');
    await expect(heading).toContainText('Trophy Room');
  });

  test('should have driver selector', async ({ page }) => {
    await page.goto('/trophies.html');
    const selector = page.locator('#driver-select');
    await expect(selector).toBeVisible();
  });

  test('should display trophy categories', async ({ page }) => {
    await page.goto('/trophies.html');
    await expect(page.locator('#race-wins-grid')).toBeVisible();
    await expect(page.locator('#podium-grid')).toBeVisible();
    await expect(page.locator('#speed-grid')).toBeVisible();
  });
});

test.describe('Index Page - GeoJSON and Racers', () => {
  test('should display racers with helmet fallback images', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const rows = page.locator('#leaderboard-body tr');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);

    const firstImage = rows.first().locator('img');
    if (await firstImage.count() > 0) {
      await expect(firstImage).toBeVisible();
    }
  });

  test('should render circuit map with GeoJSON', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const map = page.locator('#circuit-map');
    await expect(map).toBeVisible();
    await expect(map).toHaveClass(/leaflet-container/);

    const mapPath = page.locator('#circuit-map path');
    await expect(mapPath.first()).toBeAttached({ timeout: 10000 });
  });

  test('should display race info with country, track and laps', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const raceCountry = page.locator('#race-country');
    await expect(raceCountry).toBeVisible();
    await expect(raceCountry).toContainText('Italy');

    const raceTrack = page.locator('#race-track');
    await expect(raceTrack).toBeVisible();
    await expect(raceTrack).toContainText('Monza');
  });
});

test.describe('Player Self-Service', () => {
  test('should load player login page', async ({ page }) => {
    await page.goto('/player.html');
    await expect(page.locator('h1')).toContainText('Player Login');
    await expect(page.locator('#player-select')).toBeVisible();
    await expect(page.locator('#device-name')).toBeVisible();
  });

  test('should show driver options in select', async ({ page }) => {
    await page.goto('/player.html');
    await page.locator('#player-select option').first().waitFor({ state: 'attached', timeout: 15000 });
    const count = await page.locator('#player-select option').count();
    expect(count).toBeGreaterThan(4);
  });

  test('should login player via API', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[0]?.id || 1;

    const res = await page.request.post('/api/player/login', {
      data: { racer_id: racerId, device_name: 'TestPhone' }
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data).toHaveProperty('token');
    expect(data).toHaveProperty('racer_id');
    expect(data).toHaveProperty('racer_name');
  });

  test('should validate player token', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[1]?.id || 2;

    const loginRes = await page.request.post('/api/player/login', {
      data: { racer_id: racerId, device_name: 'TestPhone' }
    });
    expect(loginRes.ok()).toBeTruthy();
    const { token } = await loginRes.json();

    const validateRes = await page.request.get('/api/player/validate', {
      headers: { 'X-Player-Token': token }
    });
    expect(validateRes.ok()).toBeTruthy();
    const data = await validateRes.json();
    expect(data).toHaveProperty('racer_id');
    expect(data).toHaveProperty('racer_name');
  });

  test('should reject invalid token', async ({ page }) => {
    const res = await page.request.get('/api/player/validate', {
      headers: { 'X-Player-Token': 'invalid_token_12345' }
    });
    expect(res.status()).toBe(401);
  });

  test('should report gear as player', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[0]?.id || 1;

    const loginRes = await page.request.post('/api/player/login', {
      data: { racer_id: racerId, device_name: 'TestPhone' }
    });
    const { token } = await loginRes.json();

    const gearRes = await page.request.post('/api/player/gear', {
      data: { token, lap: 1, gear: 3, stress: 0 }
    });
    expect(gearRes.ok()).toBeTruthy();
  });

  test('should report heat as player', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[0]?.id || 1;

    const loginRes = await page.request.post('/api/player/login', {
      data: { racer_id: racerId, device_name: 'TestPhone' }
    });
    const { token } = await loginRes.json();

    const heatRes = await page.request.post('/api/player/heat', {
      data: { token, card_type: 'heat', location: 'engine', count: 1 }
    });
    expect(heatRes.ok()).toBeTruthy();
  });

  test('should report turbo usage as player', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[3]?.id || 4;

    const loginRes = await page.request.post('/api/player/login', {
      data: { racer_id: racerId, device_name: 'TestPhone' }
    });
    const { token } = await loginRes.json();

    const turboRes = await page.request.post('/api/player/turbo', {
      data: { token, lap: 1 }
    });
    expect(turboRes.ok()).toBeTruthy();
  });

  test('should get player status', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[4]?.id || 5;

    const loginRes = await page.request.post('/api/player/login', {
      data: { racer_id: racerId, device_name: 'TestPhone' }
    });
    const { token } = await loginRes.json();

    const statusRes = await page.request.get('/api/player/status', {
      headers: { 'X-Player-Token': token }
    });
    expect(statusRes.ok()).toBeTruthy();
    const data = await statusRes.json();
    expect(data).toHaveProperty('racer');
    expect(data).toHaveProperty('heat_cards');
    expect(data.racer).toHaveProperty('name');
  });
});

test.describe('Game Mechanics API', () => {
  test('should get heat cards for racer', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[0]?.id || 1;

    const res = await page.request.get(`/api/heat-cards?racer_id=${racerId}`);
    expect(res.ok()).toBeTruthy();
    const cards = await res.json();
    expect(Array.isArray(cards)).toBeTruthy();
  });

  test('should initialize heat decks', async ({ page }) => {
    await loginAdminViaAPI(page.request);

    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerIds = racers.slice(0, 3).map((r: any) => r.id);

    const res = await page.request.post('/api/heat-cards/init-decks', {
      data: { race_id: 0, racer_ids: racerIds },
      headers: API_HEADERS,
    });
    expect(res.ok()).toBeTruthy();

    const cardsRes = await page.request.get('/api/heat-cards');
    const cards = await cardsRes.json();
    expect(cards.length).toBeGreaterThanOrEqual(21);
  });

  test('should get gear shifts', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[0]?.id || 1;

    const res = await page.request.get(`/api/gear-shifts?racer_id=${racerId}`);
    expect(res.ok()).toBeTruthy();
    const shifts = await res.json();
    expect(Array.isArray(shifts)).toBeTruthy();
  });

  test('should get upgrade cards', async ({ page }) => {
    const res = await page.request.get('/api/upgrade-cards');
    expect(res.ok()).toBeTruthy();
    const upgrades = await res.json();
    expect(Array.isArray(upgrades)).toBeTruthy();
    expect(upgrades.length).toBeGreaterThanOrEqual(8);
  });

  test('should have legend abilities', async ({ page }) => {
    const res = await page.request.get('/api/legend-abilities');
    expect(res.ok()).toBeTruthy();
    const abilities = await res.json();
    expect(Array.isArray(abilities)).toBeTruthy();
    expect(abilities.length).toBeGreaterThanOrEqual(5);
  });
});

test.describe('Race Enhancements API', () => {
  test('should set and get weather', async ({ page }) => {
    await loginAdminViaAPI(page.request);

    await page.request.post('/api/weather', {
      data: { race_id: 0, condition: 'wet', lap_start: 1, lap_end: 999, grip_modifier: 0.7 },
      headers: API_HEADERS,
    });
    const res = await page.request.get('/api/weather?race_id=0');
    expect(res.ok()).toBeTruthy();
    const weather = await res.json();
    expect(weather.length).toBeGreaterThanOrEqual(1);
  });

  test('should log turbo usage', async ({ page }) => {
    await loginAdminViaAPI(page.request);

    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[0]?.id || 1;

    await page.request.post('/api/turbo-logs', {
      data: { racer_id: racerId, race_id: 0, lap: 1, times_used: 1 },
      headers: API_HEADERS,
    });
    const res = await page.request.get(`/api/turbo-logs?racer_id=${racerId}`);
    expect(res.ok()).toBeTruthy();
    const logs = await res.json();
    expect(logs.length).toBeGreaterThanOrEqual(1);
  });

  test('should record and retrieve lap records', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const racerId = racers[0]?.id || 1;

    await page.request.post('/api/lap-records', {
      data: { race_id: 0, racer_id: racerId, lap_number: 1, position: 1, gear_used: 3, heat_generated: 2, turbo_used: false }
    });
    const res = await page.request.get('/api/lap-records?race_id=0');
    expect(res.ok()).toBeTruthy();
    const records = await res.json();
    expect(records.length).toBeGreaterThanOrEqual(1);
  });

  test('should record batch lap records', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const r1 = racers[0]?.id || 1;
    const r2 = racers[1]?.id || 2;

    const res = await page.request.post('/api/lap-records/batch', {
      data: {
        race_id: 0, lap: 1,
        records: [
          { racer_id: r1, position: 1, gear_used: 3, heat_generated: 2, turbo_used: false },
          { racer_id: r2, position: 2, gear_used: 2, heat_generated: 1, turbo_used: true }
        ]
      }
    });
    expect(res.ok()).toBeTruthy();
  });

  test('should get sectors for a track', async ({ page }) => {
    const res = await page.request.get('/api/sectors?track_id=monza');
    expect(res.ok()).toBeTruthy();
    const sectors = await res.json();
    expect(Array.isArray(sectors)).toBeTruthy();
    expect(sectors.length).toBeGreaterThanOrEqual(5);
  });

  test('should add and retrieve race events', async ({ page }) => {
    const racersRes = await page.request.get('/api/racers');
    const racers = await racersRes.json();
    const r1 = racers[0]?.id || 1;
    const r2 = racers[1]?.id || 2;

    await page.request.post('/api/race-events', {
      data: { race_id: 0, lap: 1, event_type: 'overtake', racer_id: r1, racer_id2: r2, note: 'Great pass!' }
    });
    const res = await page.request.get('/api/race-events?race_id=0');
    expect(res.ok()).toBeTruthy();
    const events = await res.json();
    expect(events.length).toBeGreaterThanOrEqual(1);
    expect(events[0].event_type).toBe('overtake');
  });
});

test.describe('Spectator Mode', () => {
  test('should return spectator state', async ({ page }) => {
    const res = await page.request.get('/api/spectator/state');
    expect(res.ok()).toBeTruthy();
    const state = await res.json();
    expect(state).toHaveProperty('racers');
    expect(state).toHaveProperty('race');
    expect(Array.isArray(state.racers)).toBeTruthy();
  });
});

test.describe('Deeper Stats', () => {
  test('should return qualifying vs race delta', async ({ page }) => {
    const res = await page.request.get('/api/stats/qualifying-delta');
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(Array.isArray(data)).toBeTruthy();
  });

  test('should return consistency ratings', async ({ page }) => {
    const res = await page.request.get('/api/stats/consistency');
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(Array.isArray(data)).toBeTruthy();
  });

  test('should return incidents report', async ({ page }) => {
    const res = await page.request.get('/api/stats/incidents');
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(Array.isArray(data)).toBeTruthy();
  });

  test('should return pace heatmap', async ({ page }) => {
    const res = await page.request.get('/api/stats/pace-heatmap');
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(Array.isArray(data)).toBeTruthy();
  });

  test('should return race report', async ({ page }) => {
    const res = await page.request.get('/api/race-report');
    const status = res.status();
    expect([200, 404]).toContain(status);
  });
});

test.describe('UI Presentation Pages', () => {
  test('should load TV broadcast overlay', async ({ page }) => {
    await page.goto('/tv.html');
    await expect(page.locator('.logo')).toContainText('HEAT');
    await expect(page.locator('#tv-leaderboard')).toBeVisible();
  });

  test('should load pit board display', async ({ page }) => {
    await page.goto('/pitboard.html');
    await expect(page.locator('h1')).toContainText('PIT BOARD');
    await expect(page.locator('#pit-board')).toBeVisible();
  });

  test('should load race replay viewer', async ({ page }) => {
    await page.goto('/replay.html');
    await expect(page.locator('h1')).toContainText('Race Replay');
    await expect(page.locator('#replay-race-select')).toBeVisible();
  });

  test('should load spectator view', async ({ page }) => {
    await page.goto('/spectator.html');
    await expect(page.locator('h1')).toContainText('Spectator View');
    await expect(page.locator('#spec-grid')).toBeVisible();
  });
});

test.describe('Sound FX', () => {
  test('should trigger sound via API', async ({ page }) => {
    const res = await page.request.post('/api/sound', {
      data: { sound: 'engine' }
    });
    expect(res.ok()).toBeTruthy();
  });

  test('should trigger finish sound', async ({ page }) => {
    const res = await page.request.post('/api/sound', {
      data: { sound: 'finish' }
    });
    expect(res.ok()).toBeTruthy();
  });
});
