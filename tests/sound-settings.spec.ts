import { test, expect } from '@playwright/test';

test.describe('Sound Settings — Storage & Validation', () => {
  test('should expose sound settings helpers on the TV page', async ({ page }) => {
    await page.goto('/tv.html');
    const hasHelpers = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      return !!(s && typeof s.loadSettings === 'function' && typeof s.saveSettings === 'function'
        && typeof s.resetSettings === 'function' && typeof s.getVolume === 'function'
        && typeof s.validateUpload === 'function' && typeof s.readFileAsDataUrl === 'function');
    });
    expect(hasHelpers).toBe(true);
  });

  test('should clamp volumes to 0..1 and merge defaults', async ({ page }) => {
    await page.goto('/tv.html');
    const result = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      localStorage.removeItem('heat.soundSettings.v1');
      // Corrupt / out-of-range stored values
      localStorage.setItem('heat.soundSettings.v1', JSON.stringify({
        volumes: { engine: 5, horn: -2, finish: 0.5 },
        overrides: { engine: { name: 'x.mp3', dataUrl: 'data:audio/mpeg;base64,AAAA' } }
      }));
      const loaded = s.loadSettings();
      return {
        engine: loaded.volumes.engine,
        horn: loaded.volumes.horn,
        finish: loaded.volumes.finish,
        crash: loaded.volumes.crash,
        overrideName: loaded.overrides.engine?.name
      };
    });
    expect(result.engine).toBe(1);   // clamped from 5
    expect(result.horn).toBe(0);     // clamped from -2
    expect(result.finish).toBe(0.5); // kept
    expect(result.crash).toBe(1);    // default merged
    expect(result.overrideName).toBe('x.mp3');
  });

  test('should fall back to defaults on corrupt JSON', async ({ page }) => {
    await page.goto('/tv.html');
    const result = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      localStorage.setItem('heat.soundSettings.v1', '{not valid json');
      const loaded = s.loadSettings();
      return { engine: loaded.volumes.engine, overrides: Object.keys(loaded.overrides).length };
    });
    expect(result.engine).toBe(1);
    expect(result.overrides).toBe(0);
  });

  test('should validate uploads by MIME type and size', async ({ page }) => {
    await page.goto('/tv.html');
    const result = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      const badMime = new File(['x'], 'a.txt', { type: 'text/plain' });
      const goodMime = new File(['x'], 'a.mp3', { type: 'audio/mpeg' });
      const tooBig = new File([new Uint8Array(s.MAX_UPLOAD_BYTES + 1)], 'big.mp3', { type: 'audio/mpeg' });
      return {
        badMime: s.validateUpload(badMime),
        goodMime: s.validateUpload(goodMime),
        tooBig: s.validateUpload(tooBig)
      };
    });
    expect(result.badMime).toBeTruthy();
    expect(result.goodMime).toBeNull();
    expect(result.tooBig).toBeTruthy();
  });

  test('should round-trip an override through save/load', async ({ page }) => {
    await page.goto('/tv.html');
    const result = await page.evaluate(async () => {
      const s = (window as any).soundSettings;
      s.resetSettings();
      const settings = s.loadSettings();
      settings.overrides.engine = { name: 'custom.mp3', dataUrl: 'data:audio/mpeg;base64,AAAA' };
      settings.volumes.engine = 0.3;
      s.saveSettings(settings);
      const reloaded = s.loadSettings();
      return {
        name: reloaded.overrides.engine?.name,
        dataUrl: reloaded.overrides.engine?.dataUrl,
        volume: reloaded.volumes.engine
      };
    });
    expect(result.name).toBe('custom.mp3');
    expect(result.dataUrl).toBe('data:audio/mpeg;base64,AAAA');
    expect(result.volume).toBe(0.3);
  });

  test('should reset to defaults', async ({ page }) => {
    await page.goto('/tv.html');
    const result = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      s.saveSettings({
        volumes: { engine: 0, horn: 0, finish: 0, crash: 0 },
        overrides: { engine: { name: 'x.mp3', dataUrl: 'data:audio/mpeg;base64,AAAA' } }
      });
      s.resetSettings();
      const loaded = s.loadSettings();
      return { engine: loaded.volumes.engine, overrides: Object.keys(loaded.overrides).length };
    });
    expect(result.engine).toBe(1);
    expect(result.overrides).toBe(0);
  });
});

test.describe('Sound Settings — Modal UI', () => {
  test('should open modal from trigger and close with Esc', async ({ page }) => {
    await page.goto('/tv.html');
    const trigger = page.locator('#sound-settings-trigger');
    await expect(trigger).toBeVisible();
    await expect(trigger).toHaveAttribute('aria-haspopup', 'dialog');

    const modal = page.locator('#sound-settings-modal');
    await expect(modal).toBeHidden();

    await trigger.click();
    await expect(modal).toBeVisible();
    await expect(modal).toHaveAttribute('role', 'dialog');
    await expect(page.locator('#sound-settings-rows .sound-setting-row')).toHaveCount(4);

    await page.keyboard.press('Escape');
    await expect(modal).toBeHidden();
  });

  test('should persist volume slider changes', async ({ page }) => {
    await page.goto('/tv.html');
    await page.locator('#sound-settings-trigger').click();
    const slider = page.locator('#sound-vol-engine');
    await slider.fill('0');
    await expect(page.locator('#sound-vol-readout-engine')).toHaveText('0%');

    const stored = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      return s.getVolume('engine');
    });
    expect(stored).toBe(0);
  });

  test('should not play audio when engine volume is zero', async ({ page }) => {
    await page.goto('/tv.html');
    // Set engine volume to 0 via the modal
    await page.locator('#sound-settings-trigger').click();
    await page.locator('#sound-vol-engine').fill('0');
    await page.keyboard.press('Escape');

    // Instrument audio playback
    await page.evaluate(() => {
      (window as any).__audioStarts = 0;
      const proto = (window as any).OscillatorNode?.prototype;
      if (proto && proto.start) {
        const origStart = proto.start;
        proto.start = function (...args: any[]) {
          (window as any).__audioStarts++;
          return origStart.apply(this, args);
        };
      }
    });

    // Trigger engine sound via the API broadcast
    const res = await page.request.post('/api/sound', { data: { sound: 'engine' } });
    expect(res.ok()).toBeTruthy();

    await page.waitForTimeout(500);
    const starts = await page.evaluate(() => (window as any).__audioStarts || 0);
    expect(starts).toBe(0);
  });

  test('should store an uploaded override and replay it after reload', async ({ page }) => {
    await page.goto('/tv.html');
    await page.locator('#sound-settings-trigger').click();

    // Build a tiny valid WAV data URL in the test
    const wavDataUrl = await page.evaluate(() => {
      const sampleRate = 8000;
      const seconds = 0.1;
      const numSamples = Math.floor(sampleRate * seconds);
      const buffer = new ArrayBuffer(44 + numSamples * 2);
      const view = new DataView(buffer);
      const writeStr = (offset: number, str: string) => {
        for (let i = 0; i < str.length; i++) view.setUint8(offset + i, str.charCodeAt(i));
      };
      writeStr(0, 'RIFF');
      view.setUint32(4, 36 + numSamples * 2, true);
      writeStr(8, 'WAVE');
      writeStr(12, 'fmt ');
      view.setUint32(16, 16, true);
      view.setUint16(20, 1, true);
      view.setUint16(22, 1, true);
      view.setUint32(24, sampleRate, true);
      view.setUint32(28, sampleRate * 2, true);
      view.setUint16(32, 2, true);
      view.setUint16(34, 16, true);
      writeStr(36, 'data');
      view.setUint32(40, numSamples * 2, true);
      for (let i = 0; i < numSamples; i++) view.setInt16(44 + i * 2, 0, true);
      const bytes = new Uint8Array(buffer);
      let binary = '';
      for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
      return 'data:audio/wav;base64,' + btoa(binary);
    });

    // Upload via the file input
    await page.locator('.sound-file-input[data-category="engine"]').setInputFiles({
      name: 'custom.wav',
      mimeType: 'audio/wav',
      buffer: Buffer.from(wavDataUrl.split(',')[1], 'base64')
    });

    await expect(page.locator('#sound-vol-readout-engine')).toBeVisible();
    const stored = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      return s.loadSettings().overrides.engine;
    });
    expect(stored).toBeTruthy();
    expect(stored.name).toBe('custom.wav');

    // Reload and confirm the override persists
    await page.reload();
    const afterReload = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      return s.loadSettings().overrides.engine;
    });
    expect(afterReload).toBeTruthy();
    expect(afterReload.name).toBe('custom.wav');
  });

  test('should remove an override and reset all', async ({ page }) => {
    await page.goto('/tv.html');
    await page.evaluate(() => {
      const s = (window as any).soundSettings;
      s.saveSettings({
        volumes: { engine: 0.5, horn: 1, finish: 1, crash: 1 },
        overrides: { engine: { name: 'x.mp3', dataUrl: 'data:audio/mpeg;base64,AAAA' } }
      });
    });

    await page.locator('#sound-settings-trigger').click();
    await page.locator('.sound-remove-btn[data-category="engine"]').click();
    const afterRemove = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      return s.loadSettings().overrides.engine;
    });
    expect(afterRemove).toBeUndefined();

    await page.locator('.sound-settings-reset').click();
    const afterReset = await page.evaluate(() => {
      const s = (window as any).soundSettings;
      return s.loadSettings().volumes.engine;
    });
    expect(afterReset).toBe(1);
  });
});