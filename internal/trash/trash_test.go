package trash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gedaliah/oops/internal/config"
)

func setupTestTrash(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
}

func TestBackupAndRestore_File(t *testing.T) {
	setupTestTrash(t)
	tmp := t.TempDir()
	origFile := filepath.Join(tmp, "test.txt")
	os.WriteFile(origFile, []byte("hello world"), 0o644)

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

func TestBackupCopiesFileSoOverwriteCannotMutateBackup(t *testing.T) {
	setupTestTrash(t)
	tmp := t.TempDir()
	origFile := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(origFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	trashDir, _, err := Backup("test-copy", []string{origFile})
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if err := os.WriteFile(origFile, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreWithOptions(trashDir, RestoreOptions{Overwrite: true}); err != nil {
		t.Fatalf("RestoreWithOptions failed: %v", err)
	}
	data, err := os.ReadFile(origFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("expected copied backup to preserve original content, got %q", string(data))
	}
}

func TestRestoreWithOptionsConflictRequiresExplicitChoice(t *testing.T) {
	setupTestTrash(t)
	tmp := t.TempDir()
	origFile := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(origFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	trashDir, _, err := Backup("test-conflict", []string{origFile})
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if err := os.WriteFile(origFile, []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := RestoreWithOptions(trashDir, RestoreOptions{}); err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestRestoreWithOptionsBackupCurrent(t *testing.T) {
	setupTestTrash(t)
	tmp := t.TempDir()
	origFile := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(origFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	trashDir, _, err := Backup("test-backup-current", []string{origFile})
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if err := os.WriteFile(origFile, []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}

	restored, err := RestoreWithOptions(trashDir, RestoreOptions{BackupCurrent: true})
	if err != nil {
		t.Fatalf("RestoreWithOptions failed: %v", err)
	}
	if len(restored) != 1 || restored[0].BackupCurrent == "" {
		t.Fatalf("expected backup-current path, got %+v", restored)
	}
	data, err := os.ReadFile(origFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("expected original restore, got %q", string(data))
	}
	current, err := os.ReadFile(restored[0].BackupCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if string(current) != "current" {
		t.Fatalf("expected current file to be moved aside, got %q", string(current))
	}
}

func TestRestoreWithOptionsToDir(t *testing.T) {
	setupTestTrash(t)
	tmp := t.TempDir()
	origFile := filepath.Join(tmp, "nested", "test.txt")
	if err := os.MkdirAll(filepath.Dir(origFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(origFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	trashDir, _, err := Backup("test-to-dir", []string{origFile})
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	restoreDir := filepath.Join(tmp, "restore")
	restored, err := RestoreWithOptions(trashDir, RestoreOptions{ToDir: restoreDir})
	if err != nil {
		t.Fatalf("RestoreWithOptions failed: %v", err)
	}
	if len(restored) != 1 {
		t.Fatalf("expected one restored file, got %d", len(restored))
	}
	data, err := os.ReadFile(restored[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("expected restore copy, got %q", string(data))
	}
	if !strings.HasPrefix(restored[0].Path, restoreDir) {
		t.Fatalf("expected restore under %s, got %s", restoreDir, restored[0].Path)
	}
}

func TestBackupAndRestore_Dir(t *testing.T) {
	setupTestTrash(t)
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
	setupTestTrash(t)
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

func TestRemoveRejectsPathOutsideTrashWithSharedPrefix(t *testing.T) {
	setupTestTrash(t)

	outside := config.TrashDir() + "-sibling"
	outsideEntry := filepath.Join(outside, "entry")
	if err := os.MkdirAll(outsideEntry, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Remove(outsideEntry); err == nil {
		t.Fatal("expected Remove to reject path outside trash")
	}
	if _, err := os.Stat(outsideEntry); err != nil {
		t.Fatalf("outside path should not be removed: %v", err)
	}
}

func TestRemoveAllowsTrashChild(t *testing.T) {
	setupTestTrash(t)

	trashEntry := filepath.Join(config.TrashDir(), "entry")
	if err := os.MkdirAll(trashEntry, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Remove(trashEntry); err != nil {
		t.Fatalf("expected trash child removal to succeed: %v", err)
	}
	if _, err := os.Stat(trashEntry); !os.IsNotExist(err) {
		t.Fatalf("expected trash child to be removed, stat err=%v", err)
	}
}
