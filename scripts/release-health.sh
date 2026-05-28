#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION="${1:-}"
SUMS="${2:-$ROOT/dist/SHA256SUMS}"
FORMULA="${3:-$ROOT/packaging/homebrew/oops.rb}"
SUMMARY="${GITHUB_STEP_SUMMARY:-}"

if [ -z "$VERSION" ]; then
  echo "usage: scripts/release-health.sh <version> [SHA256SUMS] [formula]" >&2
  exit 2
fi

VERSION="${VERSION#v}"
BASE_URL="https://github.com/gedaliahs/oops/releases/download/v${VERSION}"
SITE_INSTALLER_URL="${OOPS_SITE_INSTALLER_URL:-https://oops-cli.com/install.sh}"
TAP_FORMULA_URL="${OOPS_TAP_FORMULA_URL:-}"
STRICT_LIVE="${OOPS_RELEASE_HEALTH_STRICT_LIVE:-0}"
COMPARE_LOCAL="${OOPS_RELEASE_HEALTH_COMPARE_LOCAL:-${GITHUB_ACTIONS:-0}}"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

report=()
failures=0

check() {
  local name="$1"
  shift
  if "$@"; then
    report+=("- [ok] ${name}")
  else
    report+=("- [fail] ${name}")
    failures=$((failures + 1))
  fi
}

live_check() {
  local name="$1"
  shift
  if "$@"; then
    report+=("- [ok] ${name}")
  elif [ "$STRICT_LIVE" = "1" ]; then
    report+=("- [fail] ${name}")
    failures=$((failures + 1))
  else
    report+=("- [warn] ${name}")
  fi
}

fetch() {
  curl -fsSL "$1" -o "$2"
}

asset_exists() {
  curl -fsI "$1" >/dev/null
}

formula_matches_sums() {
  local formula="$1"
  local sums="$2"
  while read -r sum archive; do
    grep -q "$sum" "$formula" || return 1
    grep -q "$archive" "$formula" || return 1
  done < "$sums"
}

installer_smoke() {
  local install_dir home_dir
  install_dir=$(mktemp -d)
  home_dir=$(mktemp -d)
  HOME="$home_dir" INSTALL_DIR="$install_dir" OOPS_BASE_URL="$BASE_URL" OOPS_VERSION="$VERSION" OOPS_SKIP_PROFILE_PROMPT=1 OOPS_SKIP_CLEANUP_PROMPT=1 bash "$ROOT/install.sh" >/tmp/oops-release-health-install.log 2>&1
  "$install_dir/oops" --version | grep -q "oops v${VERSION}"
}

site_installer_current() {
  fetch "$SITE_INSTALLER_URL" "$tmp/site-install.sh"
  grep -q "DEFAULT_VERSION=\"${VERSION}\"" "$tmp/site-install.sh"
}

site_installer_smoke() {
  local install_dir home_dir
  install_dir=$(mktemp -d)
  home_dir=$(mktemp -d)
  [ -f "$tmp/site-install.sh" ] || fetch "$SITE_INSTALLER_URL" "$tmp/site-install.sh"
  HOME="$home_dir" INSTALL_DIR="$install_dir" OOPS_BASE_URL="$BASE_URL" OOPS_VERSION="$VERSION" OOPS_SKIP_PROFILE_PROMPT=1 OOPS_SKIP_CLEANUP_PROMPT=1 bash "$tmp/site-install.sh" >/tmp/oops-site-install-health.log 2>&1
  "$install_dir/oops" --version | grep -q "oops v${VERSION}"
}

tap_formula_matches() {
  if [ -n "$TAP_FORMULA_URL" ]; then
    fetch "$TAP_FORMULA_URL" "$tmp/tap-oops.rb"
  elif command -v gh >/dev/null 2>&1; then
    gh api repos/gedaliahs/homebrew-tap/contents/Formula/oops.rb --jq '.content' | base64 --decode > "$tmp/tap-oops.rb"
  else
    fetch "https://raw.githubusercontent.com/gedaliahs/homebrew-tap/main/Formula/oops.rb" "$tmp/tap-oops.rb"
  fi
  formula_matches_sums "$tmp/tap-oops.rb" "$tmp/SHA256SUMS"
}

check "release SHA256SUMS is downloadable" fetch "${BASE_URL}/SHA256SUMS" "$tmp/SHA256SUMS"
if [ -f "$SUMS" ] && [ -f "$tmp/SHA256SUMS" ] && { [ "$COMPARE_LOCAL" = "1" ] || [ "$COMPARE_LOCAL" = "true" ]; }; then
  check "local and release SHA256SUMS match" cmp -s "$SUMS" "$tmp/SHA256SUMS"
fi

for archive in oops_darwin_amd64.tar.gz oops_darwin_arm64.tar.gz oops_linux_amd64.tar.gz oops_linux_arm64.tar.gz; do
  check "${archive} exists" asset_exists "${BASE_URL}/${archive}"
  check "${archive}.sigstore exists" asset_exists "${BASE_URL}/${archive}.sigstore"
done
check "SHA256SUMS.sigstore exists" asset_exists "${BASE_URL}/SHA256SUMS.sigstore"

if [ -f "$FORMULA" ] && [ -f "$tmp/SHA256SUMS" ]; then
  check "Homebrew formula matches release checksums" formula_matches_sums "$FORMULA" "$tmp/SHA256SUMS"
fi

check "installer smoke test from release assets" installer_smoke
live_check "live site installer default version is v${VERSION}" site_installer_current
live_check "live site installer can install v${VERSION}" site_installer_smoke
live_check "Homebrew tap formula matches release checksums" tap_formula_matches

{
  echo "## oops v${VERSION} release health"
  echo
  printf '%s\n' "${report[@]}"
} | tee "$tmp/summary.md"

if [ -n "$SUMMARY" ]; then
  cat "$tmp/summary.md" >> "$SUMMARY"
fi

if [ "$failures" -ne 0 ]; then
  exit 1
fi
