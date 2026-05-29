package cmd

import (
	"fmt"
	"os"
	"strconv"
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

	fmt.Println(style.Bold.Render("oops status") + style.Dim.Render("  v"+Version) + latestVersionSuffix())

	fmt.Println(style.Bold.Render("Backups"))
	statusRow("Trash", trashUsage(trash.TotalSize(), cfg.MaxTrashBytes))
	statusRow("Retention", fmt.Sprintf("%d hours", cfg.RetentionHours))
	statusRow("Undoable", strconv.Itoa(undoable))
	statusRow("Kept", strconv.Itoa(kept))
	statusRow("Protected", strconv.Itoa(protected))

	fmt.Println(style.Bold.Render("Protection"))
	statusRow("Confirm mode", confirmModeValue(cfg.ConfirmMode))
	statusRow("Risk warnings", boolHealth(cfg.RiskWarning, "on", "off"))
	statusRow("Onboarding", boolHealth(cfg.OnboardingHints, "on", "off"))
	statusRow("Protected paths", strconv.Itoa(len(cfg.ProtectedPaths)))

	fmt.Println(style.Bold.Render("System"))
	if os.Getenv("OOPS_HOOK") == "1" {
		statusRow("Hook loaded", style.Green.Render(style.SymOK+" yes"))
	} else {
		statusRow("Hook loaded", style.Yellow.Render(style.SymWarn+" not in this process"))
	}
	if newest != nil {
		statusRow("Last action", style.ShortenPath(newest.Desc)+style.Dim.Render("  ("+style.RelativeTime(newest.Timestamp)+")"))
	}

	return nil
}

// statusRow prints a dimmed, fixed-width label followed by its value. The label
// is padded before styling so alignment is unaffected by ANSI codes.
func statusRow(label, value string) {
	fmt.Println(style.Dim.Render(fmt.Sprintf("  %-16s", label)) + value)
}

// trashUsage renders a colored usage bar on a TTY, falling back to plain text
// when piped (Lipgloss strips color but not the bar's block glyphs).
func trashUsage(used, total int64) string {
	sizeText := style.FormatSize(used) + " / " + style.FormatSize(total)
	pct := 0
	if total > 0 {
		pct = int(float64(used)/float64(total)*100 + 0.5)
	}
	if style.IsTTY() {
		return style.UsageBar(used, total, 12) + "   " + sizeText
	}
	return fmt.Sprintf("%s (%d%%)", sizeText, pct)
}

func confirmModeValue(mode string) string {
	switch mode {
	case "all", "high":
		return style.Green.Render(mode)
	default:
		return style.Dim.Render("off")
	}
}

func boolHealth(on bool, yes, no string) string {
	if on {
		return style.Green.Render(style.SymOK + " " + yes)
	}
	return style.Yellow.Render(style.SymWarn + " " + no)
}

func latestVersionSuffix() string {
	latest, err := fetchLatestVersion()
	if err != nil || latest == "" || compareVersions(latest, Version) <= 0 {
		return ""
	}
	return style.Yellow.Render(" (latest v" + latest + ")")
}
