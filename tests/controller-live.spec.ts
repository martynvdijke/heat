import { test, expect } from './fixtures';

// End-to-end coverage for the controller -> display-page WebSocket path.
//
// The controller publishes flags over HTTP (/api/flags); the server fans them
// out to every WebSocket subscriber. These tests drive the real controller
// buttons in one browser context and assert that a separate display page in
// another context receives the update. The display context deliberately
// carries a stale `session` cookie: an old admin session must not turn the
// handshake into a 401, otherwise TV/start-lights silently reconnect forever
// and never show live flags.

const STALE_SESSION = 'stale-session-does-not-exist';

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

async function createRacer(page: import('@playwright/test').Page, name: string): Promise<number> {
  const created = await page.evaluate(async (racerName) => {
    const res = await fetch('/api/racers', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: 0, name: racerName, profile_picture: '', car_color: '#ff0000',
        car_name: 'Live Routing Car', points: 0, rank: 0, position: 0
      })
    });
    return res.ok;
  }, name);
  expect(created).toBeTruthy();

  const racers = await (await page.request.get('/api/racers')).json();
  const racer = racers.find((r: any) => r.name === name);
  expect(racer).toBeTruthy();
  return racer.id;
}

// A display page in its own browser context, holding a session cookie that no
// longer exists server-side (expired login, restart, other IP). `helloSeen`
// turns true once the server's hello frame arrives, proving the handshake was
// accepted instead of rejected with 401 and retried forever.
async function openDisplayWithStaleSession(
  browser: import('@playwright/test').Browser,
  origin: string,
  path: string
) {
  const context = await browser.newContext();
  await context.addCookies([{ name: 'session', value: STALE_SESSION, url: origin }]);
  const page = await context.newPage();

  let hello = false;
  page.on('websocket', (ws) => {
    if (!ws.url().includes('/ws')) return;
    ws.on('framereceived', (frame) => {
      if (String(frame.payload).includes('"type":"hello"')) hello = true;
    });
  });

  await page.goto(path);
  return { context, page, helloSeen: () => hello };
}

test.describe('Controller live routing', () => {
  test('blue flag reaches the TV page despite a stale session cookie', async ({ page, browser }) => {
    await loginAsAdmin(page);
    await page.goto('/controller.html');
    await expect(page.locator('#controller-start-lights')).toBeVisible();
    await page.waitForFunction(() => (window as any).__controllerWsConnected === true, undefined, { timeout: 10000 });

    const racerID = await createRacer(page, `Blue Flag Racer ${Date.now()}`);

    // The controller dropdown is rebuilt from the broadcast racer list.
    const driverSelect = page.locator('#flag-driver-select');
    await expect(driverSelect.locator(`option[value="${racerID}"]`)).toBeAttached({ timeout: 10000 });
    await driverSelect.selectOption(String(racerID));

    const origin = new URL(page.url()).origin;
    const { context, page: tv, helloSeen } = await openDisplayWithStaleSession(browser, origin, '/tv.html');
    try {
      // The stale cookie must not block the display socket.
      await expect.poll(helloSeen, { timeout: 10000 }).toBe(true);
      await expect(tv.locator('#tv-flag-text')).toBeVisible();

      // The broadcast may race the TV socket's subscribe frame, so retry the
      // action until the display actually reflects it.
      await expect(async () => {
        await page.click('[data-action="sendBlueFlag"]');
        await expect(tv.locator('#tv-flag-text')).toHaveText('Blue Flag', { timeout: 1500 });
      }).toPass({ timeout: 15000 });
    } finally {
      await context.close();
    }
  });

  test('start lights sequence reaches /startlights.html despite a stale session cookie', async ({ page, browser }) => {
    await loginAsAdmin(page);
    await page.goto('/controller.html');
    await expect(page.locator('#controller-start-lights')).toBeVisible();
    await page.waitForFunction(() => (window as any).__controllerWsConnected === true, undefined, { timeout: 10000 });

    const origin = new URL(page.url()).origin;
    // The pretty URL must resolve too, like the other display pages.
    const { context, page: lights, helloSeen } = await openDisplayWithStaleSession(browser, origin, '/startlights.html');
    try {
      await expect(lights).toHaveTitle(/HEAT: Start Lights/);
      await expect.poll(helloSeen, { timeout: 10000 }).toBe(true);
      const status = lights.locator('#start-status-bar');
      await expect(status).toContainText('READY');

      await expect(async () => {
        await page.click('[data-action="triggerStartLights"]');
        await expect(status).not.toContainText('READY', { timeout: 1500 });
      }).toPass({ timeout: 15000 });

      // At least one bulb lights as the sequence runs.
      await expect(lights.locator('#start-light-1 .start-light-bulb.red, #start-light-1 .start-light-bulb.green').first())
        .toBeVisible({ timeout: 10000 });
    } finally {
      await context.close();
    }
  });
});
