#!/usr/bin/env bash
set -euo pipefail

version="${1:-}"
if [[ -z "$version" ]]; then
  echo "usage: scripts/verify-release.sh X.Y.Z" >&2
  exit 2
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

repo="bilancetech/gogcli"

changelog="CHANGELOG.md"
if ! rg -q "^## ${version} - " "$changelog"; then
  echo "missing changelog section for $version" >&2
  exit 2
fi
if rg -q "^## ${version} - Unreleased" "$changelog"; then
  echo "changelog section still Unreleased for $version" >&2
  exit 2
fi

notes_file="$(mktemp -t gogcli-release-notes)"
awk -v ver="$version" '
  $0 ~ "^## "ver" " {print "## "ver; in_section=1; next}
  in_section && /^## / {exit}
  in_section {print}
' "$changelog" | sed '/^$/d' > "$notes_file"

if [[ ! -s "$notes_file" ]]; then
  echo "release notes empty for $version" >&2
  exit 2
fi

release_body="$(gh release view "v$version" -R "$repo" --json body -q .body)"
if [[ -z "$release_body" ]]; then
  echo "GitHub release notes empty for v$version" >&2
  exit 2
fi

assets_count="$(gh release view "v$version" -R "$repo" --json assets -q '.assets | length')"
if [[ "$assets_count" -eq 0 ]]; then
  echo "no GitHub release assets for v$version" >&2
  exit 2
fi

ci_ok="$(gh run list -R "$repo" -L 1 --workflow ci --branch main --json conclusion -q '.[0].conclusion')"
if [[ "$ci_ok" != "success" ]]; then
  echo "CI not green for main" >&2
  exit 2
fi

make ci

tmp_assets_dir="$(mktemp -d -t gogcli-release-assets)"
gh release download "v$version" -R "$repo" -p checksums.txt -D "$tmp_assets_dir" >/dev/null
checksums_file="$tmp_assets_dir/checksums.txt"

sha_for_asset() {
  local name="$1"
  awk -v n="$name" '$2==n {print $1}' "$checksums_file"
}

darwin_amd64_expected="$(sha_for_asset "gogcli_${version}_darwin_amd64.tar.gz")"
darwin_arm64_expected="$(sha_for_asset "gogcli_${version}_darwin_arm64.tar.gz")"
linux_amd64_expected="$(sha_for_asset "gogcli_${version}_linux_amd64.tar.gz")"
linux_arm64_expected="$(sha_for_asset "gogcli_${version}_linux_arm64.tar.gz")"
windows_amd64_expected="$(sha_for_asset "gogcli_${version}_windows_amd64.zip")"
windows_arm64_expected="$(sha_for_asset "gogcli_${version}_windows_arm64.zip")"

if [[ -z "$darwin_amd64_expected" || -z "$darwin_arm64_expected" || -z "$linux_amd64_expected" || -z "$linux_arm64_expected" || -z "$windows_amd64_expected" || -z "$windows_arm64_expected" ]]; then
  echo "missing expected assets in checksums.txt for v$version" >&2
  exit 2
fi

rm -rf "$tmp_assets_dir"
rm -f "$notes_file"

echo "Release v$version verified (CI, GitHub release notes/assets, checksums)."
