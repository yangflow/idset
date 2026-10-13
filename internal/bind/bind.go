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

// hasExact reports whether the identity lists the work tree at root itself, so that it can be unbound.
func hasExact(a account.Account, root string, env paths.Env) bool {
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
	if !hasExact(a, root, env) {
		return Plan{}, nil
	}
	next := without(a, func(p string) bool { return gitdir.Exact(root, p, env) })
	return Plan{Changes: []account.Account{used(next, now)}, Before: []account.Account{a}}, nil
}

// ErrNotEffective means a plan would not make the target the identity git uses; it guards an invariant.
var ErrNotEffective = errors.New("the target would not be the identity git uses")

// Rebind makes the target the identity git uses for the folder: every other identity loses the folder's own path,
// and the target gets it unless it has it already, a global one becoming limited to this folder. The target
// covering the folder through a parent path is not enough, since a more specific path of another identity would
// still win. It is one plan.
func Rebind(list []account.Account, from, to, root string, env paths.Env, now time.Time) (Plan, error) {
	if _, err := find(list, from); err != nil {
		return Plan{}, err
	}
	ti, err := find(list, to)
	if err != nil {
		return Plan{}, err
	}
	var plan Plan
	next := slices.Clone(list)
	for i, a := range list {
		if i != ti && hasExact(a, root, env) {
			next[i] = without(a, func(p string) bool { return gitdir.Exact(root, p, env) })
			plan.Changes, plan.Before = append(plan.Changes, next[i]), append(plan.Before, a)
		}
	}
	if t := list[ti]; t.Scope.Global || !hasExact(t, root, env) {
		next[ti] = used(bound(t, root, env), now)
		plan.Changes, plan.Before = append(plan.Changes, next[ti]), append(plan.Before, t)
	}
	matching := gitdir.Matching(gitdir.Blocks(next, env), root, env)
	if len(matching) == 0 || next[matching[len(matching)-1].Index].ID != to {
		return Plan{}, ErrNotEffective
	}
	return plan, nil
}

// UseOnly adds the folder's own path to the identity git uses there, so that the choice no longer rests on the
// order of broader paths. An identity that has the path already gives a plan that changes nothing.
func UseOnly(list []account.Account, winner, root string, env paths.Env, now time.Time) (Plan, error) {
	i, err := find(list, winner)
	if err != nil {
		return Plan{}, err
	}
	a := list[i]
	if a.Scope.Global || hasExact(a, root, env) {
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

// FixPath replaces the identity's stale path with the folder at root, where the stale path stands, so that the
// other paths and their order stay. When the identity lists the folder already, the stale path is only removed.
// An identity without the stale path gives a plan that changes nothing.
func FixPath(list []account.Account, id, stale, root string, env paths.Env, now time.Time) (Plan, error) {
	i, err := find(list, id)
	if err != nil {
		return Plan{}, err
	}
	a := list[i]
	old, fresh := gitdir.Normalize(stale, env), gitdir.Normalize(root, env)
	same := func(p, target string) bool { return target != "" && env.Equal(gitdir.Normalize(p, env), target) }
	if !slices.ContainsFunc(a.Scope.Paths, func(p string) bool { return same(p, old) }) {
		return Plan{}, nil
	}
	have := slices.ContainsFunc(a.Scope.Paths, func(p string) bool { return same(p, fresh) })
	var next []string
	for _, p := range a.Scope.Paths {
		switch {
		case !same(p, old):
			next = append(next, p)
		case !have:
			next, have = append(next, fresh), true
		}
	}
	return Plan{Changes: []account.Account{used(withPaths(a, next), now)}, Before: []account.Account{a}}, nil
}
