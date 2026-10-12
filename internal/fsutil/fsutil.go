// Package fsutil writes files atomically and refuses to work through symbolic links where that matters.
package fsutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrSymlink is returned when a path that must be a regular file is a symbolic link.
var ErrSymlink = errors.New("refusing to operate on a symbolic link")

// WriteAtomic writes data to a temporary file in the same directory and renames it over path, creating the
// directory with dirPerm when it is missing. A new file gets perm; an existing one keeps its mode.
func WriteAtomic(path string, data []byte, perm, dirPerm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, perm); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// RefuseSymlink returns ErrSymlink when path exists and is a symbolic link.
func RefuseSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%s: %w", path, ErrSymlink)
	}
	return nil
}

// Resolve follows symbolic links to the file they point at, so that writing to a linked dotfile changes the real
// file instead of replacing the link. A missing path is returned unchanged.
func Resolve(path string) (string, error) {
	real, err := filepath.EvalSymlinks(path)
	if errors.Is(err, fs.ErrNotExist) {
		return path, nil
	}
	return real, err
}
