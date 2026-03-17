package cmd

import (
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

func doProtect(command string) error {
	protections := detect.Analyze(command)
	if len(protections) == 0 {
		return nil
	}

	cfg := config.Load()
	cwd, _ := os.Getwd()

	for _, p := range protections {
		// Confirmation mode: print a confirm prompt marker to stderr
		// The shell hook reads this and prompts the user
		shouldConfirm := false
		if cfg.ConfirmMode == "all" {
			shouldConfirm = true
		} else if cfg.ConfirmMode == "high" && p.Risk == detect.RiskHigh {
			shouldConfirm = true
		}

		if shouldConfirm {
			fmt.Fprintf(os.Stderr, "OOPS_CONFIRM:%s\n", p.Desc)
		} else if p.Risk == detect.RiskHigh && cfg.RiskWarning {
			fmt.Fprintln(os.Stderr, style.Warning(p.Desc))
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
		}

		// Handle git-specific actions
		if p.GitAction != "" {
			if err := protectGit(p, &entry); err != nil {
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
		stashArgs := []string{"stash", "push", "-m", "oops-backup: " + p.Desc}
		// Include untracked files for git clean
		if strings.Contains(p.Desc, "clean") {
			stashArgs = []string{"stash", "push", "-u", "-m", "oops-backup: " + p.Desc}
		}
		out, err := exec.Command("git", stashArgs...).CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "No local changes") {
				return nil
			}
			return fmt.Errorf("%s", string(out))
		}
		entry.GitAction = "stash"
		entry.GitStash = "stash@{0}"

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
