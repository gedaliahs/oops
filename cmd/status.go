package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/gedaliah/oops/internal/trash"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show oops health and backup state",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	entries, err := journal.ReadAll()
	if err != nil {
		return fmt.Errorf("reading journal: %w", err)
	}

	undoable := 0
	kept := 0
	protected := 0
	var newest *journal.Entry
	now := time.Now()
	for i := range entries {
		entry := entries[i]
		if !entry.Undone {
			undoable++
			if newest == nil || entry.Timestamp > newest.Timestamp {
				newest = &entry
			}
		}
		if entry.CleanupProtected(now) {
			kept++
		}
		if entry.Protected {
			protected++
		}
	}

	fmt.Println(style.Bold.Render("oops status"))
	fmt.Println(style.Dim.Render("  Version: ") + "v" + Version + latestVersionSuffix())
	fmt.Printf("%s %d hours\n", style.Dim.Render("  Retention:"), cfg.RetentionHours)
	fmt.Printf("%s %s / %s\n", style.Dim.Render("  Trash:"), style.FormatSize(trash.TotalSize()), style.FormatSize(cfg.MaxTrashBytes))
	fmt.Printf("%s %d\n", style.Dim.Render("  Undoable entries:"), undoable)
	fmt.Printf("%s %d\n", style.Dim.Render("  Kept entries:"), kept)
	fmt.Printf("%s %d\n", style.Dim.Render("  Protected entries:"), protected)
	fmt.Printf("%s %d\n", style.Dim.Render("  Protected paths:"), len(cfg.ProtectedPaths))
	fmt.Printf("%s %s\n", style.Dim.Render("  Confirm mode:"), cfg.ConfirmMode)
	fmt.Printf("%s %v\n", style.Dim.Render("  Risk warnings:"), cfg.RiskWarning)
	if os.Getenv("OOPS_HOOK") == "1" {
		fmt.Println(style.Dim.Render("  Hook loaded: ") + "yes")
	} else {
		fmt.Println(style.Dim.Render("  Hook loaded: ") + "not in this process")
	}
	if newest != nil {
		fmt.Println(style.Dim.Render("  Last action: ") + style.ShortenPath(newest.Desc))
	}

	return nil
}

func latestVersionSuffix() string {
	latest, err := fetchLatestVersion()
	if err != nil || latest == "" || compareVersions(latest, Version) <= 0 {
		return ""
	}
	return style.Yellow.Render(" (latest v" + latest + ")")
}
