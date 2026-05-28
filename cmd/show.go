package cmd

import (
	"fmt"
	"strconv"

	"github.com/gedaliah/oops/internal/cleanup"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:     "show [N]",
	Aliases: []string{"inspect", "preview"},
	Short:   "Preview what an undo would restore",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
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
		fmt.Println(style.Warning("Nothing to show"))
		return nil
	}
	if n > len(entries) {
		return fmt.Errorf("only %d undoable actions in history", len(entries))
	}

	printEntryPreview(entries[n-1])
	return nil
}

func printEntryPreview(entry journal.Entry) {
	fmt.Println(style.Bold.Render("Undo preview"))
	fmt.Println(style.Dim.Render("  ID: ") + entry.ID)
	fmt.Println(style.Dim.Render("  Time: ") + entry.Timestamp)
	fmt.Println(style.Dim.Render("  Risk: ") + entry.Risk)
	if entry.Pinned {
		fmt.Println(style.Dim.Render("  Keep: ") + "yes")
	}
	if entry.Protected {
		fmt.Println(style.Dim.Render("  Protected: ") + "yes")
	}
	if entry.KeepUntil != "" {
		fmt.Println(style.Dim.Render("  Keep until: ") + entry.KeepUntil)
	}
	fmt.Println(style.Dim.Render("  Command: ") + style.ShortenPath(entry.Command))
	fmt.Println(style.Dim.Render("  Action: ") + style.ShortenPath(entry.Desc))

	if entry.GitAction != "" {
		fmt.Println(style.Dim.Render("  Git restore: ") + entry.GitAction)
		if entry.GitRef != "" {
			fmt.Println(style.Dim.Render("  Ref: ") + entry.GitRef)
		}
		if entry.GitSHA != "" {
			fmt.Println(style.Dim.Render("  SHA: ") + entry.GitSHA)
		}
		if entry.GitStash != "" {
			fmt.Println(style.Dim.Render("  Stash: ") + entry.GitStash)
		}
		return
	}

	if len(entry.Files) == 0 {
		fmt.Println(style.Dim.Render("  Files: none recorded"))
		return
	}
	fmt.Println(style.Dim.Render("  Files:"))
	for _, file := range entry.Files {
		fmt.Println("    " + style.Cyan.Render(style.ShortenPath(file)))
	}
}
