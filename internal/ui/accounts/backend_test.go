package accounts

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/apply"
	"github.com/yangflow/idset/internal/keyfile"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
)

type disk struct {
	t      *testing.T
	env    paths.Env
	target apply.Target
	b      Backend
}

// newDisk wires a Backend to a store, the generator and a fake `git config` in a temporary home.
func newDisk(t *testing.T, gitConfig map[string]string) *disk {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env := paths.Env{Home: home, Windows: runtime.GOOS == "windows"}
	target := apply.DefaultTarget(env)
	now := func() time.Time { return time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC) }
	s := store.Open(target.Dir, now, store.WithRegenerate(apply.Regenerate(env, target, now)))
	git := func(_ context.Context, key string) (string, error) { return gitConfig[key], nil }
	return &disk{t: t, env: env, target: target, b: Backend{Store: s, Env: env, Git: git}}
}

func (d *disk) read(path string) string {
	d.t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func (d *disk) exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func (d *disk) key(name string) string {
	d.t.Helper()
	dir := filepath.Join(d.env.Home, ".ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		d.t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	for _, p := range []string{path, path + ".pub"} {
		if err := os.WriteFile(p, []byte("placeholder, not a key"), 0o600); err != nil {
			d.t.Fatal(err)
		}
	}
	return path
}

func (d *disk) identity(label, alias, keyPath string, folders ...string) account.Account {
	a := account.New(time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC))
	a.Label, a.SSHAlias, a.KeyPath, a.GitUserName, a.GitUserEmail = label, alias, keyPath, "Ada", label+"@example.com"
	a.Scope = account.Scope{Paths: folders}
	return a
}

func (d *disk) subFile(id string) string {
	return filepath.Join(d.target.Dir, "gitconfig-"+id+".managed")
}

func TestSaveWritesTheManagedFilesAndTheIncludeBlocks(t *testing.T) {
	d := newDisk(t, nil)
	work := d.identity("work", "github-work", "~/.ssh/id_work", "~/work/")
	if err := d.b.Save(work, true); err != nil {
		t.Fatal(err)
	}
	ssh := d.read(filepath.Join(d.target.Dir, "ssh_config.managed"))
	if !strings.Contains(ssh, "Host github-work") || !strings.Contains(ssh, `IdentityFile "~/.ssh/id_work"`) {
		t.Errorf("ssh_config.managed:\n%s", ssh)
	}
	main := d.read(filepath.Join(d.target.Dir, "gitconfig.managed"))
	if !strings.Contains(main, `[includeIf "gitdir`) || !strings.Contains(main, "~/work/") {
		t.Errorf("gitconfig.managed:\n%s", main)
	}
	if sub := d.read(d.subFile(work.ID)); !strings.Contains(sub, "email = work@example.com") {
		t.Errorf("per-identity file:\n%s", sub)
	}
	for _, user := range []string{d.target.SSHConfig, d.target.GitConfig} {
		text := d.read(user)
		if strings.Count(text, "# --- idset managed ---") != 1 || !strings.Contains(text, "# --- idset managed end ---") {
			t.Errorf("%s needs the marker block once:\n%s", user, text)
		}
	}

	work.SSHAlias = "github-job"
	if err := d.b.Save(work, false); err != nil {
		t.Fatal(err)
	}
	ssh = d.read(filepath.Join(d.target.Dir, "ssh_config.managed"))
	if strings.Contains(ssh, "github-work") || !strings.Contains(ssh, "Host github-job") {
		t.Errorf("an edit must be regenerated:\n%s", ssh)
	}
	if d.b.Store.Accounts()[0].UpdatedAt.IsZero() {
		t.Error("the store stamps the update")
	}
}

func TestSaveRefusesWhatTheStoreRefuses(t *testing.T) {
	d := newDisk(t, nil)
	first := d.identity("a", "same", "")
	if err := d.b.Save(first, true); err != nil {
		t.Fatal(err)
	}
	second := d.identity("b", "same", "")
	var problems account.Problems
	if err := d.b.Save(second, true); !errors.As(err, &problems) {
		t.Fatalf("a duplicate alias: %v", err)
	}
	if err := d.b.Save(first, true); !errors.Is(err, store.ErrDuplicate) {
		t.Fatalf("adding twice: %v", err)
	}
}

func TestDeleteRemovesTheIdentityThenOnlyThePrivateKey(t *testing.T) {
	d := newDisk(t, nil)
	key := d.key("id_work")
	work := d.identity("work", "github-work", "~/.ssh/id_work", "~/work/")
	other := d.identity("other", "github-other", "~/.ssh/id_other", "~/other/")
	for _, a := range []account.Account{work, other} {
		if err := d.b.Save(a, true); err != nil {
			t.Fatal(err)
		}
	}
	if reason, _ := d.b.InspectKey(work.ID); reason != keyfile.None {
		t.Fatalf("inspect: %d", reason)
	}
	res := d.b.Delete(work.ID, true)
	if res.Err != nil || res.KeyErr != nil {
		t.Fatalf("%+v", res)
	}
	if d.exists(key) || !d.exists(key+".pub") {
		t.Error("the private key goes, the .pub file stays")
	}
	if ssh := d.read(filepath.Join(d.target.Dir, "ssh_config.managed")); strings.Contains(ssh, "github-work") || !strings.Contains(ssh, "github-other") {
		t.Errorf("the generated configuration follows:\n%s", ssh)
	}
	if d.exists(d.subFile(work.ID)) || !d.exists(d.subFile(other.ID)) {
		t.Error("only the deleted identity's file goes")
	}
	if !strings.Contains(d.read(d.target.GitConfig), "# --- idset managed ---") {
		t.Error("the marker block stays")
	}
	if got := d.b.Store.Accounts(); len(got) != 1 || got[0].ID != other.ID {
		t.Errorf("accounts: %+v", got)
	}
	// Deleting without the box leaves the key.
	otherKey := d.key("id_other")
	if res := d.b.Delete(other.ID, false); res.Err != nil || res.KeyErr != nil || !d.exists(otherKey) {
		t.Errorf("without the box: %+v", res)
	}
	if res := d.b.Delete(other.ID, false); !errors.Is(res.Err, store.ErrNotFound) {
		t.Errorf("deleting twice: %+v", res)
	}
}

func TestDeleteKeepsAKeyAnotherIdentityUses(t *testing.T) {
	d := newDisk(t, nil)
	key := d.key("id_shared")
	a := d.identity("a", "alias-a", "~/.ssh/id_shared")
	b := d.identity("b", "alias-b", "~/.ssh/id_shared")
	for _, x := range []account.Account{a, b} {
		if err := d.b.Save(x, true); err != nil {
			t.Fatal(err)
		}
	}
	if reason, other := d.b.InspectKey(a.ID); reason != keyfile.Shared || other != "b" {
		t.Fatalf("inspect: %d %q", reason, other)
	}
	// Asked to delete the key anyway (the box was checked before another identity took the key): refused.
	res := d.b.Delete(a.ID, true)
	var refused *keyfile.RefusedError
	if res.Err != nil || !errors.As(res.KeyErr, &refused) || refused.Reason != keyfile.Shared {
		t.Fatalf("%+v", res)
	}
	if !d.exists(key) {
		t.Error("the shared key must stay")
	}
	if len(d.b.Store.Accounts()) != 1 {
		t.Error("the deletion of the identity is not undone")
	}
}

func TestKeyRemovalFailureDoesNotRollBack(t *testing.T) {
	d := newDisk(t, nil)
	dir := filepath.Join(d.env.Home, ".ssh", "id_dir")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	a := d.identity("a", "alias-a", "~/.ssh/id_dir")
	if err := d.b.Save(a, true); err != nil {
		t.Fatal(err)
	}
	if reason, _ := d.b.InspectKey(a.ID); reason != keyfile.NotFile {
		t.Fatalf("inspect: %d", reason)
	}
	res := d.b.Delete(a.ID, true)
	if res.Err != nil || res.KeyErr == nil {
		t.Fatalf("%+v", res)
	}
	if len(d.b.Store.Accounts()) != 0 || !d.exists(dir) {
		t.Error("the identity is deleted and the directory untouched")
	}
	if strings.Contains(d.read(filepath.Join(d.target.Dir, "ssh_config.managed")), "alias-a") {
		t.Error("the configuration was regenerated before the key was looked at")
	}
}

func TestDeleteReportsAConfigurationThatCouldNotBeRewritten(t *testing.T) {
	d := newDisk(t, nil)
	key := d.key("id_a")
	a := d.identity("a", "alias-a", "~/.ssh/id_a")
	if err := d.b.Save(a, true); err != nil {
		t.Fatal(err)
	}
	// The list is written, then the regeneration fails.
	fail := store.Open(d.target.Dir, time.Now, store.WithRegenerate(func([]account.Account) error { return errors.New("disk full") }))
	b := Backend{Store: fail, Env: d.env}
	res := b.Delete(a.ID, true)
	var regenerate *store.RegenerateError
	if !errors.As(res.Err, &regenerate) {
		t.Fatalf("%+v", res)
	}
	if res.KeyErr != nil || d.exists(key) {
		t.Errorf("the identity is deleted, so its key goes too: %+v", res)
	}
}

func TestPrefillReadsGitConfigAndToleratesGaps(t *testing.T) {
	d := newDisk(t, map[string]string{"user.name": "Ada", "user.email": "ada@example.com"})
	if name, email := d.b.Prefill(context.Background()); name != "Ada" || email != "ada@example.com" {
		t.Errorf("%q %q", name, email)
	}
	d = newDisk(t, nil)
	if name, email := d.b.Prefill(context.Background()); name != "" || email != "" {
		t.Errorf("%q %q", name, email)
	}
	failing := Backend{Git: func(context.Context, string) (string, error) { return "", errors.New("git: not found") }}
	if name, email := failing.Prefill(context.Background()); name != "" || email != "" {
		t.Errorf("%q %q", name, email)
	}
}

func TestSystemGitConfigReadsAndTimesOut(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	git := SystemGitConfig(10 * time.Second)
	if _, err := os.Stat(filepath.Join(home, "gitconfig")); err == nil {
		t.Fatal("the file must not exist yet")
	}
	if got, err := git(context.Background(), "user.name"); err != nil || got != "" {
		t.Skipf("git is not usable here: %q %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(home, "gitconfig"), []byte("[user]\n\tname = Ada\n\temail = ada@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := git(context.Background(), "user.email"); err != nil || got != "ada@example.com" {
		t.Errorf("%q %v", got, err)
	}
	if got, err := git(context.Background(), "user.signingkey"); err != nil || got != "" {
		t.Errorf("a key that is not set is empty: %q %v", got, err)
	}
}
