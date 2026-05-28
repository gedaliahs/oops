package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/gedaliah/oops/internal/cleanup"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/gedaliah/oops/internal/trash"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff [N]",
	Short: "Show changes between a backup and current files",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	cleanup.RunIfNeeded()

	n := 1
	if len(args) == 1 {
		var err error
		n, err = strconv.Atoi(args[0])
		if err != nil || n < 1 {
			return fmt.Errorf("invalid argument: %s (expected a positive number)", args[0])
		}
	}

	entries, err := journal.Last(n)
	if err != nil {
		return fmt.Errorf("reading journal: %w", err)
	}
	if len(entries) == 0 {
		fmt.Println(style.Warning("Nothing to diff"))
		return nil
	}
	if n > len(entries) {
		return fmt.Errorf("only %d undoable actions in history", len(entries))
	}

	entry := entries[n-1]
	if entry.GitAction != "" {
		fmt.Println(style.Dim.Render("Diff is not available for git journal entries."))
		if entry.GitStash != "" {
			fmt.Println(style.Dim.Render("  Stash: ") + entry.GitStash)
		}
		if entry.GitSHA != "" {
			fmt.Println(style.Dim.Render("  SHA: ") + entry.GitSHA)
		}
		return nil
	}
	if entry.TrashDir == "" {
		return fmt.Errorf("no backup found for this action")
	}

	manifest, err := trash.ReadManifest(entry.TrashDir)
	if err != nil {
		return err
	}

	printed := false
	for _, bf := range manifest.Files {
		if _, err := os.Lstat(bf.Original); os.IsNotExist(err) {
			printMissingDiff(bf)
			printed = true
			continue
		} else if err != nil {
			return fmt.Errorf("checking %s: %w", bf.Original, err)
		}

		out, different, err := diffPaths(bf)
		if err != nil {
			return err
		}
		if different {
			fmt.Print(out)
			printed = true
		}
	}

	if !printed {
		fmt.Println(style.Dim.Render("No content changes."))
	}
	return nil
}

func diffPaths(bf trash.BackedUpFile) (string, bool, error) {
	args := []string{"-u"}
	if bf.IsDir {
		args = []string{"-ruN"}
	}
	args = append(args, bf.Backup, bf.Original)
	diff := exec.Command("diff", args...)
	out, err := diff.CombinedOutput()
	if err == nil {
		return "", false, nil
	}
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return string(out), true, nil
	}
	return "", false, fmt.Errorf("diff failed: %s", string(out))
}

func printMissingDiff(bf trash.BackedUpFile) {
	fmt.Printf("--- %s\n", style.ShortenPath(bf.Backup))
	fmt.Printf("+++ %s\n", style.ShortenPath(bf.Original))
	fmt.Println(style.Warning("current path is missing"))
}
