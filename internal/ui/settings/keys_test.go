package settings

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/keygen"
)

const (
	samplePub = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVvBRLiy1IB3trYVbYfC8KOIqOu/SBSTpKehjxdeEqt ada@example.com"
	sampleFp  = "SHA256:GaZxSsN21z2W8DOaH9fsWhWZ2d8xBcx1tRPgPmuE4qc"
)

// keyIdentities are a work identity with a key and a personal one whose key is missing.
func keyIdentities() []account.Account {
	work := identity("work", "blue", false)
	work.Provider, work.KeyPath = account.GitLab, "~/.ssh/id_work"
	own := identity("personal", "green", true)
	own.Provider, own.KeyPath = account.GitHub, "~/.ssh/id_personal"
	return []account.Account{work, own}
}

// keysOpen opens the Keys pane with the identities given before the request, as the owner does.
func keysOpen(t *testing.T, lang string, list []account.Account, req Request) *rig {
	t.Helper()
	rec := &recorder{}
	s := New(Text(lang), env, rec.actions())
	s.now = func() time.Time { return base }
	s.SetSystem("en")
	s.SetAccounts(list)
	req.Pane = Keys
	s.Open(req)
	r := &rig{s: s, tt: ui.NewTester(s.View, 720, 1400), rec: rec}
	r.tt.Frame()
	return r
}

// generated generates with the form as it is and reports a key at ~/.ssh/<name>.
func (r *rig) generated(t *testing.T, pub string) keygen.Key {
	t.Helper()
	r.click(t, "Generate")
	if len(r.rec.generated) == 0 {
		t.Fatalf("nothing generated: %q", r.tt.Texts())
	}
	req := r.rec.generated[len(r.rec.generated)-1]
	key := keygen.Key{Path: "~/.ssh/" + req.Name, Public: pub}
	if pub != "" {
		key.Fingerprint = sampleFp
	}
	r.s.GenerateDone(key, nil)
	r.frame()
	return key
}

func TestKeyFormDefaults(t *testing.T) {
	r := keysOpen(t, "en", keyIdentities(), Request{})
	r.want(t, "New key", "Ed25519", "RSA 4096", "GitHub", "Saved as ~/.ssh/id_idset and ~/.ssh/id_idset.pub. An existing file is never overwritten.")
	if r.s.keys.name != "id_idset" || r.s.keys.comment != "work@example.com" {
		t.Errorf("form = %q, %q", r.s.keys.name, r.s.keys.comment)
	}
	// Without identities the comment fills in once they arrive, and only while it is empty.
	r = keysOpen(t, "en", nil, Request{})
	r.s.SetAccounts(keyIdentities())
	if r.s.keys.comment != "work@example.com" {
		t.Errorf("comment after the list = %q", r.s.keys.comment)
	}
	r.s.keys.comment = ""
	r.s.SetAccounts(keyIdentities())
	if r.s.keys.comment != "" {
		t.Errorf("a cleared comment stays cleared: %q", r.s.keys.comment)
	}
}

func TestKeyFormSendsTheRequest(t *testing.T) {
	r := keysOpen(t, "en", keyIdentities(), Request{})
	r.s.keys.kind, r.s.keys.platform, r.s.keys.name, r.s.keys.comment = 1, "GitLab", "id_lab", " lab@example.com\n"
	r.frame()
	r.click(t, "Generate")
	want := KeyRequest{Type: keygen.RSA, Name: "id_lab", Comment: "lab@example.com"}
	if !slices.Equal(r.rec.generated, []KeyRequest{want}) {
		t.Fatalf("generated %+v", r.rec.generated)
	}
	r.want(t, "Generating…")
	r.click(t, "Generate")
	if len(r.rec.generated) != 1 {
		t.Errorf("a second Generate while busy: %+v", r.rec.generated)
	}
	r.s.GenerateDone(keygen.Key{Path: "~/.ssh/id_lab", Public: samplePub, Fingerprint: sampleFp}, nil)
	r.frame()
	r.want(t, "Key created", "RSA 4096 · GitLab", samplePub, "~/.ssh/id_lab.pub", sampleFp, "None",
		"This private key has no passphrase. You can add one later with ssh-keygen -p -f ~/.ssh/id_lab.",
		"Add the public key on GitLab", "Open GitLab Settings", "Load into ssh-agent", "Use for identity",
		"On the GitLab settings page, add a new SSH key and paste the copied public key.")
	r.wantNot(t, "Generating…")
}

func TestKeyFormRefusesBadNames(t *testing.T) {
	r := keysOpen(t, "en", nil, Request{})
	r.s.keys.name = "../id"
	r.frame()
	r.want(t, "Use letters, digits, ., _ and - only; do not start with . or include ..")
	r.click(t, "Generate")
	if len(r.rec.generated) != 0 {
		t.Errorf("generated %+v", r.rec.generated)
	}
}

func TestKeyFormErrors(t *testing.T) {
	for err, text := range map[error]string{
		&keygen.ExistsError{Path: "~/.ssh/id_idset"}:       "~/.ssh/id_idset already exists. Idset never overwrites a key; choose another file name.",
		fmt.Errorf("start: %w", exec.ErrNotFound):          "Could not run ssh-keygen: command not found. Install the OpenSSH client and try again.",
		errors.New("Saving key failed: Permission denied"): "Could not generate the key: Saving key failed: Permission denied",
	} {
		r := keysOpen(t, "en", nil, Request{})
		r.click(t, "Generate")
		r.s.GenerateDone(keygen.Key{}, err)
		r.frame()
		r.want(t, text, "New key")
		// The error is about the name it was given; another name brings the plain note back.
		r.s.keys.name = "id_other"
		r.frame()
		r.wantNot(t, text)
		r.want(t, "Saved as ~/.ssh/id_other")
	}
}

func TestKeyResultCopiesAndReveals(t *testing.T) {
	r := keysOpen(t, "en", keyIdentities(), Request{})
	r.s.goos = "darwin"
	r.generated(t, samplePub)
	r.click(t, "Copy Public Key")
	if r.tt.Clipboard() != samplePub {
		t.Errorf("clipboard = %q", r.tt.Clipboard())
	}
	r.want(t, "Copied")
	r.click(t, "Show in Finder")
	if !slices.Equal(r.rec.reveals, []string{"~/.ssh/id_idset"}) {
		t.Errorf("reveals %q", r.rec.reveals)
	}
	r.click(t, "Generate Another")
	r.want(t, "New key")
	if r.s.keys.key != nil || r.s.keys.name != "id_idset" {
		t.Errorf("back to the form: %+v", r.s.keys)
	}
}

func TestKeyResultWithoutPublicKey(t *testing.T) {
	r := keysOpen(t, "en", nil, Request{})
	r.generated(t, "")
	r.want(t, "~/.ssh/id_idset.pub does not exist, so there is no public key to copy.")
	r.wantNot(t, "Fingerprint")
}

func TestKeyResultForACustomPlatform(t *testing.T) {
	r := keysOpen(t, "en", nil, Request{})
	r.s.keys.platform = "Custom"
	r.generated(t, samplePub)
	r.want(t, "Ed25519 · Custom", "Add the public key on your server", "A custom platform has no fixed settings page.")
	r.wantNot(t, "Open GitHub Settings")
}

func TestKeyLoadsIntoTheAgent(t *testing.T) {
	r := keysOpen(t, "en", nil, Request{})
	r.generated(t, samplePub)
	r.click(t, "Load")
	if !slices.Equal(r.rec.loads, []string{"~/.ssh/id_idset"}) {
		t.Fatalf("loads %q", r.rec.loads)
	}
	r.s.LoadDone(nil)
	r.frame()
	r.want(t, "Loaded")
	if _, ok := r.tt.Find("Load"); ok {
		t.Error("Load stays after the key is loaded")
	}

	r = keysOpen(t, "en", nil, Request{})
	r.generated(t, samplePub)
	r.click(t, "Load")
	r.s.LoadDone(keygen.ErrNoAgent)
	r.frame()
	r.want(t, "ssh-agent is not running, so the key was not loaded.")
	r.wantNot(t, "On the GitHub settings page")

	r = keysOpen(t, "en", nil, Request{})
	r.generated(t, samplePub)
	r.click(t, "Load")
	r.s.LoadDone(errors.New("Invalid key"))
	r.frame()
	r.want(t, "Could not load the key: Invalid key", "Load")
}

func TestKeyGoesToAChosenIdentity(t *testing.T) {
	r := keysOpen(t, "en", keyIdentities(), Request{})
	r.generated(t, samplePub)
	r.click(t, "Choose…")
	r.want(t, "Use this key for which identity?", "Now: ~/.ssh/id_work", "New identity",
		"Email work@example.com · GitHub. Finish and save it in the accounts window.")
	if r.s.keys.pick != "work-id" {
		t.Errorf("pick = %q", r.s.keys.pick)
	}
	r.click(t, "personal")
	r.click(t, "Use")
	if !slices.Equal(r.rec.uses, [][3]string{{"personal-id", "~/.ssh/id_idset", sampleFp}}) {
		t.Fatalf("uses %q", r.rec.uses)
	}
	r.wantNot(t, "Use this key for which identity?")
	r.s.UseDone("personal-id", nil)
	r.frame()
	r.want(t, "personal", "Change…", "The key path and fingerprint of personal were written and the configuration regenerated.")

	r.click(t, "Change…")
	if r.s.keys.pick != "personal-id" {
		t.Errorf("pick on change = %q", r.s.keys.pick)
	}
	r.click(t, "Cancel")
	if len(r.rec.uses) != 1 {
		t.Errorf("Cancel used the key: %q", r.rec.uses)
	}

	r.s.UseDone("work-id", errors.New("disk full"))
	r.frame()
	r.want(t, "Could not write the key to the identity: disk full")
}

func TestKeyMakesANewIdentity(t *testing.T) {
	r := keysOpen(t, "en", nil, Request{})
	r.s.keys.platform, r.s.keys.comment = "GitLab", "ada@example.com"
	r.generated(t, samplePub)
	r.click(t, "Choose…")
	if r.s.keys.pick != newIdentity {
		t.Fatalf("pick = %q", r.s.keys.pick)
	}
	r.click(t, "Use")
	want := KeyDraft{Provider: account.GitLab, Email: "ada@example.com", KeyPath: "~/.ssh/id_idset", Fingerprint: sampleFp}
	if !slices.Equal(r.rec.drafts, []KeyDraft{want}) {
		t.Errorf("drafts %+v", r.rec.drafts)
	}
	// A comment that is not an email address gives no email.
	r = keysOpen(t, "en", nil, Request{})
	r.s.keys.comment = "laptop key"
	r.generated(t, samplePub)
	r.click(t, "Choose…")
	r.want(t, "GitHub. Finish and save it in the accounts window.")
	r.click(t, "Use")
	if len(r.rec.drafts) != 1 || r.rec.drafts[0].Email != "" {
		t.Errorf("drafts %+v", r.rec.drafts)
	}
}

func TestKeyFormFilledInFromTheDiagnostics(t *testing.T) {
	r := keysOpen(t, "en", keyIdentities(), Request{For: "work-id"})
	r.want(t, "The private key of work, ~/.ssh/id_work, does not exist.")
	if k := r.s.keys; k.name != "id_work" || k.comment != "work@example.com" || k.platform != "GitLab" || k.target != "work-id" {
		t.Fatalf("form = %+v", k)
	}
	r.generated(t, samplePub)
	if !slices.Equal(r.rec.uses, [][3]string{{"work-id", "~/.ssh/id_work", sampleFp}}) {
		t.Errorf("the key goes to work at once: %q", r.rec.uses)
	}
	r.wantNot(t, "Use this key for which identity?")

	// Under another name the key is not the one work names, so it is not given to it.
	r = keysOpen(t, "en", keyIdentities(), Request{For: "work-id"})
	r.s.keys.name = "id_work2"
	r.generated(t, samplePub)
	if len(r.rec.uses) != 0 {
		t.Errorf("uses %q", r.rec.uses)
	}
}

func TestKeyFormPlainForAKeyElsewhere(t *testing.T) {
	list := keyIdentities()
	list[0].KeyPath = "~/keys/id_work"
	r := keysOpen(t, "en", list, Request{For: "work-id"})
	r.wantNot(t, "The private key of work")
	if k := r.s.keys; k.name != "id_idset" || k.target != "" {
		t.Errorf("form = %+v", k)
	}
}

func TestKeyReadOnlyWritesNoIdentity(t *testing.T) {
	r := keysOpen(t, "en", keyIdentities(), Request{For: "work-id"})
	r.s.SetReadOnly(true)
	r.generated(t, samplePub)
	if len(r.rec.uses) != 0 {
		t.Errorf("uses %q", r.rec.uses)
	}
	r.click(t, "Choose…")
	r.wantNot(t, "Use this key for which identity?")
}

func TestKeysInChinese(t *testing.T) {
	r := keysOpen(t, "zh-Hans", keyIdentities(), Request{})
	r.s.goos = "darwin"
	r.want(t, "新密钥", "类型", "文件名", "注释", "保存为 ~/.ssh/id_idset 和 ~/.ssh/id_idset.pub。")
	r.click(t, "生成")
	r.s.GenerateDone(keygen.Key{Path: "~/.ssh/id_idset", Public: samplePub, Fingerprint: sampleFp}, nil)
	r.frame()
	r.want(t, "已生成", "复制公钥", "加载到 ssh-agent", "用于身份", "选择…", "再生成一把", "在 Finder 中显示")
}
