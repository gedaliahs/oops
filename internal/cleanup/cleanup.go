package cleanup

import (
	"os"
	"time"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/trash"
)

// RunIfNeeded checks if cleanup should run and executes it lazily.
func RunIfNeeded() {
	if !config.ShouldCleanup() {
		return
	}
	config.MarkCleanup()
	Run(config.Load())
}

// Run performs cleanup: removes old entries and enforces max trash size.
func Run(cfg config.Config) (removedEntries int, freedBytes int64) {
	// 1. Remove entries older than retention period
	cutoff := time.Now().Add(-time.Duration(cfg.RetentionDays) * 24 * time.Hour)
	removed, _ := journal.DeleteBefore(cutoff)
	removedEntries = removed

	// Clean up orphaned trash dirs
	dirs, err := trash.ListTrashDirs()
	if err != nil {
		return
	}

	entries, _ := journal.ReadAll()
	activeTrash := make(map[string]bool)
	for _, e := range entries {
		if e.TrashDir != "" {
			activeTrash[e.TrashDir] = true
		}
	}

	for _, d := range dirs {
		if !activeTrash[d] {
			size := trash.Size(d)
			if err := trash.Remove(d); err == nil {
				freedBytes += size
			}
		}
	}

	// 2. Enforce max trash size (remove oldest first)
	total := trash.TotalSize()
	if total > cfg.MaxTrashBytes {
		dirs, _ = trash.ListTrashDirs()
		// dirs are newest first, so remove from the end
		for i := len(dirs) - 1; i >= 0 && total > cfg.MaxTrashBytes; i-- {
			size := trash.Size(dirs[i])
			if err := trash.Remove(dirs[i]); err == nil {
				freedBytes += size
				total -= size
			}
		}
	}

	return
}

// Purge removes all trash and clears the journal.
func Purge() error {
	if err := os.RemoveAll(config.TrashDir()); err != nil {
		return err
	}
	if err := os.MkdirAll(config.TrashDir(), 0o755); err != nil {
		return err
	}
	return os.Remove(config.JournalPath())
}

// PurgeBefore removes trash and journal entries older than a given time.
func PurgeBefore(t time.Time) (int, int64, error) {
	removed, err := journal.DeleteBefore(t)
	if err != nil {
		return 0, 0, err
	}

	// Clean orphaned dirs
	dirs, _ := trash.ListTrashDirs()
	entries, _ := journal.ReadAll()
	activeTrash := make(map[string]bool)
	for _, e := range entries {
		if e.TrashDir != "" {
			activeTrash[e.TrashDir] = true
		}
	}

	var freed int64
	for _, d := range dirs {
		if !activeTrash[d] {
			size := trash.Size(d)
			if err := trash.Remove(d); err == nil {
				freed += size
			}
		}
	}

	return removed, freed, nil
}
