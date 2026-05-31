package cmd

import (
	"path/filepath"
	"testing"

	"github.com/gedaliah/oops/internal/journal"
)

func TestNormalizeRiskFilter(t *testing.T) {
	cases := map[string]string{
		"":       "",
		"high":   "high",
		"HIGH":   "high",
		"h":      "high",
		"med":    "medium",
		"medium": "medium",
		"low":    "low",
		"  low ": "low",
	}
	for in, want := range cases {
		got, err := normalizeRiskFilter(in)
		if err != nil {
			t.Fatalf("normalizeRiskFilter(%q) returned error: %v", in, err)
		}
		if got != want {
			t.Fatalf("normalizeRiskFilter(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := normalizeRiskFilter("bogus"); err == nil {
		t.Fatal("expected error for invalid risk filter")
	}
}

func TestSelectLogRowsPreservesIndex(t *testing.T) {
	// Simulates journal.Last output (newest-first). The index `oops N` restores
	// is the position in THIS slice, so filtering must keep those indices intact.
	entries := []journal.Entry{
		{Risk: "high", CWD: "/home/u/projA", Desc: "rm -r /home/u/projA/src"},  // oops 1
		{Risk: "medium", CWD: "/home/u/projB", Desc: "sed -i /home/u/projB/x"}, // oops 2
		{Risk: "high", CWD: "/home/u/projA", Desc: "git reset --hard"},         // oops 3
		{Risk: "low", CWD: "/home/u/projC", Desc: "cp a b"},                    // oops 4
	}

	rows := selectLogRows(entries, "high", "", "", false)
	if len(rows) != 2 {
		t.Fatalf("expected 2 high-risk rows, got %d", len(rows))
	}
	if rows[0].index != 1 || rows[1].index != 3 {
		t.Fatalf("filtered rows must keep their true oops N (1, 3), got (%d, %d)", rows[0].index, rows[1].index)
	}

	// Path filter narrows to projB's entry, which must still be oops 2.
	rows = selectLogRows(entries, "", "projB", "", false)
	if len(rows) != 1 || rows[0].index != 2 {
		t.Fatalf("expected single row with index 2, got %+v", rows)
	}

	// No filters: every entry present and index == position+1.
	rows = selectLogRows(entries, "", "", "", false)
	for i, r := range rows {
		if r.index != i+1 {
			t.Fatalf("row %d has index %d, want %d", i, r.index, i+1)
		}
	}
}

func TestEntryInDir(t *testing.T) {
	root := filepath.Join("/tmp", "proj")
	sub := filepath.Join(root, "src")
	other := filepath.Join("/tmp", "other")

	if !entryInDir(journal.Entry{CWD: root}, root) {
		t.Fatal("entry in the dir itself should match")
	}
	if !entryInDir(journal.Entry{CWD: sub}, root) {
		t.Fatal("entry in a subdir should match")
	}
	if entryInDir(journal.Entry{CWD: other}, root) {
		t.Fatal("entry in an unrelated dir should not match")
	}
	if !entryInDir(journal.Entry{CWD: other, Files: []string{filepath.Join(sub, "a.go")}}, root) {
		t.Fatal("entry whose file is under the dir should match")
	}
	if !entryInDir(journal.Entry{CWD: other}, "") {
		t.Fatal("empty cwd should match all (filter is a no-op)")
	}
}

func TestEntryMatchesPath(t *testing.T) {
	e := journal.Entry{
		CWD:     "/home/u/project",
		Desc:    "rm -r /home/u/project/src",
		Command: "rm -r src",
		Files:   []string{"/home/u/project/src/main.go"},
	}
	for _, q := range []string{"project", "main.go", "PROJECT", "/home/u/project"} {
		if !entryMatchesPath(e, q) {
			t.Fatalf("expected %q to match", q)
		}
	}
	if entryMatchesPath(e, "nonexistent-token") {
		t.Fatal("unrelated query should not match")
	}
	if !entryMatchesPath(e, "") {
		t.Fatal("empty query should match (filter is a no-op)")
	}
}

func TestExpandTilde(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	if got := expandTilde("~/proj"); got != filepath.Join("/home/tester", "proj") {
		t.Fatalf("expandTilde(~/proj) = %q", got)
	}
	if got := expandTilde("/abs/path"); got != "/abs/path" {
		t.Fatalf("expandTilde should leave absolute paths unchanged, got %q", got)
	}
	if got := expandTilde("relative"); got != "relative" {
		t.Fatalf("expandTilde should leave non-tilde paths unchanged, got %q", got)
	}
}
