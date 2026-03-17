# oops

Undo for your terminal. A shell hook that backs up files before destructive commands run and lets you restore them with one command.

## Install

```
curl -fsSL oops-cli.com/install.sh | bash
```

The installer handles everything — downloads the binary, adds the shell hook to your shell config, and creates the backup directory.

## Usage

```
$ rm -rf src/
▲ rm -r ~/project/src

$ oops
✓ Undid: rm -r ~/project/src
↩ restored ~/project/src
```

`oops 2` undoes the second-to-last action. `oops log` shows history.

## Supported commands

| Command | What oops does | Undo |
|---|---|---|
| `rm` / `rm -rf` | Copies files to trash | restore |
| `mv a b` | Backs up overwrite target | restore b |
| `> file.txt` | Snapshots before redirect | restore |
| `sed -i` | Copies before in-place edit | restore |
| `chmod` / `chown` | Records permissions | restore |
| `git reset --hard` | Creates stash | stash apply |
| `git checkout .` | Creates stash | stash apply |
| `git branch -D` | Logs SHA | recreate |
| `git clean -fd` | Stashes untracked files | stash apply |

## Commands

| Command | Description |
|---|---|
| `oops` | Undo last action (pass a number to go further back) |
| `oops log` | Show undo history |
| `oops size` | Show backup disk usage |
| `oops clean` | Remove old backups (`--all` for everything) |
| `oops config` | View or change settings |
| `oops doctor` | Check installation health |
| `oops tutorial` | Interactive walkthrough |
| `oops uninstall` | Remove oops from your system |
| `oops --version` | Print version |
| `oops --upgrade` | Upgrade to the latest version |

## Works with AI coding agents

Any tool that runs shell commands in your terminal goes through the same hook — Claude Code, Cursor, Aider, Codex, etc. If an AI agent accidentally runs `rm -rf` or `git reset --hard`, oops catches it. Type `oops` to undo what the agent did.

## How it works

A `preexec` shell hook pattern-matches each command. Non-destructive commands pass through with zero overhead (no subprocess). Destructive commands trigger `oops protect`, which backs up affected files to `~/.oops/trash/` then lets the original command run.

Backups use **hard links** — a second reference to the same data on disk. This means backing up a 10GB directory is as fast as a 10KB file (instant, no extra disk space). The data only takes up space after the original is deleted. Falls back to copying on external drives.

## Uninstall

```
oops uninstall
```

Removes the shell hook and backup directory. Then run `sudo rm /usr/local/bin/oops` to remove the binary.

## License

MIT
