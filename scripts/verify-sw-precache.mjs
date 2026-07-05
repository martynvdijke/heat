// verify-sw-precache.mjs — CI guard that checks every precached URL in sw.js
// exists as a real file on disk. Run after `npm run build:frontend`.
//
// This prevents the bug where a stale hash in the precache list causes
// the service worker install to fail (cache.addAll rejects on 404).

import fs from 'fs';
import path from 'path';

const swPath = 'static/sw.js';
const manifestPath = 'static/vendor/manifest.json';

if (!fs.existsSync(swPath)) {
  console.error(`❌ ${swPath} not found. Run 'npm run build:frontend' first.`);
  process.exit(1);
}

if (!fs.existsSync(manifestPath)) {
  console.error(`❌ ${manifestPath} not found. Run 'npm run build:frontend' first.`);
  process.exit(1);
}

// Extract PRECACHE_URLS from sw.js
const swContent = fs.readFileSync(swPath, 'utf8');
const match = swContent.match(/const PRECACHE_URLS\s*=\s*(\[[\s\S]*?\]);/);

if (!match) {
  console.error('❌ Could not extract PRECACHE_URLS from sw.js');
  process.exit(1);
}

let precacheUrls;
try {
  precacheUrls = JSON.parse(match[1]);
} catch {
  console.error('❌ Failed to parse PRECACHE_URLS array. Check sw.js formatting.');
  process.exit(1);
}

if (!Array.isArray(precacheUrls) || precacheUrls.length === 0) {
  console.error('❌ PRECACHE_URLS is empty or not an array');
  process.exit(1);
}

// Verify each URL corresponds to a real file on disk
let allOk = true;
for (const url of precacheUrls) {
  // URL like "/static/vendor/admin-nav.abc123.css" → "static/vendor/admin-nav.abc123.css"
  const filePath = url.startsWith('/') ? url.slice(1) : url;
  const fullPath = path.resolve(filePath);

  if (!fs.existsSync(fullPath)) {
    console.error(`❌ MISSING: ${url} → ${fullPath} not found on disk`);
    allOk = false;
  } else {
    console.log(`✓ ${url} — OK`);
  }
}

if (!allOk) {
  console.error('\n❌ Some precache URLs point to nonexistent files.');
  console.error('   Run `npm run build:frontend` to regenerate sw.js from manifest.json.');
  process.exit(1);
}

// Cross-check: verify sw.js URLs match the values in manifest.json
const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
const manifestValues = new Set(Object.values(manifest).flatMap(v => Array.isArray(v) ? v : [v]));
for (const url of precacheUrls) {
  if (!manifestValues.has(url)) {
    console.error(`❌ CROSS-CHECK FAILED: ${url} is in sw.js but not in manifest.json`);
    allOk = false;
  }
}
console.log(`✓ Cross-check: All sw.js URLs present in manifest.json`);

// Also verify cache version is bumped past v3
const cacheVersionMatch = swContent.match(/const CACHE\s*=\s*"([^"]+)"/);
if (cacheVersionMatch) {
  const version = cacheVersionMatch[1];
  if (version === 'heat-cache-v3') {
    console.error('❌ Cache version is still v3 — bump to v4');
    process.exit(1);
  }
  console.log(`✓ Cache version: ${version}`);
}

if (!allOk) {
  console.error('\n❌ Some checks failed. Fix the issues above before committing.');
  process.exit(1);
}

console.log(`\n✅ All ${precacheUrls.length} precache URLs verified on disk.`);
