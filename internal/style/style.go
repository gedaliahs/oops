package style

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	Green  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	Yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("#eab308"))
	Red    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	Dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	Bold   = lipgloss.NewStyle().Bold(true)
	Cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("#06b6d4"))
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
	return Dim.Render(SymBackup) + " backed up " + Cyan.Render(path)
}

func Restored(path string) string {
	return Green.Render(SymRestore) + " restored " + Cyan.Render(path)
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
