package trash

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gedaliah/oops/internal/config"
)

// Manifest describes the backed-up files for an action.
type Manifest struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"ts"`
	Command   string         `json:"cmd"`
	Files     []BackedUpFile `json:"files"`
}

// BackedUpFile records where a file was backed up from and to.
type BackedUpFile struct {
	Original string      `json:"original"`
	Backup   string      `json:"backup"`
	IsDir    bool        `json:"is_dir,omitempty"`
	HardLink bool        `json:"hard_link,omitempty"`
	Mode     os.FileMode `json:"mode"`
	UID      int         `json:"uid,omitempty"`
	GID      int         `json:"gid,omitempty"`
}

type RestoreOptions struct {
	Overwrite     bool
	BackupCurrent bool
	ToDir         string
}

type RestoredFile struct {
	Path          string
	BackupCurrent string
}

type ConflictError struct {
	Path string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s already exists; use --overwrite, --backup-current, or --to <dir>", e.Path)
}

// Backup backs up the given files into a new trash directory.
// It copies data instead of hard-linking so overwrites and in-place edits cannot
// mutate the backup through a shared inode.
func Backup(id string, files []string) (string, []BackedUpFile, error) {
	trashDir := filepath.Join(config.TrashDir(), id)
	filesDir := filepath.Join(trashDir, "files")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("creating trash dir: %w", err)
	}

	var backed []BackedUpFile

	for _, f := range files {
		absPath, err := filepath.Abs(f)
		if err != nil {
			continue
		}

		info, err := os.Lstat(absPath)
		if err != nil {
			continue
		}

		backupPath := filepath.Join(filesDir, absPath)
		backupDir := filepath.Dir(backupPath)
		if err := os.MkdirAll(backupDir, 0o755); err != nil {
			continue
		}

		var uid, gid int
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			uid = int(stat.Uid)
			gid = int(stat.Gid)
		}

		bf := BackedUpFile{
			Original: absPath,
			Backup:   backupPath,
			IsDir:    info.IsDir(),
			Mode:     info.Mode(),
			UID:      uid,
			GID:      gid,
		}

		if info.IsDir() {
			if err := copyDir(absPath, backupPath); err != nil {
				continue
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(absPath)
			if err != nil {
				continue
			}
			if err := os.Symlink(link, backupPath); err != nil {
				continue
			}
		} else {
			if err := copyFile(absPath, backupPath); err != nil {
				continue
			}
		}

		backed = append(backed, bf)
	}

	if len(backed) == 0 {
		os.RemoveAll(trashDir)
		return "", nil, fmt.Errorf("no files were backed up")
	}

	manifest := Manifest{
		ID:    id,
		Files: backed,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return trashDir, backed, nil
	}
	_ = os.WriteFile(filepath.Join(trashDir, "manifest.json"), manifestData, 0o644)

	return trashDir, backed, nil
}

func ReadManifest(trashDir string) (Manifest, error) {
	manifestPath := filepath.Join(trashDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parsing manifest: %w", err)
	}
	return manifest, nil
}

// Restore restores files from a trash directory to their original locations.
func Restore(trashDir string) ([]string, error) {
	result, err := RestoreWithOptions(trashDir, RestoreOptions{Overwrite: true})
	if err != nil {
		return nil, err
	}
	restored := make([]string, len(result))
	for i, file := range result {
		restored[i] = file.Path
	}
	return restored, nil
}

func RestoreWithOptions(trashDir string, opts RestoreOptions) ([]RestoredFile, error) {
	manifest, err := ReadManifest(trashDir)
	if err != nil {
		return nil, err
	}

	var restored []RestoredFile
	for _, bf := range manifest.Files {
		target := restoreTarget(bf.Original, opts.ToDir)
		parentDir := filepath.Dir(target)
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return restored, fmt.Errorf("creating parent dir %s: %w", parentDir, err)
		}

		currentBackup := ""
		if _, err := os.Lstat(target); err == nil {
			switch {
			case opts.BackupCurrent:
				currentBackup, err = moveCurrentAside(target)
				if err != nil {
					return restored, fmt.Errorf("backing up current %s: %w", target, err)
				}
			case opts.Overwrite:
				if err := removeTarget(target, bf.IsDir); err != nil {
					return restored, fmt.Errorf("removing existing %s: %w", target, err)
				}
			default:
				return restored, &ConflictError{Path: target}
			}
		} else if !os.IsNotExist(err) {
			return restored, fmt.Errorf("checking %s: %w", target, err)
		}

		if bf.IsDir {
			if err := copyDir(bf.Backup, target); err != nil {
				return restored, fmt.Errorf("restoring dir %s: %w", target, err)
			}
		} else if bf.Mode&os.ModeSymlink != 0 {
			link, err := os.Readlink(bf.Backup)
			if err != nil {
				return restored, fmt.Errorf("reading symlink %s: %w", bf.Backup, err)
			}
			if err := os.Symlink(link, target); err != nil {
				return restored, fmt.Errorf("restoring symlink %s: %w", target, err)
			}
		} else {
			if err := copyFile(bf.Backup, target); err != nil {
				return restored, fmt.Errorf("restoring file %s: %w", target, err)
			}
		}

		if bf.Mode&os.ModeSymlink == 0 {
			_ = os.Chmod(target, bf.Mode)
		}
		restored = append(restored, RestoredFile{Path: target, BackupCurrent: currentBackup})
	}

	return restored, nil
}

func restoreTarget(original, toDir string) string {
	if toDir == "" {
		return original
	}
	base, err := filepath.Abs(toDir)
	if err != nil {
		base = toDir
	}
	rel := strings.TrimPrefix(filepath.Clean(original), string(os.PathSeparator))
	return filepath.Join(base, rel)
}

func removeTarget(path string, isDir bool) error {
	if isDir {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

func moveCurrentAside(path string) (string, error) {
	stamp := time.Now().Format("20060102-150405")
	base := path + ".oops-current-" + stamp
	target := base
	for i := 1; ; i++ {
		if _, err := os.Lstat(target); err != nil {
			if os.IsNotExist(err) {
				return target, os.Rename(path, target)
			}
			return "", err
		}
		target = fmt.Sprintf("%s-%d", base, i)
	}
}

// Size returns the total size of a trash directory in bytes.
func Size(trashDir string) int64 {
	var total int64
	filepath.Walk(trashDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// TotalSize returns the total size of all trash in bytes.
func TotalSize() int64 {
	return Size(config.TrashDir())
}

// ListTrashDirs returns all trash entry directories sorted by name (newest first).
func ListTrashDirs() ([]string, error) {
	entries, err := os.ReadDir(config.TrashDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(config.TrashDir(), e.Name()))
		}
	}

	for i, j := 0, len(dirs)-1; i < j; i, j = i+1, j-1 {
		dirs[i], dirs[j] = dirs[j], dirs[i]
	}

	return dirs, nil
}

// Remove deletes a trash directory.
func Remove(trashDir string) error {
	root, err := filepath.Abs(config.TrashDir())
	if err != nil {
		return err
	}
	target, err := filepath.Abs(trashDir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("refusing to remove path outside trash: %s", trashDir)
	}
	return os.RemoveAll(target)
}

func copyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	info, err := sf.Stat()
	if err != nil {
		return err
	}

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer df.Close()

	_, err = io.Copy(df, sf)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}

		return copyFile(path, target)
	})
}
