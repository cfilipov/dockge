package docker

import (
	"io/fs"
	"os"
	"path/filepath"
)

// ClearDirContents removes all entries inside dir without removing dir itself.
// This preserves the directory inode so fsnotify watchers remain valid.
func ClearDirContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// samePath returns true if a and b resolve to the same filesystem path.
func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	// Try to resolve symlinks; fall back to cleaned absolute paths.
	realA, err := filepath.EvalSymlinks(absA)
	if err != nil {
		realA = absA
	}
	realB, err := filepath.EvalSymlinks(absB)
	if err != nil {
		realB = absB
	}
	return realA == realB
}

// CopyDirRecursive copies all files from src to dst recursively.
func CopyDirRecursive(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
