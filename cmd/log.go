package cmd

import (
	"fmt"

	"github.com/gedaliah/oops/internal/cleanup"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

var logLimit int

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show the undo history",
	RunE:  runLog,
}

func init() {
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 20, "Number of entries to show")
	rootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, args []string) error {
	cleanup.RunIfNeeded()

	entries, err := journal.Last(logLimit)
	if err != nil {
		return fmt.Errorf("reading journal: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println(style.Dim.Render("No entries in the journal."))
		return nil
	}

	for i, e := range entries {
		num := fmt.Sprintf("[%d]", i+1)
		ts := style.Dim.Render(e.Timestamp)
		risk := formatRisk(e.Risk)
		desc := e.Desc

		undone := ""
		if e.Undone {
			undone = style.Dim.Render(" (undone)")
		}

		fmt.Printf("%s %s %s %s%s\n", style.Bold.Render(num), ts, risk, desc, undone)
		if len(e.Files) > 0 {
			for _, f := range e.Files {
				fmt.Printf("    %s\n", style.Cyan.Render(f))
			}
		}
	}

	return nil
}

func formatRisk(risk string) string {
	switch risk {
	case "high":
		return style.Red.Render("HIGH")
	case "medium":
		return style.Yellow.Render("MED ")
	case "low":
		return style.Dim.Render("LOW ")
	default:
		return style.Dim.Render("    ")
	}
}
