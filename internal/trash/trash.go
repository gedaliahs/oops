package trash

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

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

// Backup backs up the given files into a new trash directory.
// Uses hard links when possible (instant, no extra disk space).
// Falls back to copying when hard links aren't supported (cross-filesystem).
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
			hardLinked, err := linkDir(absPath, backupPath)
			if err != nil {
				// Fall back to full copy
				if err := copyDir(absPath, backupPath); err != nil {
					continue
				}
			} else {
				bf.HardLink = hardLinked
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
			// Try hard link first (instant, no disk cost)
			if err := os.Link(absPath, backupPath); err != nil {
				// Fall back to copy (cross-filesystem)
				if err := copyFile(absPath, backupPath); err != nil {
					continue
				}
			} else {
				bf.HardLink = true
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

// Restore restores files from a trash directory to their original locations.
func Restore(trashDir string) ([]string, error) {
	manifestPath := filepath.Join(trashDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	var restored []string

	for _, bf := range manifest.Files {
		parentDir := filepath.Dir(bf.Original)
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return restored, fmt.Errorf("creating parent dir %s: %w", parentDir, err)
		}

		if bf.IsDir {
			os.RemoveAll(bf.Original)
			// Try hard link restore first, fall back to copy
			if _, err := linkDir(bf.Backup, bf.Original); err != nil {
				if err := copyDir(bf.Backup, bf.Original); err != nil {
					return restored, fmt.Errorf("restoring dir %s: %w", bf.Original, err)
				}
			}
		} else {
			// Try hard link first, fall back to copy
			os.Remove(bf.Original) // remove if exists
			if err := os.Link(bf.Backup, bf.Original); err != nil {
				if err := copyFile(bf.Backup, bf.Original); err != nil {
					return restored, fmt.Errorf("restoring file %s: %w", bf.Original, err)
				}
			}
		}

		os.Chmod(bf.Original, bf.Mode)
		restored = append(restored, bf.Original)
	}

	return restored, nil
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
	if !strings.HasPrefix(trashDir, config.TrashDir()) {
		return fmt.Errorf("refusing to remove path outside trash: %s", trashDir)
	}
	return os.RemoveAll(trashDir)
}

// linkDir hard-links all files in a directory tree.
// Returns true if all files were hard-linked, false if any fell back to copy.
func linkDir(src, dst string) (bool, error) {
	allLinked := true
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
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

		// Try hard link
		if err := os.Link(path, target); err != nil {
			// Fall back to copy
			allLinked = false
			return copyFile(path, target)
		}
		return nil
	})
	return allLinked, err
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
