package journal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gofrs/flock"
)

// Entry represents a single journal entry.
type Entry struct {
	ID        string `json:"id"`
	Timestamp string `json:"ts"`
	Command   string `json:"cmd"`
	Action    string `json:"action"`
	Risk      string `json:"risk"`
	TrashDir  string `json:"trash_dir"`
	Desc      string `json:"desc"`
	CWD       string `json:"cwd"`
	Undone    bool   `json:"undone,omitempty"`

	// Git-specific
	GitAction string `json:"git_action,omitempty"`
	GitRef    string `json:"git_ref,omitempty"`
	GitSHA    string `json:"git_sha,omitempty"`
	GitStash  string `json:"git_stash,omitempty"`

	// Files that were backed up
	Files []string `json:"files,omitempty"`
}

// Append adds an entry to the journal with file locking.
func Append(entry Entry) error {
	if err := config.EnsureDir(); err != nil {
		return err
	}

	lock := flock.New(config.JournalPath() + ".lock")
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("acquiring journal lock: %w", err)
	}
	defer lock.Unlock()

	f, err := os.OpenFile(config.JournalPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

// ReadAll reads all journal entries.
func ReadAll() ([]Entry, error) {
	f, err := os.Open(config.JournalPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

// Last returns the N most recent undoable entries.
func Last(n int) ([]Entry, error) {
	all, err := ReadAll()
	if err != nil {
		return nil, err
	}

	// Filter to undoable entries (not already undone)
	var undoable []Entry
	for _, e := range all {
		if !e.Undone {
			undoable = append(undoable, e)
		}
	}

	// Sort by timestamp descending
	sort.Slice(undoable, func(i, j int) bool {
		return undoable[i].Timestamp > undoable[j].Timestamp
	})

	if n > len(undoable) {
		n = len(undoable)
	}
	return undoable[:n], nil
}

// MarkUndone marks an entry as undone by ID.
func MarkUndone(id string) error {
	lock := flock.New(config.JournalPath() + ".lock")
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("acquiring journal lock: %w", err)
	}
	defer lock.Unlock()

	all, err := ReadAll()
	if err != nil {
		return err
	}

	found := false
	for i := range all {
		if all[i].ID == id {
			all[i].Undone = true
			found = true
		}
	}
	if !found {
		return fmt.Errorf("entry %s not found", id)
	}

	return writeAll(all)
}

// DeleteBefore removes entries older than the given time.
func DeleteBefore(t time.Time) (int, error) {
	lock := flock.New(config.JournalPath() + ".lock")
	if err := lock.Lock(); err != nil {
		return 0, fmt.Errorf("acquiring journal lock: %w", err)
	}
	defer lock.Unlock()

	all, err := ReadAll()
	if err != nil {
		return 0, err
	}

	var kept []Entry
	removed := 0
	cutoff := t.Format(time.RFC3339)
	for _, e := range all {
		if e.Timestamp < cutoff {
			removed++
		} else {
			kept = append(kept, e)
		}
	}

	if removed == 0 {
		return 0, nil
	}

	return removed, writeAll(kept)
}

func writeAll(entries []Entry) error {
	f, err := os.Create(config.JournalPath())
	if err != nil {
		return err
	}
	defer f.Close()

	for _, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			continue
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	return nil
}

// GenerateID creates a unique ID for a journal entry.
func GenerateID() string {
	now := time.Now()
	return now.Format("20060102-150405") + "-" + randomHex(4)
}

func randomHex(n int) string {
	b := make([]byte, n)
	f, err := os.Open("/dev/urandom")
	if err != nil {
		// Fallback to time-based
		t := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(t >> (i * 8))
		}
	} else {
		f.Read(b)
		f.Close()
	}
	const hex = "0123456789abcdef"
	out := make([]byte, n*2)
	for i, v := range b {
		out[i*2] = hex[v>>4]
		out[i*2+1] = hex[v&0x0f]
	}
	return string(out)
}
