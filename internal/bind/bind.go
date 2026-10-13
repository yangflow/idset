// Package bind changes which folders an identity applies to: it binds a folder, unbinds it, moves it to another
// identity and takes a change back, and it finds the identities that cover one folder at once. The planning
// functions are pure; Runner writes a plan with one store write, so the configuration is regenerated once, and
// looks at the folder again.
package bind

import (
	"errors"
	"slices"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/gitdir"
	"github.com/yangflow/idset/internal/paths"
)

// ErrNotFound means an identity of the request is not in the list.
var ErrNotFound = errors.New("account not found")

// Plan is a change to the list. Changes are the identities as they should be, Before the same identities as
// they are. A plan without Changes does nothing; CoveredBy then says that the identity CoveredID already has a
// path that covers the folder.
type Plan struct {
	Changes, Before      []account.Account
	CoveredID, CoveredBy string
}

// Empty reports whether the plan changes nothing.
func (p Plan) Empty() bool { return len(p.Changes) == 0 }

// Hit is an identity whose gitdir path covers a folder, with the path that does.
type Hit struct{ ID, Path string }

// Overlap lists the identities whose gitdir paths cover the work tree at root, in the order git reads them. The
// last one is what git uses. An identity with several paths covering the folder appears once, with the one that
// is read last. A global identity covers nothing by path and is never listed.
func Overlap(list []account.Account, root string, env paths.Env) []Hit {
	matching := gitdir.Matching(gitdir.Blocks(list, env), root, env)
	var hits []Hit
	for i := len(matching) - 1; i >= 0; i-- {
		id := list[matching[i].Index].ID
		if !slices.ContainsFunc(hits, func(h Hit) bool { return h.ID == id }) {
			hits = append(hits, Hit{ID: id, Path: matching[i].Path})
		}
	}
	slices.Reverse(hits)
	return hits
}

// HasExact reports whether the identity lists the work tree at root itself, so that it can be unbound.
func HasExact(a account.Account, root string, env paths.Env) bool {
	return slices.ContainsFunc(a.Scope.Paths, func(p string) bool { return gitdir.Exact(root, p, env) })
}

func find(list []account.Account, id string) (int, error) {
	i := slices.IndexFunc(list, func(a account.Account) bool { return a.ID == id })
	if i < 0 {
		return 0, ErrNotFound
	}
	return i, nil
}

// used marks a change as a use of the identity, which moves it up the panel's list.
func used(a account.Account, now time.Time) account.Account {
	a.LastUsedAt = &now
	return a
}

// withPaths returns a with the given paths, as a copy that shares nothing with a.
func withPaths(a account.Account, p []string) account.Account {
	a.Scope = account.Scope{Paths: slices.Clone(p)}
	if p == nil {
		a.Scope.Paths = []string{}
	}
	return a
}

// bound is the identity a with the folder bound: appended to its paths, or, for a global identity, its only path.
func bound(a account.Account, root string, env paths.Env) account.Account {
	stored := gitdir.Normalize(root, env)
	if a.Scope.Global {
		return withPaths(a, []string{stored})
	}
	return withPaths(a, append(slices.Clone(a.Scope.Paths), stored))
}

// without is the identity a without the paths for which drop is true.
func without(a account.Account, drop func(string) bool) account.Account {
	return withPaths(a, slices.DeleteFunc(slices.Clone(a.Scope.Paths), drop))
}

// Bind adds the folder at root to the identity's paths. A path that covers the folder already makes it a plan that
// changes nothing. A global identity becomes one that applies to this folder only.
func Bind(list []account.Account, id, root string, env paths.Env, now time.Time) (Plan, error) {
	i, err := find(list, id)
	if err != nil {
		return Plan{}, err
	}
	a := list[i]
	if !a.Scope.Global {
		for _, p := range a.Scope.Paths {
			if gitdir.Matches(root, p, env) {
				return Plan{CoveredID: id, CoveredBy: gitdir.Normalize(p, env)}, nil
			}
		}
	}
	return Plan{Changes: []account.Account{used(bound(a, root, env), now)}, Before: []account.Account{a}}, nil
}

// Unbind removes the paths that are the folder at root itself from the identity; a parent folder's path stays.
// An identity without such a path gives a plan that changes nothing.
func Unbind(list []account.Account, id, root string, env paths.Env, now time.Time) (Plan, error) {
	i, err := find(list, id)
	if err != nil {
		return Plan{}, err
	}
	a := list[i]
	if !HasExact(a, root, env) {
		return Plan{}, nil
	}
	next := without(a, func(p string) bool { return gitdir.Exact(root, p, env) })
	return Plan{Changes: []account.Account{used(next, now)}, Before: []account.Account{a}}, nil
}

// Rebind unbinds the folder from one identity and binds it to another, as one plan. When the target covers the
// folder already nothing changes at all.
func Rebind(list []account.Account, from, to, root string, env paths.Env, now time.Time) (Plan, error) {
	bind, err := Bind(list, to, root, env, now)
	if err != nil || bind.Empty() {
		return bind, err
	}
	unbind, err := Unbind(list, from, root, env, now)
	if err != nil {
		return Plan{}, err
	}
	// The identity left only gets a different path list: it is not used.
	for i := range unbind.Changes {
		unbind.Changes[i].LastUsedAt = unbind.Before[i].LastUsedAt
	}
	return Plan{Changes: append(unbind.Changes, bind.Changes...), Before: append(unbind.Before, bind.Before...)}, nil
}

// UseOnly adds the folder's own path to the identity git uses there, so that the choice no longer rests on the
// order of broader paths. An identity that has the path already gives a plan that changes nothing.
func UseOnly(list []account.Account, winner, root string, env paths.Env, now time.Time) (Plan, error) {
	i, err := find(list, winner)
	if err != nil {
		return Plan{}, err
	}
	a := list[i]
	if a.Scope.Global || HasExact(a, root, env) {
		return Plan{}, nil
	}
	return Plan{Changes: []account.Account{used(bound(a, root, env), now)}, Before: []account.Account{a}}, nil
}

// RemovePath removes one of the identity's paths, in stored or normalized form.
func RemovePath(list []account.Account, id, path string, env paths.Env, now time.Time) (Plan, error) {
	i, err := find(list, id)
	if err != nil {
		return Plan{}, err
	}
	a := list[i]
	target := gitdir.Normalize(path, env)
	drop := func(p string) bool { return target != "" && env.Equal(gitdir.Normalize(p, env), target) }
	if !slices.ContainsFunc(a.Scope.Paths, drop) {
		return Plan{}, nil
	}
	return Plan{Changes: []account.Account{used(without(a, drop), now)}, Before: []account.Account{a}}, nil
}
