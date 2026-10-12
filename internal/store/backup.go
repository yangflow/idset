package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/yangflow/idset/internal/fsutil"
)

// KeepBackups is how many backups of one file are kept; older ones are deleted after each new backup.
const KeepBackups = 10

const stampLayout = "20060102-150405"

// Backup is one snapshot file.
type Backup struct {
	Path string
	Time time.Time
	seq  int
}

// Snapshot copies src into dir as <name>.<UTC time>, adding -1, -2… when that name is taken, then deletes all
// but the newest KeepBackups backups of src.
func Snapshot(src, dir string, now time.Time) (string, error) {
	if err := fsutil.RefuseSymlink(src); err != nil {
		return "", err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	base := filepath.Base(src) + "." + now.UTC().Format(stampLayout)
	var path string
	for n := 0; ; n++ {
		path = filepath.Join(dir, base)
		if n > 0 {
			path += "-" + strconv.Itoa(n)
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, werr := f.Write(data)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			os.Remove(path)
			return "", werr
		}
		break
	}
	list, err := List(dir, filepath.Base(src))
	if err == nil {
		for _, old := range list[min(len(list), KeepBackups):] {
			os.Remove(old.Path)
		}
	}
	return path, nil
}

// List returns the backups of the file named name, newest first; at the same second, -2 before -1 before none.
func List(dir, name string) ([]Backup, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Backup
	for _, e := range entries {
		rest, ok := strings.CutPrefix(e.Name(), name+".")
		if !ok || e.IsDir() {
			continue
		}
		stamp, suffix, _ := strings.Cut(rest, "-")
		if len(stamp) != 8 {
			continue
		}
		clock, n, hasN := strings.Cut(suffix, "-")
		seq := 0
		if hasN {
			if seq, err = strconv.Atoi(n); err != nil || seq < 1 {
				continue
			}
		}
		t, err := time.Parse(stampLayout, stamp+"-"+clock)
		if err != nil {
			continue
		}
		out = append(out, Backup{Path: filepath.Join(dir, e.Name()), Time: t, seq: seq})
	}
	slices.SortFunc(out, func(a, b Backup) int {
		if c := b.Time.Compare(a.Time); c != 0 {
			return c
		}
		return b.seq - a.seq
	})
	return out, nil
}

// restoreFile replaces target with the content of backup atomically. Neither may be a symbolic link.
func restoreFile(backup, target string) error {
	if err := fsutil.RefuseSymlink(backup); err != nil {
		return err
	}
	if err := fsutil.RefuseSymlink(target); err != nil {
		return err
	}
	data, err := os.ReadFile(backup)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}
	return fsutil.WriteAtomic(target, data, 0o600, 0o700)
}

// DeleteBackup removes one backup file, refusing a symbolic link.
func DeleteBackup(path string) error {
	if err := fsutil.RefuseSymlink(path); err != nil {
		return err
	}
	return os.Remove(path)
}
