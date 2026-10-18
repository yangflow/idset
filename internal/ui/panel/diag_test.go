package panel

import (
	"errors"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/diagnose"
	"github.com/yangflow/idset/internal/match"
)

// diagList is a GitHub identity with a key, one without an alias, which is never checked, and a custom one.
func diagList() []account.Account {
	w := work()
	w.KeyPath = "~/.ssh/id_work"
	own := identity("own", "git-own", "o@example.com", account.Custom, 0, "~/own/")
	own.HostName = "git.example.com"
	return []account.Account{w, client(), own}
}

func passed(id, user string) diagnose.Check { return diagnose.Check{ID: id, OK: true, User: user} }

func failed(id string, f diagnose.Failure) diagnose.Check { return diagnose.Check{ID: id, Failure: f} }

var errFake = errors.New("no passphrase in the keychain")

var lockedKey = diagnose.KeyState{HasPath: true, Exists: true, Agent: true, Passphrase: true, Fingerprint: "SHA256:abc"}

// diagScreen shows the list with a finished round of checks.
func diagScreen(t *testing.T, checks ...diagnose.Check) (*Screen, *ui.Tester, *recorder) {
	t.Helper()
	s, tt, r := screen(t, diagList()...)
	s.SetDiagnostics(Diagnostics{Checks: checks})
	tall(tt)
	return s, tt, r
}

// tall makes the panel as high as the screen, so that the summary at the end of the list shows without scrolling.
func tall(tt *ui.Tester) {
	tt.SetSize(340, 900)
	tt.Frame()
}

func TestRowsHaveNoDotsBeforeTheFirstReport(t *testing.T) {
	s, tt, _ := screen(t, diagList()...)
	wantNot(t, tt, "Checking the connection…", "Configuration OK")
	if s.Worst() != 0 {
		t.Errorf("worst %v", s.Worst())
	}
}

func TestDotsWhileTheFirstRoundRuns(t *testing.T) {
	s, tt, _ := screen(t, diagList()...)
	s.SetDiagnostics(Diagnostics{Pending: []string{"work", "own"}})
	tt.Frame()
	// Each identity with an alias has a ring; the summary waits for the round.
	if n := count(tt, "Checking the connection…"); n != 2 {
		t.Errorf("two pending dots: %q", tt.Texts())
	}
	wantNot(t, tt, "Configuration OK")
	if s.Worst() != 0 {
		t.Errorf("worst %v", s.Worst())
	}
	// Re-check waits while a check runs.
	if r, _ := tt.Find("Re-check"); r.W == 0 {
		t.Error("re-check is in the footer")
	}
}

func TestAHealthyRoundSaysSoOnce(t *testing.T) {
	s, tt, rec := diagScreen(t, passed("work", "bob"), passed("own", ""))
	for _, dot := range []string{"Connected · GitHub user bob", "Connected"} {
		if _, ok := tt.Find(dot); !ok {
			t.Errorf("no dot %q", dot)
		}
	}
	want(t, tt, "Configuration OK")
	if s.Worst() != 0 {
		t.Errorf("worst %v", s.Worst())
	}
	click(t, tt, "Re-check")
	if len(rec.checked) != 1 || rec.checked[0] != nil {
		t.Errorf("re-check checks all: %q", rec.checked)
	}
}

func TestRejectedKeyBar(t *testing.T) {
	s, tt, rec := diagScreen(t, failed("work", diagnose.Rejected), passed("own", ""))
	want(t, tt, "GitHub rejected the key", "Copy Public Key", "Open GitHub Settings")
	if _, ok := tt.Find("Check failed · GitHub rejected the key"); !ok {
		t.Error("the dot says why the check failed")
	}
	click(t, tt, "Copy Public Key")
	if !slices.Equal(rec.pubs, []string{"work"}) {
		t.Fatalf("pubs %q", rec.pubs)
	}
	s.PublicKeyDone("work", diagnose.ErrNoPublicKey)
	tt.Frame()
	want(t, tt, "No .pub file")
	s.PublicKeyDone("work", nil)
	tt.Frame()
	want(t, tt, "Copied")
	wantNot(t, tt, "No .pub file")
	click(t, tt, "Open GitHub Settings")
	if got := tt.OpenedURLs(); !slices.Equal(got, []string{"https://github.com/settings/keys"}) {
		t.Errorf("opened %q", got)
	}
	click(t, tt, "Check Again")
	if len(rec.checked) != 1 || !slices.Equal(rec.checked[0], []string{"work"}) {
		t.Errorf("checked %q", rec.checked)
	}
	if s.Worst() != match.Error {
		t.Errorf("worst %v", s.Worst())
	}
}

func TestCustomIdentityNamesItsHostAndHasNoSettingsPage(t *testing.T) {
	_, tt, _ := diagScreen(t, passed("work", "bob"), failed("own", diagnose.Rejected))
	want(t, tt, "git.example.com rejected the key", "Copy Public Key")
	wantNot(t, tt, "Open Custom Settings", "Open  Settings")
}

func TestUnreachableBars(t *testing.T) {
	for f, text := range map[diagnose.Failure]string{
		diagnose.TimedOut: "github.com timed out",
		diagnose.NoHost:   "github.com not found",
		diagnose.Refused:  "github.com refused the connection",
		diagnose.HostKey:  "github.com’s host key changed",
	} {
		_, tt, _ := diagScreen(t, failed("work", f), passed("own", ""))
		want(t, tt, text, "Check Again")
		wantNot(t, tt, "Copy Public Key")
	}
}

func TestMissingKeyBar(t *testing.T) {
	c := failed("work", diagnose.NoKeyFile)
	c.Key = diagnose.KeyState{HasPath: true}
	_, tt, rec := diagScreen(t, c, passed("own", ""))
	want(t, tt, "Key not found", "~/.ssh/id_work", "Check Again", "Generate Key")
	wantNot(t, tt, "Copy Public Key")
	click(t, tt, "Generate Key")
	if !slices.Equal(rec.keygen, []string{"work"}) {
		t.Errorf("generate key for %q", rec.keygen)
	}
}

func TestLockedKeyUnlocksWithTheKeychainOnMacOS(t *testing.T) {
	c := failed("work", diagnose.Rejected)
	c.Key = lockedKey
	s, tt, rec := diagScreen(t, c, passed("own", ""))
	s.goos = "darwin"
	tt.Frame()
	want(t, tt, "Key is locked", "Unlock with Keychain")
	click(t, tt, "Unlock with Keychain")
	if !slices.Equal(rec.unlocked, []string{"work"}) {
		t.Fatalf("unlocked %q", rec.unlocked)
	}
	// Without a passphrase in the keychain the bar gives the command to run instead.
	s.UnlockDone("work", errFake)
	tt.Frame()
	want(t, tt, "The keychain has no passphrase for it. Run this once in a terminal:")
	wantNot(t, tt, "Unlock with Keychain")
	click(t, tt, "Copy the ssh-add command")
	if !slices.Equal(rec.copied, []string{"ssh-add ~/.ssh/id_work"}) {
		t.Errorf("copied %q", rec.copied)
	}
}

func TestLockedKeyGivesTheCommandElsewhere(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		c := failed("work", diagnose.Rejected)
		c.Key = lockedKey
		s, tt, rec := diagScreen(t, c, passed("own", ""))
		s.goos = goos
		tt.Frame()
		wantNot(t, tt, "Unlock with Keychain", "The keychain has no passphrase for it. Run this once in a terminal:")
		click(t, tt, "Copy the ssh-add command")
		if len(rec.copied) != 1 {
			t.Errorf("%s: copied %q", goos, rec.copied)
		}
	}
}

func TestMissingSSHIsSaidOnce(t *testing.T) {
	for goos, hint := range map[string]string{
		"windows": "Idset checks connections with the system’s OpenSSH client. Add “OpenSSH Client” in Settings → System → Optional features, or install Git for Windows.",
		"darwin":  "Idset checks connections with the system’s OpenSSH client, /usr/bin/ssh. Check that it has not been removed or replaced.",
		"linux":   "Idset checks connections with the system’s OpenSSH client. Install it with your package manager, usually the openssh-client package.",
	} {
		s, tt, rec := diagScreen(t, failed("work", diagnose.NoCommand), failed("own", diagnose.NoCommand))
		s.goos = goos
		tt.Frame()
		if count(tt, "ssh not found") != 1 {
			t.Errorf("%s: one notice: %q", goos, tt.Texts())
		}
		want(t, tt, hint)
		// No bars under the rows and no summary: the round could not run.
		wantNot(t, tt, "GitHub rejected the key", "Configuration OK", "2 errors")
		if s.Worst() != match.Error {
			t.Errorf("%s: worst %v", goos, s.Worst())
		}
		click(t, tt, "Check Again")
		if len(rec.checked) != 1 || !slices.Equal(rec.checked[0], []string{"own", "work"}) {
			t.Errorf("%s: checked %q", goos, rec.checked)
		}
	}
}

func TestSummaryOpensOnNewEntriesAndFolds(t *testing.T) {
	s, tt, rec := diagScreen(t, failed("work", diagnose.Rejected), passed("own", ""))
	want(t, tt, "1 error", "github-work failed its check", "Run ssh -vT git@github-work in a terminal to see why.", "NET001")
	click(t, tt, "Copy the Command")
	if !slices.Equal(rec.copied, []string{"ssh -vT git@github-work"}) {
		t.Errorf("copied %q", rec.copied)
	}
	click(t, tt, "1 error")
	tt.Frame()
	wantNot(t, tt, "github-work failed its check")
	// The same entries keep it folded; a new one opens it.
	s.SetDiagnostics(Diagnostics{Checks: []diagnose.Check{failed("work", diagnose.Rejected), passed("own", "")}})
	tt.Frame()
	wantNot(t, tt, "github-work failed its check")
	s.SetDiagnostics(Diagnostics{Checks: []diagnose.Check{failed("work", diagnose.Rejected), passed("own", "")}, UserHosts: []string{"git-own"}})
	tt.Frame()
	want(t, tt, "1 error · 1 warning", "github-work failed its check", "~/.ssh/config also has Host git-own", "SSH001")
}

func TestShadowedAliasIsAWarning(t *testing.T) {
	s, tt, _ := screen(t, diagList()...)
	s.SetDiagnostics(Diagnostics{Checks: []diagnose.Check{passed("work", "bob"), passed("own", "")}, UserHosts: []string{"github-work"}})
	tall(tt)
	want(t, tt, "1 warning", "Idset’s block comes first, so it wins. Consider removing the one you wrote.")
	if s.Worst() != match.Warning {
		t.Errorf("worst %v", s.Worst())
	}
}

func TestMatchedFindingsRegenerate(t *testing.T) {
	r := withFindings(otherEmail)
	s, tt, rec := card(t, r, diagList()...)
	s.SetDiagnostics(Diagnostics{Checks: []diagnose.Check{passed("work", "bob"), passed("own", "")}})
	tall(tt)
	want(t, tt, "This folder disagrees with work", "GIT001")
	click(t, tt, "Regenerate")
	if rec.regen != 1 {
		t.Errorf("regen %d", rec.regen)
	}
}

func TestTwoGlobalsNameTheOneInEffect(t *testing.T) {
	s, tt, _ := screen(t, personal(), identity("zed", "", "z@example.com", account.GitHub, 0))
	s.SetDiagnostics(Diagnostics{Checks: []diagnose.Check{passed("personal", "p")}})
	tall(tt)
	want(t, tt, "2 global identities", "zed is the one in effect; give the others folders.", "GIT002")
}

func TestHTTPSRemoteBarAddsTheRewrites(t *testing.T) {
	s, tt, rec := card(t, httpsResult(httpsURL), diagList()...)
	s.SetDiagnostics(Diagnostics{Checks: []diagnose.Check{passed("work", "bob"), passed("own", "")}})
	tall(tt)
	click(t, tt, "Add Rewrite")
	if !slices.Equal(rec.rewrites, []string{"work"}) {
		t.Errorf("rewrites %q", rec.rewrites)
	}
	wantNot(t, tt, "Check Again")
	// With the rule in place the bar goes.
	list := diagList()
	list[0].URLRewrites = account.DefaultRewrites(account.GitHub, "github-work")
	s.SetAccounts(list, nil, "")
	tt.Frame()
	wantNot(t, tt, "Add Rewrite")
}

func TestDiagnosticsInChinese(t *testing.T) {
	r := &recorder{}
	s := New(Text("zh-Hans"), env, r.actions())
	s.SetAccounts(diagList(), nil, "")
	s.SetDiagnostics(Diagnostics{Checks: []diagnose.Check{failed("work", diagnose.TimedOut), passed("own", "")}})
	tt := ui.NewTester(s.View, 340, 480)
	want(t, tt, "连接 github.com 超时", "1 个错误", "github-work 检查失败", "复制排查命令")
}
