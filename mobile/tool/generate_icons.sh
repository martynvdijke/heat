#!/usr/bin/env bash
# Regenerate Android launcher PNGs from the web favicon so the app icon stays
# in sync with the Heat server branding. Requires rsvg-convert (librsvg).
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source_svg="$repo_root/static/favicon.svg"
res_dir="$repo_root/mobile/android/app/src/main/res"

for entry in mdpi:48 hdpi:72 xhdpi:96 xxhdpi:144 xxxhdpi:192; do
  density="${entry%%:*}"
  size="${entry##*:}"
  rsvg-convert -w "$size" -h "$size" "$source_svg" -o "$res_dir/mipmap-$density/ic_launcher.png"
done

echo "Regenerated launcher icons from $source_svg"
