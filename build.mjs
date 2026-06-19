import * as esbuild from 'esbuild';
import fs from 'fs';

const isDev = process.argv.includes('--dev');

// Shared modules that are imported by page entry points (not standalone entry points)
const sharedModules = new Set(['toast.ts', 'i18n.ts', 'theme.ts']);

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

if (isDev) {
  const ctx = await esbuild.context(config);
  await ctx.watch();
  console.log('Watching for changes...');
} else {
  await esbuild.build(config);
  console.log('Build complete');
}
