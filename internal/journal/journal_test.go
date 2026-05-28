package journal

import (
	"os"
	"testing"
	"time"
)

func setupTestJournal(t *testing.T) func() {
	t.Helper()
	tmp := t.TempDir()

	// Override config paths for testing
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	os.MkdirAll(tmp+"/.oops", 0o755)

	return func() {
		os.Setenv("HOME", origHome)
	}
}

func TestAppendAndRead(t *testing.T) {
	cleanup := setupTestJournal(t)
	defer cleanup()

	entry := Entry{
		ID:        "test-001",
		Timestamp: time.Now().Format(time.RFC3339),
		Command:   "rm test.txt",
		Action:    "rm",
		Risk:      "medium",
		Desc:      "rm test.txt",
	}

	if err := Append(entry); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	entries, err := ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "test-001" {
		t.Errorf("expected ID test-001, got %s", entries[0].ID)
	}
}

func TestLast(t *testing.T) {
	cleanup := setupTestJournal(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		Append(Entry{
			ID:        GenerateID(),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			Command:   "rm file",
			Action:    "rm",
			Risk:      "medium",
			Desc:      "rm file",
		})
	}

	last, err := Last(3)
	if err != nil {
		t.Fatalf("Last failed: %v", err)
	}
	if len(last) != 3 {
		t.Fatalf("expected 3, got %d", len(last))
	}

	// Should be sorted newest first
	if last[0].Timestamp < last[1].Timestamp {
		t.Error("entries not sorted newest first")
	}
}

func TestMarkUndone(t *testing.T) {
	cleanup := setupTestJournal(t)
	defer cleanup()

	Append(Entry{
		ID:        "undo-me",
		Timestamp: time.Now().Format(time.RFC3339),
		Command:   "rm file",
		Action:    "rm",
	})

	if err := MarkUndone("undo-me"); err != nil {
		t.Fatalf("MarkUndone failed: %v", err)
	}

	// Should not appear in Last()
	last, _ := Last(10)
	for _, e := range last {
		if e.ID == "undo-me" {
			t.Error("undone entry should not appear in Last()")
		}
	}
}

func TestMarkPinnedAndDeleteBefore(t *testing.T) {
	cleanup := setupTestJournal(t)
	defer cleanup()

	old := time.Now().Add(-3 * time.Hour).Format(time.RFC3339)
	if err := Append(Entry{ID: "keep-me", Timestamp: old, Command: "rm kept"}); err != nil {
		t.Fatal(err)
	}
	if err := Append(Entry{ID: "drop-me", Timestamp: old, Command: "rm old"}); err != nil {
		t.Fatal(err)
	}
	if err := MarkPinned("keep-me", true); err != nil {
		t.Fatalf("MarkPinned failed: %v", err)
	}

	removed, err := DeleteBefore(time.Now().Add(-2 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 unpinned entry removed, got %d", removed)
	}

	entries, err := ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "keep-me" || !entries[0].Pinned {
		t.Fatalf("expected pinned entry to remain, got %+v", entries)
	}
}

func TestDeleteBeforeKeepsEntryUntilKeepUntilExpires(t *testing.T) {
	cleanup := setupTestJournal(t)
	defer cleanup()

	old := time.Now().Add(-3 * time.Hour).Format(time.RFC3339)
	if err := Append(Entry{
		ID:        "protected",
		Timestamp: old,
		Command:   "rm protected",
		KeepUntil: time.Now().Add(time.Hour).Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := Append(Entry{ID: "drop-me", Timestamp: old, Command: "rm old"}); err != nil {
		t.Fatal(err)
	}

	removed, err := DeleteBefore(time.Now().Add(-2 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 expired entry removed, got %d", removed)
	}

	entries, err := ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "protected" {
		t.Fatalf("expected protected entry to remain, got %+v", entries)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()
	if id1 == id2 {
		t.Error("IDs should be unique")
	}
	if len(id1) < 10 {
		t.Errorf("ID too short: %s", id1)
	}
}
