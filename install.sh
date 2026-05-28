#!/usr/bin/env bash
set -euo pipefail

DEFAULT_VERSION="0.5.1"
REPO="gedaliahs/oops"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors
R='\033[0;31m'
G='\033[0;32m'
B='\033[1m'
D='\033[0;90m'
N='\033[0m'

info()  { echo -e "  ${D}>${N} $1"; }
ok()    { echo -e "  ${G}✓${N} $1"; }
warn()  { echo -e "  ${D}!${N} $1"; }
err()   { echo -e "  ${R}✗${N} $1"; exit 1; }
section() {
  echo ""
  echo -e "  ${B}$1${N}"
  echo ""
}

download_stdout() {
  local url="$1"
  if command -v curl &>/dev/null; then
    curl -fsSL --max-time 8 "$url"
  elif command -v wget &>/dev/null; then
    wget -qO- --timeout=8 "$url"
  else
    return 127
  fi
}

download_to() {
  local url="$1"
  local dest="$2"
  if ! try_download_to "$url" "$dest"; then
    if ! command -v curl &>/dev/null && ! command -v wget &>/dev/null; then
      err "curl or wget is required to download oops"
    fi
    err "download failed: $url"
  fi
}

try_download_to() {
  local url="$1"
  local dest="$2"
  if command -v curl &>/dev/null; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget &>/dev/null; then
    wget -qO "$dest" "$url"
  else
    return 127
  fi
}

discover_latest_version() {
  download_stdout "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\([^"]*\)".*/\1/p' \
    | head -n 1
}

if [ -n "${OOPS_VERSION:-}" ]; then
  VERSION="${OOPS_VERSION#v}"
elif [ -n "${OOPS_BASE_URL:-}" ]; then
  VERSION="$DEFAULT_VERSION"
else
  VERSION="$(discover_latest_version || true)"
  VERSION="${VERSION:-$DEFAULT_VERSION}"
fi

if [ -n "${OOPS_BASE_URL:-}" ]; then
  BASE_URL="${OOPS_BASE_URL%/}"
else
  BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"
fi

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
  echo -e "  Backs up risky terminal changes before they run."
  echo -e "  Type ${R}oops${N} to restore."
fi
echo ""

# ── Detect platform ──────────────────────────────────

section "1. System check"

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

info "Detected ${B}${OS}/${ARCH}${N}"

# ── Download and install binary ──────────────────────

section "2. Download"

ARCHIVE="oops_${OS}_${ARCH}.tar.gz"
URL="${BASE_URL}/${ARCHIVE}"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

info "Downloading..."
download_to "$URL" "$TMP/$ARCHIVE"

verify_checksum() {
  if [ "${OOPS_SKIP_CHECKSUM:-}" = "1" ]; then
    info "Skipping checksum verification"
    return
  fi

  info "Verifying checksum..."
  download_to "${BASE_URL}/SHA256SUMS" "$TMP/SHA256SUMS"

  grep "  ${ARCHIVE}$" "$TMP/SHA256SUMS" > "$TMP/SHA256SUM"
  if command -v shasum &>/dev/null; then
    (cd "$TMP" && shasum -a 256 -c SHA256SUM >/dev/null)
  elif command -v sha256sum &>/dev/null; then
    (cd "$TMP" && sha256sum -c SHA256SUM >/dev/null)
  else
    err "shasum or sha256sum required for verification"
  fi
  ok "Checksum verified"
}

verify_sigstore() {
  if [ "${OOPS_SKIP_SIGSTORE:-}" = "1" ]; then
    info "Skipping Sigstore verification"
    return
  fi

  local bundle="$TMP/${ARCHIVE}.sigstore"
  if ! try_download_to "${BASE_URL}/${ARCHIVE}.sigstore" "$bundle" 2>/dev/null; then
    if [ "${OOPS_REQUIRE_SIGSTORE:-}" = "1" ]; then
      err "Sigstore bundle missing"
    fi
    warn "Sigstore bundle not available; checksum verified"
    return
  fi

  if ! command -v cosign &>/dev/null; then
    if [ "${OOPS_REQUIRE_SIGSTORE:-}" = "1" ]; then
      err "cosign required for Sigstore verification"
    fi
    warn "cosign not found; checksum verified"
    return
  fi

  info "Verifying Sigstore signature..."
  local identity="https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/v${VERSION}"
  if ! cosign verify-blob "$TMP/$ARCHIVE" \
    --bundle "$bundle" \
    --certificate-identity "$identity" \
    --certificate-oidc-issuer "https://token.actions.githubusercontent.com" >/dev/null 2>&1; then
    err "Sigstore verification failed"
  fi
  ok "Sigstore signature verified"
}

verify_checksum
verify_sigstore
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"

section "3. Install"

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

run_self_test() {
  if [ "${OOPS_SKIP_SELF_TEST:-}" = "1" ]; then
    return
  fi

  info "Running self-test..."
  local test_home="$TMP/self-test-home"
  local test_work="$TMP/self-test-work"
  local test_file="$test_work/victim.txt"
  mkdir -p "$test_home" "$test_work"
  printf "oops self-test\n" > "$test_file"
  HOME="$test_home" "$INSTALL_DIR/oops" protect -- rm "$test_file" >/dev/null 2>&1
  rm "$test_file"
  HOME="$test_home" "$INSTALL_DIR/oops" >/dev/null 2>&1
  if [ "$(cat "$test_file" 2>/dev/null)" != "oops self-test" ]; then
    err "self-test failed"
  fi
  ok "Self-test passed"
}

run_self_test

# ── Add shell hook ───────────────────────────────────

section "4. Shell setup"

SHELL_NAME=$(basename "${SHELL:-zsh}")
HOOK_LINE=""
RC_FILE=""

case "$SHELL_NAME" in
  zsh)
    HOOK_LINE="eval \"\$(${INSTALL_DIR}/oops init zsh)\""
    # .zshenv loads for ALL zsh invocations (including AI agents, scripts, subshells)
    # .zshrc only loads for interactive shells
    RC_FILE="$HOME/.zshenv"
    ;;
  bash)
    HOOK_LINE="eval \"\$(${INSTALL_DIR}/oops init bash)\""
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

configure_profile() {
  local preset="${OOPS_INSTALL_PRESET:-}"
  if [ -z "$preset" ] && ! $UPGRADE && [ "${OOPS_SKIP_PROFILE_PROMPT:-}" != "1" ] && [ -r /dev/tty ] && [ -w /dev/tty ]; then
    echo ""
    echo -e "  ${B}Choose a protection profile${N}"
    echo -e "    ${G}1${N}) normal   ${D}default 2-hour retention, warnings on${N}"
    echo -e "    ${G}2${N}) agent    ${D}confirm every protected command${N}"
    echo -e "    ${G}3${N}) cautious ${D}24-hour retention, confirm high-risk commands${N}"
    echo -e "    ${G}4${N}) quiet    ${D}minimal prompts and warnings${N}"
    printf "  Select [1]: " > /dev/tty
    local reply
    read -r reply < /dev/tty || reply=""
    case "$reply" in
      2|agent) preset="agent" ;;
      3|cautious) preset="cautious" ;;
      4|quiet) preset="quiet" ;;
      *) preset="normal" ;;
    esac
  fi
  preset="${preset:-normal}"
  if [ "$preset" = "normal" ]; then
    ok "Profile: normal"
    return
  fi
  if HOME="$HOME" "$INSTALL_DIR/oops" config preset "$preset" >/dev/null; then
    ok "Profile: ${preset}"
  else
    warn "Could not apply profile: ${preset}"
  fi
}

maybe_install_cleanup_service() {
  local choice="${OOPS_INSTALL_CLEANUP_SERVICE:-}"
  if [ -z "$choice" ] && ! $UPGRADE && [ "${OOPS_SKIP_CLEANUP_PROMPT:-}" != "1" ] && [ -r /dev/tty ] && [ -w /dev/tty ]; then
    printf "  Install hourly background cleanup? [y/N] " > /dev/tty
    read -r choice < /dev/tty || choice=""
  fi
  case "$choice" in
    1|y|Y|yes|YES|true|TRUE)
      if "$INSTALL_DIR/oops" cleanup-service install >/dev/null; then
        ok "Background cleanup enabled"
      else
        warn "Could not enable background cleanup; run ${INSTALL_DIR}/oops cleanup-service install"
      fi
      ;;
    *)
      info "Optional: run ${INSTALL_DIR}/oops cleanup-service install for hourly cleanup"
      ;;
  esac
}

section "5. Preferences"
configure_profile
maybe_install_cleanup_service

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
