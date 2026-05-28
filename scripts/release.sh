#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SITE_ROOT=$(cd "$ROOT/../oops-site" 2>/dev/null && pwd || true)
VERSION="${1:-}"
PUBLISH=false

if [ -z "$VERSION" ]; then
  echo "usage: scripts/release.sh <version> [--publish]" >&2
  exit 2
fi

if [ "${2:-}" = "--publish" ]; then
  PUBLISH=true
fi

VERSION="${VERSION#v}"
cd "$ROOT"

perl -0pi -e "s/var Version = \"[^\"]+\"/var Version = \"$VERSION\"/" cmd/root.go
perl -0pi -e "s/DEFAULT_VERSION=\"[^\"]+\"/DEFAULT_VERSION=\"$VERSION\"/" install.sh

gofmt -w cmd internal
go test -race ./...
go vet ./...

scripts/build-release.sh "$VERSION"
scripts/update-homebrew-formula.sh "$VERSION"

if [ -n "$SITE_ROOT" ] && [ -d "$SITE_ROOT/site" ]; then
  cp install.sh "$SITE_ROOT/site/install.sh"
  perl -0pi -e "s/\"softwareVersion\": \"[^\"]+\"/\"softwareVersion\": \"$VERSION\"/" "$SITE_ROOT/site/index.html"
  rm -rf "$SITE_ROOT/site/releases"
  echo "Synced installer to $SITE_ROOT and removed site-hosted release archives"
fi

echo
echo "Release files are prepared for v$VERSION."

if ! $PUBLISH; then
  echo "Review changes, then run: scripts/release.sh $VERSION --publish"
  exit 0
fi

git add cmd/root.go install.sh .github/workflows/ci.yml .github/workflows/release.yml scripts packaging README.md internal cmd go.mod go.sum
git commit -m "v$VERSION" || true
git tag "v$VERSION"
git push origin main
git push origin "v$VERSION"

if [ -n "$SITE_ROOT" ] && [ -d "$SITE_ROOT/.git" ]; then
  (
    cd "$SITE_ROOT"
    git add -u site
    git commit -m "Update site for v$VERSION" || true
    git push origin main
  )
fi
