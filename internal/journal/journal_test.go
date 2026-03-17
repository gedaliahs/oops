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
