package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gedaliah/oops/internal/cleanup"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

var (
	logLimit    int
	logRisk     string
	logPathFlag string
	logHere     bool
	logAbsolute bool
	logFlat     bool
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show the undo history",
	RunE:  runLog,
}

func init() {
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 20, "Number of entries to show")
	logCmd.Flags().StringVar(&logRisk, "risk", "", "Filter by risk: high, medium, or low")
	logCmd.Flags().StringVar(&logPathFlag, "path", "", "Filter to entries whose command or files contain this text")
	logCmd.Flags().BoolVar(&logHere, "here", false, "Filter to actions taken in the current directory (or below)")
	logCmd.Flags().BoolVar(&logAbsolute, "absolute", false, "Show absolute timestamps instead of relative")
	logCmd.Flags().BoolVar(&logFlat, "flat", false, "Disable day/directory grouping headers")
	rootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, args []string) error {
	cleanup.RunIfNeeded()

	risk, err := normalizeRiskFilter(logRisk)
	if err != nil {
		return err
	}

	filtersActive := risk != "" || logPathFlag != "" || logHere
	// journal.Last returns undoable entries newest-first; an entry's position here
	// (i+1) is exactly the N that `oops N` restores. We must label rows with that
	// true index and never renumber a filtered subset. When a filter is active we
	// scan the whole history so matches beyond the default window are still found
	// and still carry their correct global index.
	scan := logLimit
	if filtersActive {
		scan = 1 << 30
	}
	entries, err := journal.Last(scan)
	if err != nil {
		return fmt.Errorf("reading journal: %w", err)
	}
	if len(entries) == 0 {
		fmt.Println(style.Dim.Render("No entries in the journal."))
		return nil
	}

	cwd, _ := os.Getwd()
	rows := selectLogRows(entries, risk, logPathFlag, cwd, logHere)

	if len(rows) == 0 {
		fmt.Println(style.Dim.Render("No matching entries."))
		return nil
	}

	hidden := 0
	if len(rows) > logLimit {
		hidden = len(rows) - logLimit
		rows = rows[:logLimit]
	}

	prevBucket, prevCWD := "", ""
	first := true
	for _, r := range rows {
		e := r.entry
		if !logFlat {
			bucket := style.DayBucket(e.Timestamp)
			if bucket != prevBucket || e.CWD != prevCWD {
				if !first {
					fmt.Println()
				}
				header := bucket
				if e.CWD != "" {
					if header != "" {
						header += " · "
					}
					header += style.ShortenPath(e.CWD)
				}
				if header != "" {
					fmt.Println(style.Dim.Render(header))
				}
				prevBucket, prevCWD = bucket, e.CWD
			}
		}
		first = false

		num := fmt.Sprintf("[%d]", r.index)
		ts := e.Timestamp
		if !logAbsolute {
			ts = style.RelativeTime(e.Timestamp)
		}
		riskCol := formatRisk(e.Risk)
		desc := style.ShortenPath(e.Desc)

		state := ""
		if e.Undone {
			state += style.Dim.Render(" (undone)")
		}
		if e.Pinned {
			state += style.Green.Render(" (kept)")
		}
		if e.Protected {
			state += style.Yellow.Render(" (protected)")
		}

		fmt.Printf("%s %s %s %s%s\n", style.Bold.Render(num), style.Dim.Render(ts), riskCol, desc, state)
		for _, f := range e.Files {
			fmt.Printf("    %s\n", style.Cyan.Render(style.ShortenPath(f)))
		}
	}

	if hidden > 0 {
		fmt.Println(style.Dim.Render(fmt.Sprintf("… and %d more (use -n to show more)", hidden)))
	}

	return nil
}

// logRow pairs an entry with its true `oops N` index — the position of the entry
// in the unfiltered, undoable, newest-first ordering returned by journal.Last,
// which is exactly the number `oops N` restores.
type logRow struct {
	index int
	entry journal.Entry
}

// selectLogRows applies the active filters while preserving each surviving
// entry's true index. It must never renumber the filtered subset, or `oops N`
// would stop matching the [N] printed by `oops log`.
func selectLogRows(entries []journal.Entry, risk, pathQuery, cwd string, here bool) []logRow {
	var rows []logRow
	for i, e := range entries {
		if risk != "" && e.Risk != risk {
			continue
		}
		if here && !entryInDir(e, cwd) {
			continue
		}
		if pathQuery != "" && !entryMatchesPath(e, pathQuery) {
			continue
		}
		rows = append(rows, logRow{index: i + 1, entry: e})
	}
	return rows
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

func normalizeRiskFilter(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return "", nil
	case "high", "h":
		return "high", nil
	case "medium", "med", "m":
		return "medium", nil
	case "low", "l":
		return "low", nil
	default:
		return "", fmt.Errorf("invalid --risk %q (use high, medium, or low)", s)
	}
}

func expandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// pathWithin reports whether target is root itself or nested under it.
func pathWithin(root, target string) bool {
	if root == "" || target == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel))
}

func entryInDir(e journal.Entry, cwd string) bool {
	if cwd == "" {
		return true
	}
	if pathWithin(cwd, e.CWD) {
		return true
	}
	for _, f := range e.Files {
		if pathWithin(cwd, f) {
			return true
		}
	}
	return false
}

// entryMatchesPath does a case-insensitive substring match of the query against
// the entry's command, description, and paths — in both absolute and ~-shortened
// forms, and with ~ in the query expanded — so `--path ~/proj`, `--path /abs`,
// and `--path proj` all work against the absolute paths stored in the journal.
func entryMatchesPath(e journal.Entry, query string) bool {
	if query == "" {
		return true
	}
	needles := []string{strings.ToLower(query)}
	if exp := expandTilde(query); exp != query {
		needles = append(needles, strings.ToLower(exp))
	}

	haystacks := []string{
		e.CWD, style.ShortenPath(e.CWD),
		e.Desc, style.ShortenPath(e.Desc),
		e.Command,
	}
	for _, f := range e.Files {
		haystacks = append(haystacks, f, style.ShortenPath(f))
	}

	for _, h := range haystacks {
		hl := strings.ToLower(h)
		for _, n := range needles {
			if n != "" && strings.Contains(hl, n) {
				return true
			}
		}
	}
	return false
}
