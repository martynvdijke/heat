import * as esbuild from 'esbuild';
import fs from 'fs';
import path from 'path';
import crypto from 'crypto';

const isDev = process.argv.includes('--dev');

// Shared modules that are imported by page entry points (not standalone entry points)
const sharedModules = new Set(['toast.ts', 'i18n.ts', 'theme.ts', 'color.ts', 'sound.ts', 'sound-settings.ts', 'commentary.ts', 'startlights-core.ts', 'weather.ts']);

// Entry points: all TS files in ts/ except shared modules
const entryPoints = fs.readdirSync('ts')
  .filter(f => f.endsWith('.ts') && !sharedModules.has(f))
  .map(f => `ts/${f}`);

const config = {
  entryPoints,
  outdir: 'static/js',
  bundle: true,
  format: 'iife',
  target: 'es2020',
  sourcemap: isDev,
  minify: !isDev,
  outbase: 'ts',
};

// --- Vendor bundling: self-host Bootstrap + FontAwesome with content-hashed filenames ---
function hashContent(buf) {
  return crypto.createHash('sha256').update(buf).digest('hex').slice(0, 12);
}

async function buildVendor() {
  const outDir = 'static/vendor';
  fs.rmSync(outDir, { recursive: true, force: true });
  fs.mkdirSync(path.join(outDir, 'webfonts'), { recursive: true });

  const manifest = {};

  // Bootstrap CSS (only inline data: URIs — safe to copy verbatim)
  const bsCssSrc = fs.readFileSync('node_modules/bootstrap/dist/css/bootstrap.min.css');
  const bsCssHash = hashContent(bsCssSrc);
  const bsCssName = `bootstrap.${bsCssHash}.min.css`;
  fs.writeFileSync(path.join(outDir, bsCssName), bsCssSrc);
  manifest.bootstrapCss = `/static/vendor/${bsCssName}`;

  // Bootstrap JS bundle (includes Popper)
  const bsJsSrc = fs.readFileSync('node_modules/bootstrap/dist/js/bootstrap.bundle.min.js');
  const bsJsHash = hashContent(bsJsSrc);
  const bsJsName = `bootstrap.${bsJsHash}.bundle.min.js`;
  fs.writeFileSync(path.join(outDir, bsJsName), bsJsSrc);
  manifest.bootstrapJs = `/static/vendor/${bsJsName}`;

  // FontAwesome CSS — rewrite ../webfonts/ -> ./webfonts/ so it resolves next to the hashed file
  const faCssSrcRaw = fs.readFileSync('node_modules/@fortawesome/fontawesome-free/css/all.min.css', 'utf8');
  const faCssSrc = faCssSrcRaw.replace(/\.\.\/webfonts\//g, './webfonts/');
  const faCssHash = hashContent(Buffer.from(faCssSrc, 'utf8'));
  const faCssName = `fontawesome.${faCssHash}.min.css`;
  fs.writeFileSync(path.join(outDir, faCssName), faCssSrc);
  manifest.fontawesomeCss = `/static/vendor/${faCssName}`;

  // FontAwesome webfonts — copied verbatim (their contents are stable per release)
  const webfontsDir = 'node_modules/@fortawesome/fontawesome-free/webfonts';
  const webfontFiles = fs.readdirSync(webfontsDir);
  for (const f of webfontFiles) {
    fs.copyFileSync(path.join(webfontsDir, f), path.join(outDir, 'webfonts', f));
  }
  manifest.fontawesomeWebfonts = webfontFiles.map((f) => `/static/vendor/webfonts/${f}`);

  // Admin nav CSS — content-hashed for cache-busting
  const navCssPath = 'static/css/admin-nav.css';
  try {
    const navCssSrc = fs.readFileSync(navCssPath);
    const navCssHash = hashContent(navCssSrc);
    const navCssName = `admin-nav.${navCssHash}.css`;
    fs.writeFileSync(path.join(outDir, navCssName), navCssSrc);
    manifest.adminNavCss = `/static/vendor/${navCssName}`;
  } catch {
    console.warn('Warning: static/css/admin-nav.css not found, skipping');
  }

  fs.writeFileSync(path.join(outDir, 'manifest.json'), JSON.stringify(manifest, null, 2));
  console.log('Vendor manifest:', manifest);

  // --- Generate service worker from template ---
  const templatePath = 'static/sw.template.js';
  const swOutPath = 'static/sw.js';
  try {
    let swTemplate = fs.readFileSync(templatePath, 'utf8');
    // Precaches: the same set the old hardcoded sw.js used (core vendor assets)
    const precacheUrls = [
      manifest.bootstrapCss,
      manifest.bootstrapJs,
      manifest.fontawesomeCss,
      manifest.adminNavCss,
    ].filter(Boolean);
    swTemplate = swTemplate.replace('__PRECACHE_URLS__', JSON.stringify(precacheUrls, null, 2));
    // Content-derived cache version: changes whenever any page bundle or vendor asset
    // changes, so sw.js bytes change on every frontend update. Browsers then install
    // the new service worker, which purges the old cache (see activate handler) —
    // stale bundles can no longer be served cache-first forever.
    const bundleHashes = fs.readdirSync('static/js')
      .filter(f => f.endsWith('.js') && !f.endsWith('.map'))
      .sort()
      .map(f => crypto.createHash('sha256').update(fs.readFileSync(path.join('static/js', f))).digest('hex').slice(0, 8))
      .join('');
    const cacheVersion = 'heat-cache-' + crypto.createHash('sha256')
      .update(bundleHashes + JSON.stringify(precacheUrls))
      .digest('hex').slice(0, 12);
    swTemplate = swTemplate.replaceAll('__CACHE_VERSION__', cacheVersion);
    fs.writeFileSync(swOutPath, swTemplate);
    console.log(`Generated ${swOutPath} (cache ${cacheVersion}) with ${precacheUrls.length} precache URLs`);
  } catch (err) {
    console.error('Error generating sw.js from template:', err.message);
  }
}

// --- end vendor bundling ---

if (isDev) {
  const ctx = await esbuild.context(config);
  await ctx.watch();
  console.log('Watching for changes...');
} else {
  await esbuild.build(config);
  console.log('Build complete');
}

// Always (re)build vendor bundle so manifest stays in sync with installed deps
await buildVendor();
