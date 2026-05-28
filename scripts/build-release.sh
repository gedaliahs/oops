#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION="${1:-}"
DIST="${ROOT}/dist"

if [ -z "$VERSION" ]; then
  echo "usage: scripts/build-release.sh <version>" >&2
  exit 2
fi

VERSION="${VERSION#v}"
rm -rf "$DIST"
mkdir -p "$DIST"

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
  os=${target%/*}
  arch=${target#*/}
  archive="oops_${os}_${arch}.tar.gz"

  echo "Building ${os}/${arch}..."
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags="-s -w -X github.com/gedaliah/oops/cmd.Version=${VERSION}" -o "$DIST/oops" "$ROOT"
  touch -t 200001010000 "$DIST/oops"
  COPYFILE_DISABLE=1 tar --format ustar --uid 0 --gid 0 --uname root --gname root -C "$DIST" -cf "$DIST/oops.tar" oops
  gzip -n -c "$DIST/oops.tar" > "$DIST/$archive"
  rm -f "$DIST/oops.tar"
  rm -f "$DIST/oops"
done

(
  cd "$DIST"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 *.tar.gz > SHA256SUMS
  else
    sha256sum *.tar.gz > SHA256SUMS
  fi
)

echo "Wrote release assets to $DIST"
