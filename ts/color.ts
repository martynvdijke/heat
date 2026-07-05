// Shared color normalization for car colors.
// Supports both named colors (from the admin seed) and arbitrary hex values.

const NAMED_COLORS: Record<string, string> = {
  'red': '#ff4444',
  'blue': '#4444ff',
  'green': '#44ff44',
  'yellow': '#ffff44',
  'grey': '#aaaaaa',
  'silver': '#aaaaaa',
  'black': '#333333',
  'purple': '#9b59b6',
  'orange': '#e67e22',
};

export function normalizeHex(color: string): string {
  if (!color) return '#cccccc';
  const trimmed = color.trim();
  // Named color lookup (case-insensitive, lowercase map)
  const named = NAMED_COLORS[trimmed.toLowerCase()];
  if (named) return named;
  // Ensure # prefix
  let hex = trimmed.startsWith('#') ? trimmed : '#' + trimmed;
  // Validate 6-digit hex
  if (/^#[0-9a-fA-F]{6}$/.test(hex)) return hex;
  return '#cccccc';
}
