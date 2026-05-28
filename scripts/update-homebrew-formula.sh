#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION="${1:-}"
SUMS="${2:-$ROOT/dist/SHA256SUMS}"
FORMULA="$ROOT/packaging/homebrew/oops.rb"

if [ -z "$VERSION" ]; then
  echo "usage: scripts/update-homebrew-formula.sh <version> [SHA256SUMS]" >&2
  exit 2
fi

VERSION="${VERSION#v}"
if [ ! -f "$SUMS" ]; then
  echo "missing checksum file: $SUMS" >&2
  exit 1
fi

checksum_for() {
  awk -v name="$1" '$2 == name { print $1 }' "$SUMS"
}

darwin_amd64=$(checksum_for oops_darwin_amd64.tar.gz)
darwin_arm64=$(checksum_for oops_darwin_arm64.tar.gz)
linux_amd64=$(checksum_for oops_linux_amd64.tar.gz)
linux_arm64=$(checksum_for oops_linux_arm64.tar.gz)

for value in "$darwin_amd64" "$darwin_arm64" "$linux_amd64" "$linux_arm64"; do
  if [ -z "$value" ]; then
    echo "checksum file is missing one or more release archives" >&2
    exit 1
  fi
done

mkdir -p "$(dirname "$FORMULA")"
cat > "$FORMULA" <<EOF
class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "$VERSION"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v$VERSION/oops_darwin_arm64.tar.gz"
      sha256 "$darwin_arm64"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v$VERSION/oops_darwin_amd64.tar.gz"
      sha256 "$darwin_amd64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v$VERSION/oops_linux_arm64.tar.gz"
      sha256 "$linux_arm64"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v$VERSION/oops_linux_amd64.tar.gz"
      sha256 "$linux_amd64"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
EOF

echo "Updated $FORMULA"
