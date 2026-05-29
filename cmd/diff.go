package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gedaliah/oops/internal/cleanup"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/gedaliah/oops/internal/trash"
	"github.com/spf13/cobra"
)

// diffSizeThreshold is the per-file size above which `oops diff` shows a size
// summary instead of dumping the full diff, unless --full is passed.
const diffSizeThreshold = 2 << 20 // 2 MB

var diffFull bool

var diffCmd = &cobra.Command{
	Use:   "diff [N]",
	Short: "Show changes between a backup and current files",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().BoolVar(&diffFull, "full", false, "Show the full diff even for large files")
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

		if !bf.IsDir && !diffFull {
			if summary, skip := largeFileSummary(bf); skip {
				fmt.Println(summary)
				printed = true
				continue
			}
		}

		out, different, err := diffPaths(bf)
		if err != nil {
			return err
		}
		if different {
			fmt.Print(colorizeUnifiedDiff(out))
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
	fmt.Println(style.DiffHeader.Render("--- " + style.ShortenPath(bf.Backup)))
	fmt.Println(style.DiffHeader.Render("+++ " + style.ShortenPath(bf.Original)))
	fmt.Println(style.Warning("current path is missing"))
}

// colorizeUnifiedDiff styles a `diff -u` blob: green additions, red deletions,
// cyan hunk headers, bold/dim file headers. Header (---/+++) and hunk (@@) lines
// are matched BEFORE the generic +/- rules because they also start with +/-.
// Lipgloss strips color automatically when stdout is not a TTY, so piping stays
// plain text.
func colorizeUnifiedDiff(out string) string {
	lines := strings.Split(out, "\n")
	var b strings.Builder
	for i, ln := range lines {
		// Skip the empty trailing element produced by a final newline.
		if i == len(lines)-1 && ln == "" {
			continue
		}
		switch {
		case strings.HasPrefix(ln, "Binary files ") && strings.HasSuffix(ln, " differ"):
			b.WriteString(renderBinaryDiffLine(ln))
		case strings.HasPrefix(ln, "+++"), strings.HasPrefix(ln, "---"):
			b.WriteString(style.DiffHeader.Render(ln))
		case strings.HasPrefix(ln, "@@"):
			b.WriteString(style.DiffHunk.Render(ln))
		case strings.HasPrefix(ln, "+"):
			b.WriteString(style.DiffAdd.Render(ln))
		case strings.HasPrefix(ln, "-"):
			b.WriteString(style.DiffDel.Render(ln))
		default:
			b.WriteString(ln)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderBinaryDiffLine turns diff's opaque "Binary files A and B differ" into a
// clean one-liner with a backup→current size delta. Handles both single-file
// diffs and per-file lines inside a directory diff. Falls back to dimming the
// original line if it cannot be parsed.
func renderBinaryDiffLine(ln string) string {
	inner := strings.TrimSuffix(strings.TrimPrefix(ln, "Binary files "), " differ")
	parts := strings.SplitN(inner, " and ", 2)
	if len(parts) != 2 {
		return style.Dim.Render(ln)
	}
	backup, original := parts[0], parts[1]
	sizeInfo := ""
	if a, err := os.Stat(backup); err == nil {
		if c, err := os.Stat(original); err == nil {
			sizeInfo = ", " + style.FormatSize(a.Size()) + " → " + style.FormatSize(c.Size())
		}
	}
	return style.Dim.Render(style.SymBackup) + " " + style.Cyan.Render(style.ShortenPath(original)) +
		style.Dim.Render("  binary"+sizeInfo+" (changed)")
}

// largeFileSummary returns a size summary (and true) when a backed-up file is
// large enough that dumping its full diff would flood the terminal. The wording
// is neutral about whether content changed, since detecting that cheaply isn't
// possible at this size — `oops diff --full` runs the real diff.
func largeFileSummary(bf trash.BackedUpFile) (string, bool) {
	bi, err := os.Stat(bf.Backup)
	if err != nil {
		return "", false
	}
	ci, err := os.Stat(bf.Original)
	if err != nil {
		return "", false
	}
	if bi.Size() < diffSizeThreshold && ci.Size() < diffSizeThreshold {
		return "", false
	}
	return style.Dim.Render(style.SymBackup) + " " + style.Cyan.Render(style.ShortenPath(bf.Original)) +
		style.Dim.Render("  large file "+style.FormatSize(bi.Size())+" → "+style.FormatSize(ci.Size())+" — run oops diff --full to view"), true
}
