// ws.test.mjs — unit tests for the shared WebSocket helpers in ts/ws.ts.
// Bundles the TS source to ESM via esbuild (the repo's own toolchain).
// Run with: node --test tests/ws.test.mjs
import { test, before } from 'node:test';
import assert from 'node:assert/strict';
import { build } from 'esbuild';

let backoffDelay;
let detectGap;
let mergeById;

before(async () => {
  const result = await build({
    entryPoints: ['ts/ws.ts'],
    bundle: true,
    format: 'esm',
    write: false,
    logLevel: 'silent',
  });
  const code = result.outputFiles[0].text;
  const dataUrl = 'data:text/javascript;base64,' + Buffer.from(code).toString('base64');
  ({ backoffDelay, detectGap, mergeById } = await import(dataUrl));
});

test('backoffDelay uses full jitter within [0, window)', () => {
  assert.equal(backoffDelay(0, 500, 30000, () => 0), 0);
  assert.equal(backoffDelay(0, 500, 30000, () => 0.5), 250);
  assert.equal(backoffDelay(0, 500, 30000, () => 0.999), 499);
});

test('backoffDelay grows exponentially with the attempt counter', () => {
  // window = min(cap, base * 2^attempt): 500, 1000, 2000, ...
  assert.equal(backoffDelay(1, 500, 30000, () => 0.5), 500);
  assert.equal(backoffDelay(2, 500, 30000, () => 0.5), 1000);
  assert.equal(backoffDelay(3, 500, 30000, () => 0.5), 2000);
});

test('backoffDelay is capped and never exceeds the cap', () => {
  // 2^10 * 500 = 512000 -> clamped to cap 30000
  assert.equal(backoffDelay(10, 500, 30000, () => 0.999), 29970);
  assert.equal(backoffDelay(50, 500, 30000, () => 0.999), 29970);
});

test('detectGap only fires when seq skipped ahead', () => {
  assert.equal(detectGap(undefined, 5), false);
  assert.equal(detectGap(5, undefined), false);
  assert.equal(detectGap(5, 5), false);
  assert.equal(detectGap(5, 6), false);
  assert.equal(detectGap(5, 7), true);
  assert.equal(detectGap(5, 100), true);
});

test('mergeById appends only unseen ids and preserves existing order', () => {
  const existing = [{ id: 1 }, { id: 2 }];
  const incoming = [{ id: 2 }, { id: 3 }, { id: 1 }, { id: 4 }];
  assert.deepEqual(mergeById(existing, incoming), [{ id: 1 }, { id: 2 }, { id: 3 }, { id: 4 }]);
});

test('mergeById is idempotent when applied twice', () => {
  const once = mergeById([{ id: 1 }], [{ id: 2 }, { id: 3 }]);
  const twice = mergeById(once, [{ id: 2 }, { id: 3 }]);
  assert.deepEqual(twice, [{ id: 1 }, { id: 2 }, { id: 3 }]);
});

test('mergeById on an empty existing list returns the incoming list', () => {
  assert.deepEqual(mergeById([], [{ id: 7 }]), [{ id: 7 }]);
});
