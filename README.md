# oops

Undo for your terminal. A shell hook that backs up files before destructive commands run and lets you restore them with one command.

## Install

```
curl -fsSL oops-cli.com/install.sh | bash
```

The installer handles everything — discovers the latest GitHub Release, verifies checksums, verifies Sigstore signatures when `cosign` is available, adds the shell hook, offers an arrow-key protection profile picker, repairs local backup directory ownership when needed, and runs a quick restore self-test.

Homebrew is available from the public tap:

```
brew install gedaliahs/tap/oops
```

## Usage

```
$ rm -rf src/
▲ rm -r ~/project/src

$ oops
✓ Undid: rm -r ~/project/src
↩ restored ~/project/src
```

`oops 2` undoes the second-to-last action. `oops log` shows history, `oops status` summarizes current protection, and `oops diff` compares the backup to the current file.

## Supported commands

| Command | What oops does | Undo |
|---|---|---|
| `rm` / `rm -rf` | Copies files to trash | restore |
| `mv a b` | Backs up overwrite target | restore b |
| `cp a b` | Backs up overwrite target | restore b |
| `> file.txt` | Snapshots before redirect | restore |
| `sed -i` | Copies before in-place edit | restore |
| `perl -pi` | Copies before in-place edit | restore |
| `chmod` / `chown` | Records permissions | restore |
| `git reset --hard` | Creates stash | stash apply |
| `git checkout .` | Creates stash | stash apply |
| `git restore .` / `git switch -f` | Creates stash | stash apply |
| `git branch -D` | Logs SHA | recreate |
| `git clean -fd` | Stashes untracked files | stash apply |
| `find ... -delete` | Backs up search roots | restore |
| `xargs rm` / `fd -x rm` / `parallel rm` | Backs up the current tree | restore |
| `rsync --delete` | Backs up destination | restore |
| `dd of=...` | Backs up output file | restore |
| `git worktree remove` | Backs up the worktree path | restore |
| `make clean` / `npm run clean` / `yarn clean` / `pnpm clean` | Backs up the current tree | restore |

## Commands

| Command | Description |
|---|---|
| `oops` | Undo last action (pass a number to go further back) |
| `oops undo` / `oops restore` | Explicit undo command and alias |
| `oops undo --dry-run` | Show what would happen without changing files |
| `oops restore --plan` | Show conflicts, backups, overwrite behavior, and git actions |
| `oops --overwrite` | Restore over an existing target |
| `oops --backup-current` | Move an existing target aside before restore |
| `oops --to DIR` | Restore into a separate directory |
| `oops status` | Show health, hook, trash, and policy state |
| `oops diff` | Show changes between a backup and current files (colorized; `--full` for large files) |
| `oops show` | Preview what would be restored |
| `oops log` | Show undo history (filter with `--risk`, `--path`, `--here`; `--absolute`, `--flat`) |
| `oops keep` | Keep a backup from automatic cleanup |
| `oops unkeep` | Allow a kept backup to be cleaned up |
| `oops size` | Show backup disk usage |
| `oops clean` | Remove old backups (`--all` for everything) |
| `oops cleanup-service` | Install, remove, or inspect hourly background cleanup |
| `oops config` | View or change settings (`onboarding_hints` toggles the new-user undo hint) |
| `oops config preset agent` | Apply a risk policy preset (`normal`, `agent`, `cautious`, `quiet`) |
| `oops protect-path` | Add high-safety rules for important paths |
| `oops doctor` | Check installation health |
| `oops doctor --fix` | Repair common local permission problems |
| `oops tutorial` | Interactive walkthrough |
| `oops uninstall` | Remove oops from your system |
| `oops --version` | Print version |
| `oops --upgrade` | Upgrade to the latest version |

## Works with AI coding agents

Any tool that runs shell commands in your terminal goes through the same hook — Claude Code, Cursor, Aider, Codex, etc. If an AI agent accidentally runs `rm -rf` or `git reset --hard`, oops catches it. Type `oops` to undo what the agent did.

## How it works

A `preexec` shell hook pattern-matches each command. Non-destructive commands pass through with zero overhead (no subprocess). Destructive commands trigger `oops protect`, which backs up affected files to `~/.oops/trash/` then lets the original command run.

Backups are copied into `~/.oops/trash/` with a manifest in the journal. Copying costs more disk than hard links, but it keeps backups correct for overwrites, redirects, and in-place edits where shared inodes would be unsafe. Restore builds a plan first, detects conflicts before mutating files, stages backup content, then commits the restore. Auto-cleanup removes old entries after 2 hours by default, and `oops keep` or protected-path rules can retain important backups longer.

## Uninstall

```
oops uninstall
```

Removes the shell hook and backup directory. Then run `sudo rm /usr/local/bin/oops` to remove the binary.

## License

MIT
