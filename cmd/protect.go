package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/detect"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/gedaliah/oops/internal/trash"
	"github.com/spf13/cobra"
)

var errNoGitBackup = errors.New("no git backup created")

var protectCmd = &cobra.Command{
	Use:    "protect -- <command>",
	Short:  "Back up files before a destructive command (called by shell hook)",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	RunE:   runProtect,
}

func init() {
	rootCmd.AddCommand(protectCmd)
}

func runProtect(cmd *cobra.Command, args []string) error {
	command := strings.Join(args, " ")
	return doProtect(command)
}

// catchLearnLimit is how many catches a new user sees the inline undo hint for
// before it fades to the quiet baseline.
const catchLearnLimit = 8

// emitCatchNotice prints the post-command notice for a backed-up destructive
// command. For brand-new users it teaches how to undo — a one-time two-line
// nudge, then a fading inline hint — so the tool stops being invisible. Once the
// user has run a successful undo (config.HasSeenUndo), it takes a read-only fast
// path and reverts to the original quiet baseline: warn only on high-risk
// commands when risk warnings are enabled.
func emitCatchNotice(cfg config.Config, p *detect.Protection) {
	desc := style.ShortenPath(p.Desc)
	highRisk := p.Risk == detect.RiskHigh

	if cfg.OnboardingHints && !config.HasSeenUndo() {
		switch n := config.IncrementCatchCount(); {
		case n <= 1:
			fmt.Fprintln(os.Stderr, style.FirstCatchNudge(desc, highRisk))
			return
		case n <= catchLearnLimit:
			fmt.Fprintln(os.Stderr, style.CatchHint(desc, highRisk))
			return
		}
		// Past the learning window: fall through to the quiet baseline.
	}

	if highRisk && cfg.RiskWarning {
		fmt.Fprintln(os.Stderr, style.Warning(desc))
	}
}

func doProtect(command string) error {
	protections := detect.Analyze(command)
	if len(protections) == 0 {
		return nil
	}

	cfg := config.Load()
	cwd, _ := os.Getwd()

	for _, p := range protections {
		rule, protected := cfg.MatchProtectedPath(p.Files, cwd)
		if protected {
			p.Risk = detect.RiskHigh
		}

		// Confirmation mode: print a confirm prompt marker to stderr
		// The shell hook reads this and prompts the user
		shouldConfirm := false
		if protected && rule.AlwaysConfirm {
			shouldConfirm = true
		} else if cfg.ConfirmMode == "all" {
			shouldConfirm = true
		} else if cfg.ConfirmMode == "high" && p.Risk == detect.RiskHigh {
			shouldConfirm = true
		}

		if shouldConfirm {
			fmt.Fprintf(os.Stderr, "OOPS_CONFIRM:%s\n", style.ShortenPath(p.Desc))
		} else {
			emitCatchNotice(cfg, p)
		}

		id := journal.GenerateID()

		entry := journal.Entry{
			ID:        id,
			Timestamp: time.Now().Format(time.RFC3339),
			Command:   command,
			Action:    string(p.Action),
			Risk:      p.Risk.String(),
			Desc:      p.Desc,
			CWD:       cwd,
			Files:     p.Files,
			Protected: protected,
		}
		if protected && rule.RetentionHours > 0 {
			entry.KeepUntil = time.Now().Add(time.Duration(rule.RetentionHours) * time.Hour).Format(time.RFC3339)
		}

		// Handle git-specific actions
		if p.GitAction != "" {
			if err := protectGit(p, &entry); err != nil {
				if errors.Is(err, errNoGitBackup) {
					continue
				}
				fmt.Fprintln(os.Stderr, style.Error("oops: git backup failed: "+err.Error()))
				continue
			}
			if err := journal.Append(entry); err != nil {
				fmt.Fprintln(os.Stderr, style.Error("oops: journal write failed: "+err.Error()))
			}
			continue
		}

		// Standard file backup
		if len(p.Files) == 0 {
			continue
		}

		trashDir, backed, err := trash.Backup(id, p.Files)
		if err != nil {
			fmt.Fprintln(os.Stderr, style.Error("oops: backup failed: "+err.Error()))
			continue
		}

		entry.TrashDir = trashDir
		entry.Files = make([]string, len(backed))
		for i, b := range backed {
			entry.Files[i] = b.Original
		}

		if err := journal.Append(entry); err != nil {
			fmt.Fprintln(os.Stderr, style.Error("oops: journal write failed: "+err.Error()))
		}
	}

	return nil
}

func protectGit(p *detect.Protection, entry *journal.Entry) error {
	switch p.GitAction {
	case "stash":
		before := currentStashRef()
		stashArgs := []string{"stash", "push", "-m", "oops-backup: " + p.Desc}
		// Include untracked files for git clean
		if strings.Contains(p.Desc, "clean") {
			stashArgs = []string{"stash", "push", "-u", "-m", "oops-backup: " + p.Desc}
		}
		out, err := exec.Command("git", stashArgs...).CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "No local changes") {
				return errNoGitBackup
			}
			return fmt.Errorf("%s", string(out))
		}
		if strings.Contains(string(out), "No local changes") {
			return errNoGitBackup
		}
		after := currentStashRef()
		entry.GitAction = "stash"
		if after != "" && after != before {
			entry.GitStash = after
		} else {
			entry.GitStash = "stash@{0}"
		}

	case "log-branch":
		out, err := exec.Command("git", "rev-parse", p.GitRef).CombinedOutput()
		if err != nil {
			return fmt.Errorf("could not resolve ref %s: %s", p.GitRef, string(out))
		}
		entry.GitAction = "log-branch"
		entry.GitRef = p.GitRef
		entry.GitSHA = strings.TrimSpace(string(out))
	}

	return nil
}

func currentStashRef() string {
	out, err := exec.Command("git", "rev-parse", "--verify", "refs/stash").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
