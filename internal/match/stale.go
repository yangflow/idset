package match

import (
	"cmp"
	"errors"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/gitdir"
	"github.com/yangflow/idset/internal/paths"
)

// Stale is an identity's gitdir path that no longer exists but looks like the folder of a Result before it was
// renamed or moved. ID is the identity and Path the path as the identity stores it.
type Stale struct{ ID, Path string }

// staleCandidate is a stale path with what the choice between them depends on.
type staleCandidate struct {
	Stale
	index   int
	renamed bool
	named   bool
}

// FindStale looks for the gitdir path that the work tree at root has probably replaced, or nil. A candidate is a
// path of any identity that gone reports as missing, and either has the same parent folder as root (a rename) or
// ends in the same folder name, ignoring case (a move). A folder directly under the file system root, and the
// home directory, have no candidates, since neither is a project that was renamed. Of several candidates a rename
// comes before a move, an identity named like the folder before others, then the stored order of identities, then
// the path. gone answers for an absolute path in the platform's form and reads the disk, so this is for background
// work.
func FindStale(list []account.Account, root string, env paths.Env, gone func(path string) bool) *Stale {
	root = canon(root)
	parent, name := pathpkg.Split(root)
	if name == "" || isTop(parent) || isHome(root, env) {
		return nil
	}
	parent = strings.TrimSuffix(parent, "/")
	var found []staleCandidate
	for i, a := range list {
		if a.Scope.Global {
			continue
		}
		for _, stored := range a.Scope.Paths {
			normal := gitdir.Normalize(stored, env)
			if normal == "" {
				continue
			}
			abs := canon(env.Expand(normal))
			oldParent, oldName := pathpkg.Split(abs)
			oldParent = strings.TrimSuffix(oldParent, "/")
			if env.Equal(abs, root) || oldName == "" {
				continue
			}
			renamed := env.Equal(oldParent, parent)
			if !renamed && !strings.EqualFold(oldName, name) {
				continue
			}
			if !gone(env.Native(normal)) {
				continue
			}
			found = append(found, staleCandidate{Stale: Stale{ID: a.ID, Path: stored}, index: i, renamed: renamed,
				named: strings.EqualFold(strings.TrimSpace(a.Label), name)})
		}
	}
	if len(found) == 0 {
		return nil
	}
	best := slices.MinFunc(found, func(a, b staleCandidate) int {
		return cmp.Or(
			-boolCmp(a.renamed, b.renamed),
			-boolCmp(a.named, b.named),
			cmp.Compare(a.index, b.index),
			strings.Compare(a.Path, b.Path),
		)
	})
	return &best.Stale
}

// boolCmp orders true before false when negated: 1 when only a is true, -1 when only b is.
func boolCmp(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	}
	return -1
}

// canon is p in slash form without a trailing slash, with symbolic links resolved as far as the path exists: the
// nearest existing ancestor is resolved and the rest is appended, so that a path that is gone, such as a stale
// one, compares equal to the resolved path git reports (/tmp and /private/tmp, /var and /private/var).
func canon(p string) string {
	p = strings.TrimSuffix(paths.Slash(p), "/")
	if p == "" {
		return p
	}
	rest := ""
	for cur := p; ; {
		if real, err := filepath.EvalSymlinks(filepath.FromSlash(cur)); err == nil {
			return strings.TrimSuffix(paths.Slash(real), "/") + rest
		}
		parent, name := pathpkg.Split(cur)
		if name == "" || parent == "" {
			return p
		}
		rest, cur = "/"+name+rest, strings.TrimSuffix(parent, "/")
		if cur == "" {
			cur = "/"
		}
	}
}

// isTop reports whether dir, in slash form, is the root of the file system or of a drive.
func isTop(dir string) bool {
	if dir == "/" || dir == "" {
		return true
	}
	return len(dir) == 3 && dir[1] == ':' && dir[2] == '/'
}

func isHome(root string, env paths.Env) bool {
	for _, home := range []string{env.Home, env.RealHome} {
		if home = canon(home); home != "" && env.Equal(root, home) {
			return true
		}
	}
	return false
}

// Gone reports whether the file system says there is nothing at path. Any other error, such as a folder that
// cannot be read, does not count as gone.
func Gone(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, fs.ErrNotExist)
}
