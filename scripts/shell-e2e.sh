#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

BIN="$TMP/oops"
CGO_ENABLED=0 go build -o "$BIN" "$ROOT"

run_bash() {
  command -v bash >/dev/null 2>&1 || return 0
  echo "shell-e2e: bash"
  local home="$TMP/bash-home"
  local work="$TMP/bash-work"
  mkdir -p "$home" "$work"
  printf "bash\n" > "$work/victim.txt"
  HOME="$home" bash --noprofile --norc -c '
    set -euo pipefail
    cd "$1"
    eval "$("$2" init bash)"
    rm victim.txt
    test ! -e victim.txt
    "$2" --plan >/tmp/oops-bash-plan.txt
    grep -q "victim.txt" /tmp/oops-bash-plan.txt
    "$2" >/dev/null
    test "$(cat victim.txt)" = "bash"
  ' _ "$work" "$BIN"
}

run_zsh() {
  command -v zsh >/dev/null 2>&1 || return 0
  echo "shell-e2e: zsh"
  local home="$TMP/zsh-home"
  local work="$TMP/zsh-work"
  mkdir -p "$home" "$work"
  printf "zsh\n" > "$work/victim.txt"
  HOME="$home" zsh -f -c '
    set -e
    cd "$1"
    eval "$("$2" init zsh)"
    _oops_preexec "rm victim.txt"
    rm victim.txt
    test ! -e victim.txt
    "$2" undo --dry-run >/tmp/oops-zsh-plan.txt
    grep -q "victim.txt" /tmp/oops-zsh-plan.txt
    "$2" undo >/dev/null
    test "$(cat victim.txt)" = "zsh"
  ' _ "$work" "$BIN"
}

run_fish() {
  command -v fish >/dev/null 2>&1 || return 0
  echo "shell-e2e: fish"
  local home="$TMP/fish-home"
  local work="$TMP/fish-work"
  mkdir -p "$home/.config/fish" "$work"
  printf "fish\n" > "$work/victim.txt"
  HOME="$home" fish -c '
    cd $argv[1]
    $argv[2] init fish | source
    emit fish_preexec "rm victim.txt"
    rm victim.txt
    test ! -e victim.txt
    $argv[2] restore --plan >/tmp/oops-fish-plan.txt
    grep -q "victim.txt" /tmp/oops-fish-plan.txt
    $argv[2] restore >/dev/null
    test (cat victim.txt) = fish
  ' "$work" "$BIN"
}

run_bash
run_zsh
run_fish

echo "shell-e2e: ok"
