package trash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupAndRestore_File(t *testing.T) {
	tmp := t.TempDir()
	origFile := filepath.Join(tmp, "test.txt")
	os.WriteFile(origFile, []byte("hello world"), 0o644)

	// Override trash dir for test
	trashRoot := filepath.Join(tmp, "trash")
	os.MkdirAll(trashRoot, 0o755)

	trashDir, backed, err := Backup("test-001", []string{origFile})
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	if len(backed) != 1 {
		t.Fatalf("expected 1 backed file, got %d", len(backed))
	}
	if backed[0].Original != origFile {
		t.Errorf("expected original %s, got %s", origFile, backed[0].Original)
	}

	// Delete the original
	os.Remove(origFile)
	if _, err := os.Stat(origFile); !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}

	// Restore
	restored, err := Restore(trashDir)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if len(restored) != 1 {
		t.Fatalf("expected 1 restored file, got %d", len(restored))
	}

	data, err := os.ReadFile(origFile)
	if err != nil {
		t.Fatalf("could not read restored file: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(data))
	}
}

func TestBackupAndRestore_Dir(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "mydir")
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("file a"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("file b"), 0o644)

	trashDir, backed, err := Backup("test-002", []string{dir})
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if len(backed) != 1 {
		t.Fatalf("expected 1 backed dir, got %d", len(backed))
	}

	// Delete the original directory
	os.RemoveAll(dir)

	// Restore
	_, err = Restore(trashDir)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	if string(data) != "file a" {
		t.Errorf("expected 'file a', got %q", string(data))
	}
	data, _ = os.ReadFile(filepath.Join(dir, "sub", "b.txt"))
	if string(data) != "file b" {
		t.Errorf("expected 'file b', got %q", string(data))
	}
}

func TestBackup_NonexistentFile(t *testing.T) {
	_, _, err := Backup("test-003", []string{"/nonexistent/file"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSize(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.txt")
	os.WriteFile(f, []byte("0123456789"), 0o644) // 10 bytes

	s := Size(tmp)
	if s != 10 {
		t.Errorf("expected 10 bytes, got %d", s)
	}
}
