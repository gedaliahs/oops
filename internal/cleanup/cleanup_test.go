package cleanup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/journal"
)

func TestRunRemovesTrashOlderThanRetentionHours(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	oldTrash := filepath.Join(config.TrashDir(), "old")
	recentTrash := filepath.Join(config.TrashDir(), "recent")
	if err := os.MkdirAll(oldTrash, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(recentTrash, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldTrash, "old.txt"), []byte("old backup"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recentTrash, "recent.txt"), []byte("recent backup"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	if err := journal.Append(journal.Entry{
		ID:        "old",
		Timestamp: now.Add(-3 * time.Hour).Format(time.RFC3339),
		Command:   "rm old.txt",
		TrashDir:  oldTrash,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Append(journal.Entry{
		ID:        "recent",
		Timestamp: now.Add(-90 * time.Minute).Format(time.RFC3339),
		Command:   "rm recent.txt",
		TrashDir:  recentTrash,
	}); err != nil {
		t.Fatal(err)
	}

	removed, freed := Run(config.Config{
		RetentionHours: 2,
		MaxTrashBytes:  1 << 60,
	})

	if removed != 1 {
		t.Fatalf("expected 1 old entry removed, got %d", removed)
	}
	if freed == 0 {
		t.Fatal("expected old trash bytes to be freed")
	}
	if _, err := os.Stat(oldTrash); !os.IsNotExist(err) {
		t.Fatalf("expected old trash to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(recentTrash); err != nil {
		t.Fatalf("expected recent trash to remain: %v", err)
	}

	entries, err := journal.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "recent" {
		t.Fatalf("expected only recent entry to remain, got %+v", entries)
	}
}

func TestPurgeSucceedsWithoutJournal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := os.MkdirAll(config.TrashDir(), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Purge(); err != nil {
		t.Fatalf("expected purge without journal to succeed: %v", err)
	}

	if _, err := os.Stat(config.TrashDir()); err != nil {
		t.Fatalf("expected trash dir to be recreated: %v", err)
	}
}

func TestRunDoesNotPrunePinnedTrashForSizeLimit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	pinnedTrash := filepath.Join(config.TrashDir(), "pinned")
	oldTrash := filepath.Join(config.TrashDir(), "old")
	if err := os.MkdirAll(pinnedTrash, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(oldTrash, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pinnedTrash, "pinned.txt"), []byte("pinned"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldTrash, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	if err := journal.Append(journal.Entry{
		ID:        "pinned",
		Timestamp: now.Format(time.RFC3339),
		TrashDir:  pinnedTrash,
		Pinned:    true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Append(journal.Entry{
		ID:        "old",
		Timestamp: now.Format(time.RFC3339),
		TrashDir:  oldTrash,
	}); err != nil {
		t.Fatal(err)
	}

	Run(config.Config{RetentionHours: 24, MaxTrashBytes: 1})

	if _, err := os.Stat(pinnedTrash); err != nil {
		t.Fatalf("expected pinned trash to remain: %v", err)
	}
	if _, err := os.Stat(oldTrash); !os.IsNotExist(err) {
		t.Fatalf("expected unpinned trash to be pruned, stat err=%v", err)
	}
}
