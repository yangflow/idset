package accounts

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/ui/theme"
)

var env = paths.Env{Home: "/home/ada"}

var base = time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

type saveCall struct {
	a     account.Account
	isNew bool
}

type deleteCall struct {
	id      string
	withKey bool
}

type folderCall struct {
	id    string
	index int
}

type recorder struct {
	saves     []saveCall
	deletes   []deleteCall
	inspected []string
	prefilled []string
	keys      []string
	folders   []folderCall
	imports   int
}

func (r *recorder) actions() Actions {
	return Actions{
		Save:         func(a account.Account, isNew bool) { r.saves = append(r.saves, saveCall{a, isNew}) },
		Delete:       func(id string, withKey bool) { r.deletes = append(r.deletes, deleteCall{id, withKey}) },
		InspectKey:   func(id string) { r.inspected = append(r.inspected, id) },
		Prefill:      func(id string) { r.prefilled = append(r.prefilled, id) },
		ChooseKey:    func(id string) { r.keys = append(r.keys, id) },
		ChooseFolder: func(id string, i int) { r.folders = append(r.folders, folderCall{id, i}) },
		OpenImport:   func() { r.imports++ },
	}
}

// sample is the list of the mockup: a scoped work identity first in storage order, a global personal one, a
// custom client identity and an unnamed one.
func sample() []account.Account {
	used := func(day int) *time.Time {
		at := time.Date(2026, 10, day, 9, 0, 0, 0, time.UTC)
		return &at
	}
	work := account.Account{
		ID: "11111111-1111-1111-1111-111111111111", Label: "work", Color: "blue", Provider: account.GitHub, Username: "bob-work",
		GitUserName: "Bob", GitUserEmail: "bob@example.com", SSHAlias: "github-work", KeyPath: "~/.ssh/id_ed25519_work",
		KeyFingerprint: "SHA256:AbCdEf", SSHPort: 443, Scope: account.Scope{Paths: []string{"~/work/", "~/src/new/", "~/clients/acme/"}},
		URLRewrites: []account.Rewrite{{From: "https://github.com/", To: "git@github-work:"},
			{From: "https://github.com/acme/", To: "git@github-work:acme/", Push: true}},
		Notes: "Employer repositories.", CreatedAt: base, UpdatedAt: base.AddDate(0, 1, 0), LastUsedAt: used(12),
	}
	personal := account.Account{
		ID: "22222222-2222-2222-2222-222222222222", Label: "personal", Color: "green", Provider: account.GitHub, Username: "bob-example",
		GitUserName: "Bob", GitUserEmail: "bob@example.net", SSHAlias: "github-personal", KeyPath: "~/.ssh/id_ed25519",
		SSHPort: 443, Scope: account.Scope{Global: true}, URLRewrites: []account.Rewrite{}, CreatedAt: base, UpdatedAt: base, LastUsedAt: used(11),
	}
	client := account.Account{
		ID: "33333333-3333-3333-3333-333333333333", Label: "client", Color: "orange", Provider: account.Custom, HostName: "git.client.example",
		Username: "bob", GitUserName: "Bob Example", GitUserEmail: "bob@client.example", SSHAlias: "client-git", KeyPath: "~/.ssh/id_client",
		SSHPort: 22, Scope: account.Scope{Paths: []string{"~/work/client/"}}, URLRewrites: []account.Rewrite{}, CreatedAt: base, UpdatedAt: base, LastUsedAt: used(2),
	}
	misc := account.Account{
		ID: "44444444-4444-4444-4444-444444444444", Color: "purple", Provider: account.GitLab, GitUserEmail: "nobody@example.com",
		SSHPort: 443, Scope: account.Scope{Paths: []string{"~/misc/"}}, URLRewrites: []account.Rewrite{}, CreatedAt: base, UpdatedAt: base,
	}
	return []account.Account{work, personal, client, misc}
}

const (
	workID     = "11111111-1111-1111-1111-111111111111"
	personalID = "22222222-2222-2222-2222-222222222222"
	clientID   = "33333333-3333-3333-3333-333333333333"
	miscID     = "44444444-4444-4444-4444-444444444444"
)

type rig struct {
	s   *Screen
	tt  *ui.Tester
	rec *recorder
}

// open returns a loaded window of 760×520 with the sample list, with the request applied.
func open(t *testing.T, lang string, list []account.Account, req Request) *rig {
	t.Helper()
	rec := &recorder{}
	s := New(Text(lang), env, rec.actions())
	s.SetAccounts(list, nil, "/home/ada/.config/idset/accounts.json")
	s.Open(req)
	return &rig{s: s, tt: ui.NewTester(withDialog(s), 760, 1500), rec: rec}
}

// withDialog draws the screen and, at the bottom of the frame where the owner's dialog window would be, its dialog.
func withDialog(s *Screen) func(c *ui.Context) {
	return func(c *ui.Context) {
		s.View(c)
		if s.DialogOpen() {
			ui.Column(c).Absolute().Left(0).Bottom(0).Width(theme.DialogWidth).Background(c.Theme().Background).
				Children(func() { s.Dialog(c) })
		}
	}
}
