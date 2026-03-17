#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${OOPS_BASE_URL:-https://oops-cli.com/releases}"
VERSION="0.4.2"
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
  # Check if hook needs migrating even if version is current
  NEEDS_MIGRATE=false
  SHELL_NAME=$(basename "${SHELL:-zsh}")
  if [ "$SHELL_NAME" = "zsh" ] && [ -f "$HOME/.zshrc" ] && grep -q "oops init" "$HOME/.zshrc" 2>/dev/null && ! grep -q "oops init" "$HOME/.zshenv" 2>/dev/null; then
    NEEDS_MIGRATE=true
  fi

  if [ "$CURRENT" = "$VERSION" ] && ! $NEEDS_MIGRATE; then
    echo -e "${R}  oops${N} ${D}v${VERSION}${N}"
    echo ""
    echo -e "  ${G}Already on the latest version.${N}"
    echo ""
    exit 0
  fi

  if [ "$CURRENT" = "$VERSION" ] && $NEEDS_MIGRATE; then
    echo -e "${R}  oops${N} ${D}v${VERSION}${N} — migrating shell hook"
  else
    echo -e "${R}  oops${N} upgrading ${D}v${CURRENT}${N} → ${D}v${VERSION}${N}"
  fi
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
  # Migrate: remove hook from old locations if it exists elsewhere
  OLD_FILES=""
  case "$SHELL_NAME" in
    zsh)  OLD_FILES="$HOME/.zshrc" ;;
    bash) OLD_FILES="$HOME/.bash_profile" ;;
  esac

  if [ -n "$OLD_FILES" ] && [ "$OLD_FILES" != "$RC_FILE" ]; then
    for old in $OLD_FILES; do
      if [ -f "$old" ] && grep -q "oops init" "$old" 2>/dev/null; then
        # Remove from old file
        grep -v "oops init" "$old" > "$old.tmp" && mv "$old.tmp" "$old"
        info "Migrated hook from ${old}"
      fi
    done
  fi

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
echo -e "  ${G}All set.${N} Open a new terminal tab to activate."
echo ""
if ! $UPGRADE; then
  echo -e "  ${B}Try it:${N}"
  echo -e "    ${D}\$${N} rm something.txt && ${R}oops${N}"
  echo -e "    ${D}\$${N} oops tutorial"
  echo ""
  echo -e "  ${D}Using AI agents? Run${N} ${R}oops agent-mode${N} ${D}to protect against${N}"
  echo -e "  ${D}Claude Code, Cursor, Aider, etc.${N}"
  echo ""
fi
