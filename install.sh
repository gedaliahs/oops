#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${OOPS_BASE_URL:-https://oops-cli.com/releases}"
VERSION="0.3.3"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors
R='\033[0;31m'
G='\033[0;32m'
B='\033[1m'
D='\033[0;90m'
N='\033[0m'

info()  { echo -e "  ${D}>${N} $1"; }
ok()    { echo -e "  ${G}✓${N} $1"; }
err()   { echo -e "  ${R}✗${N} $1"; exit 1; }

UPGRADE=false
if command -v oops &>/dev/null; then
  CURRENT=$(oops --version 2>/dev/null | sed 's/oops v//')
  UPGRADE=true
fi

echo ""
if $UPGRADE; then
  if [ "$CURRENT" = "$VERSION" ]; then
    echo -e "${R}  oops${N} ${D}v${VERSION}${N}"
    echo ""
    echo -e "  ${G}Already on the latest version.${N}"
    echo ""
    exit 0
  fi
  echo -e "${R}  oops${N} upgrading ${D}v${CURRENT}${N} → ${D}v${VERSION}${N}"
else
  echo -e "${R}  oops${N} installer ${D}v${VERSION}${N}"
  echo -e "${D}  undo for your terminal${N}"
  echo ""
  echo -e "  A shell hook that backs up files before destructive"
  echo -e "  commands run. Type ${R}oops${N} to restore them."
  echo -e "  ${D}Works with rm, mv, sed -i, git reset, chmod, and more.${N}"
fi
echo ""

# ── Detect platform ──────────────────────────────────

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       err "Unsupported architecture: $ARCH" ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) err "Unsupported OS: $OS" ;;
esac

echo -e "  ${B}Installing${N}"
echo ""
info "Detected ${B}${OS}/${ARCH}${N}"

# ── Download and install binary ──────────────────────

URL="${BASE_URL}/oops_${OS}_${ARCH}.tar.gz"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

info "Downloading..."
if command -v curl &>/dev/null; then
  curl -fsSL "$URL" -o "$TMP/oops.tar.gz"
elif command -v wget &>/dev/null; then
  wget -qO "$TMP/oops.tar.gz" "$URL"
else
  err "curl or wget required"
fi

tar -xzf "$TMP/oops.tar.gz" -C "$TMP"

mkdir -p "$INSTALL_DIR" 2>/dev/null || true
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/oops" "$INSTALL_DIR/oops"
else
  info "Requires sudo for ${INSTALL_DIR}"
  sudo mkdir -p "$INSTALL_DIR"
  sudo mv "$TMP/oops" "$INSTALL_DIR/oops"
fi
chmod +x "$INSTALL_DIR/oops"
ok "Installed binary to ${INSTALL_DIR}/oops"

# ── Add shell hook ───────────────────────────────────

SHELL_NAME=$(basename "${SHELL:-zsh}")
HOOK_LINE=""
RC_FILE=""

case "$SHELL_NAME" in
  zsh)
    HOOK_LINE='eval "$(oops init zsh)"'
    # .zshenv loads for ALL zsh invocations (including AI agents, scripts, subshells)
    # .zshrc only loads for interactive shells
    RC_FILE="$HOME/.zshenv"
    ;;
  bash)
    HOOK_LINE='eval "$(oops init bash)"'
    if [ -f "$HOME/.bashrc" ]; then
      RC_FILE="$HOME/.bashrc"
    elif [ -f "$HOME/.bash_profile" ]; then
      RC_FILE="$HOME/.bash_profile"
    else
      RC_FILE="$HOME/.bashrc"
    fi
    ;;
  fish)
    HOOK_LINE='oops init fish | source'
    RC_FILE="$HOME/.config/fish/config.fish"
    ;;
esac

if [ -n "$RC_FILE" ]; then
  if [ -f "$RC_FILE" ] && grep -q "oops init" "$RC_FILE" 2>/dev/null; then
    ok "Shell hook already in ${RC_FILE}"
  else
    if [ "$SHELL_NAME" = "fish" ]; then
      mkdir -p "$(dirname "$RC_FILE")"
    fi
    echo "" >> "$RC_FILE"
    echo "$HOOK_LINE" >> "$RC_FILE"
    ok "Added shell hook to ${RC_FILE}"
  fi
fi

# ── Create oops directory ────────────────────────────

if [ ! -d "$HOME/.oops" ]; then
  mkdir -p "$HOME/.oops/trash"
  ok "Created ~/.oops backup directory"
else
  ok "~/.oops already exists"
fi

# ── Done ─────────────────────────────────────────────

echo ""
if $UPGRADE; then
  echo -e "  ${G}Upgraded to v${VERSION}.${N} Open a new terminal tab to activate."
else
  echo -e "  ${G}All set.${N} Open a new terminal tab to activate."
  echo ""
  echo -e "  Then try: ${D}\$${N} rm something.txt && ${R}oops${N}"
  echo -e "  Or run:   ${D}\$${N} oops tutorial"
fi
echo ""
