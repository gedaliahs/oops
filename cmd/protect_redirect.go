package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gedaliah/oops/internal/detect"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/gedaliah/oops/internal/trash"
	"github.com/spf13/cobra"
)

var protectRedirectCmd = &cobra.Command{
	Use:    "protect-redirect -- <command>",
	Short:  "Back up files before a redirect overwrites them (called by shell hook)",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	RunE:   runProtectRedirect,
}

func init() {
	rootCmd.AddCommand(protectRedirectCmd)
}

func runProtectRedirect(cmd *cobra.Command, args []string) error {
	command := strings.Join(args, " ")

	tokens := detect.Tokenize(command)

	for i, t := range tokens {
		if t == ">" && i+1 < len(tokens) {
			target := tokens[i+1]
			if target == "/dev/null" || strings.HasPrefix(target, "&") {
				continue
			}

			resolved := resolveFilePath(target)
			if _, err := os.Lstat(resolved); err != nil {
				continue // file doesn't exist, no need to back up
			}

			id := journal.GenerateID()

			trashDir, backed, err := trash.Backup(id, []string{resolved})
			if err != nil {
				fmt.Fprintln(os.Stderr, style.Error("oops: backup failed: "+err.Error()))
				continue
			}

			cwd, _ := os.Getwd()
			entry := journal.Entry{
				ID:        id,
				Timestamp: time.Now().Format(time.RFC3339),
				Command:   command,
				Action:    string(detect.ActionRedirect),
				Risk:      detect.RiskMedium.String(),
				TrashDir:  trashDir,
				Desc:      "> " + target,
				CWD:       cwd,
			}
			entry.Files = make([]string, len(backed))
			for j, b := range backed {
				entry.Files[j] = b.Original
			}

			if err := journal.Append(entry); err != nil {
				fmt.Fprintln(os.Stderr, style.Error("oops: journal write failed: "+err.Error()))
			}
		}
	}

	return nil
}

func resolveFilePath(p string) string {
	if len(p) == 0 {
		return p
	}
	if p[0] == '/' {
		return p
	}
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	return cwd + "/" + p
}
