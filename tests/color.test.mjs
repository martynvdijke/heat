// color.test.mjs — unit tests for ts/color.ts normalizeHex.
// Bundles the TS source to ESM via esbuild (the repo's own toolchain) so the
// tests exercise the real implementation regardless of package.json module type.
// Run with: node --test tests/color.test.mjs
import { test, before } from 'node:test';
import assert from 'node:assert/strict';
import { build } from 'esbuild';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
const esbuildPath = require.resolve('esbuild');

let normalizeHex;

before(async () => {
  const result = await build({
    entryPoints: ['ts/color.ts'],
    bundle: true,
    format: 'esm',
    write: false,
    logLevel: 'silent',
  });
  const code = result.outputFiles[0].text;
  const dataUrl = 'data:text/javascript;base64,' + Buffer.from(code).toString('base64');
  normalizeHex = (await import(dataUrl)).normalizeHex;
});

const NAMED = {
  red: '#ff4444',
  blue: '#4444ff',
  green: '#44ff44',
  yellow: '#ffff44',
  grey: '#aaaaaa',
  silver: '#aaaaaa',
  black: '#333333',
  purple: '#9b59b6',
  orange: '#e67e22',
};

test('normalizes all 9 named colors to their mapped hex', () => {
  for (const [name, hex] of Object.entries(NAMED)) {
    assert.equal(normalizeHex(name), hex, `named color '${name}'`);
  }
});

test('normalizes named colors case-insensitively', () => {
  assert.equal(normalizeHex('RED'), '#ff4444');
  assert.equal(normalizeHex('Silver'), '#aaaaaa');
});

test('passes through valid 6-digit hex with # prefix', () => {
  assert.equal(normalizeHex('#800080'), '#800080');
});

test('adds # prefix to bare 6-digit hex', () => {
  assert.equal(normalizeHex('800080'), '#800080');
});

test('falls back for empty string', () => {
  assert.equal(normalizeHex(''), '#cccccc');
});

test('falls back for non-color input', () => {
  assert.equal(normalizeHex('not-a-color'), '#cccccc');
});

test('falls back for 3-digit hex (spec requires 6-digit)', () => {
  assert.equal(normalizeHex('#fff'), '#cccccc');
});

test('falls back for 8-digit hex with alpha', () => {
  assert.equal(normalizeHex('#800080ff'), '#cccccc');
});

test('trims surrounding whitespace before matching', () => {
  assert.equal(normalizeHex('  red  '), '#ff4444');
  assert.equal(normalizeHex('  #800080  '), '#800080');
});
