package style

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

var (
	Green  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	Yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("#eab308"))
	Red    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	Dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	Bold   = lipgloss.NewStyle().Bold(true)
	Cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("#06b6d4"))

	// Diff rendering
	DiffAdd    = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	DiffDel    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	DiffHunk   = lipgloss.NewStyle().Foreground(lipgloss.Color("#06b6d4"))
	DiffHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")).Bold(true)
)

const (
	SymRestore = "↩"
	SymBackup  = "●"
	SymWarn    = "▲"
	SymOK      = "✓"
	SymFail    = "✗"
	SymTrash   = "🗑"
)

func Success(msg string) string {
	return Green.Render(SymOK) + " " + msg
}

func Warning(msg string) string {
	return Yellow.Render(SymWarn) + " " + msg
}

func Error(msg string) string {
	return Red.Render(SymFail) + " " + msg
}

func Backed(path string) string {
	return Dim.Render(SymBackup) + " backed up " + Cyan.Render(ShortenPath(path))
}

func Restored(path string) string {
	return Green.Render(SymRestore) + " restored " + Cyan.Render(ShortenPath(path))
}

// ShortenPath replaces the home directory with ~ for cleaner display.
func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	return strings.Replace(path, home, "~", 1)
}

func Banner() string {
	return Bold.Render("oops") + Dim.Render(" — terminal undo")
}

func FormatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// catchSymbol returns the leading glyph for a catch notice: a warning triangle
// for high-risk commands, a quiet bullet otherwise.
func catchSymbol(highRisk bool) string {
	if highRisk {
		return Yellow.Render(SymWarn)
	}
	return Dim.Render(SymBackup)
}

// FirstCatchNudge is the two-line teaching notice shown the very first time oops
// backs something up, so a brand-new user learns the tool exists and how to undo.
func FirstCatchNudge(shortDesc string, highRisk bool) string {
	head := catchSymbol(highRisk) + " " + shortDesc + Dim.Render("  · backed up")
	sub := "    " + Dim.Render("Type ") + Bold.Render("oops") + Dim.Render(" to undo · ") +
		Bold.Render("oops tutorial") + Dim.Render(" to learn more")
	return head + "\n" + sub
}

// CatchHint is the single-line notice shown while a new user is still learning:
// it carries the recovery instruction next to the backed-up command.
func CatchHint(shortDesc string, highRisk bool) string {
	return catchSymbol(highRisk) + " " + shortDesc + Dim.Render("  · backed up, type oops to undo")
}

// RelativeTime renders an RFC3339 timestamp as a human relative string
// ("just now", "3 min ago", "yesterday"), falling back to the raw value on a
// parse error and to an absolute local date for anything older than a week.
func RelativeTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	d := time.Since(t)
	switch {
	case d < 0:
		return t.Local().Format("Jan 2 15:04")
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hr ago"
		}
		return fmt.Sprintf("%d hr ago", h)
	case d < 48*time.Hour:
		return "yesterday"
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	default:
		return t.Local().Format("Jan 2 2006")
	}
}

// DayBucket groups a timestamp into "Today", "Yesterday", or "Earlier" using the
// local calendar day. Returns "" when the timestamp cannot be parsed.
func DayBucket(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	now := time.Now()
	ty, tm, td := t.Local().Date()
	ny, nm, nd := now.Date()
	if ty == ny && tm == nm && td == nd {
		return "Today"
	}
	yy, ym, yd := now.AddDate(0, 0, -1).Date()
	if ty == yy && tm == ym && td == yd {
		return "Yesterday"
	}
	return "Earlier"
}

// UsageBar renders a fixed-width horizontal bar that turns green/yellow/red as it
// fills, followed by the percentage. Callers should guard the block-glyph output
// behind IsTTY and fall back to plain text when piped.
func UsageBar(used, total int64, width int) string {
	if width < 1 {
		width = 10
	}
	ratio := 0.0
	if total > 0 {
		ratio = float64(used) / float64(total)
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	if used > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	barStyle := Green
	switch {
	case ratio >= 0.9:
		barStyle = Red
	case ratio >= 0.7:
		barStyle = Yellow
	}
	full := strings.Repeat("█", filled)
	empty := strings.Repeat("░", width-filled)
	return "[" + barStyle.Render(full) + Dim.Render(empty) + "] " + fmt.Sprintf("%d%%", int(ratio*100+0.5))
}

// IsTTY reports whether standard output is an interactive terminal. Used to gate
// glyph-heavy rendering (block bars) that should degrade to plain text when piped.
func IsTTY() bool {
	return term.IsTerminal(os.Stdout.Fd())
}
