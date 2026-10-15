package panel

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
)

func cloneList() []account.Account {
	return []account.Account{work(), client(), personal()}
}

func httpsResult(origin string) match.Result {
	r := matched("/home/ada/work/api", "work", false)
	r.Origin = origin
	return r
}

// clock makes the panel's time move only when the test says so.
func clock(s *Screen) func(time.Duration) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	return func(d time.Duration) { now = now.Add(d) }
}

const (
	setURL   = "git remote set-url origin git@github-work:acme/api.git"
	httpsURL = "https://github.com/acme/api.git"
)

func TestHTTPSRowCopiesTheRemoteCommand(t *testing.T) {
	s, tt, rec := card(t, httpsResult(httpsURL), cloneList()...)
	advance := clock(s)
	want(t, tt, "Remote is HTTPS")
	if tip := hover(t, tt, "Remote is HTTPS"); !tip("Pushes over HTTPS do not use this identity’s SSH alias or key.") {
		t.Errorf("the row explains itself: %q", tt.Texts())
	}
	if edge(t, tt, "Remote is HTTPS") != edge(t, tt, "Folders") {
		t.Error("an HTTPS remote is not a finding: the outline stays as it was")
	}
	click(t, tt, "Copy command to use SSH")
	if !slices.Equal(rec.copied, []string{setURL}) {
		t.Fatalf("copied %q", rec.copied)
	}
	tt.Frame()
	want(t, tt, "Copied")
	if !slices.Contains(tt.Announcements(), "Copied") {
		t.Errorf("the copy is announced: %q", tt.Announcements())
	}
	advance(1300 * time.Millisecond)
	tt.Frame()
	wantNot(t, tt, "Copied")
	if len(rec.cloned) != 0 {
		t.Errorf("the remote command is not a clone: %v", rec.cloned)
	}
}

func TestHTTPSRowIsHiddenWithoutAnHTTPSRemoteOrAnAlias(t *testing.T) {
	for _, c := range map[string]struct {
		origin string
		list   []account.Account
		id     string
	}{
		"ssh origin": {"git@github.com:acme/api.git", cloneList(), "work"},
		"no origin":  {"", cloneList(), "work"},
		"no alias":   {httpsURL, cloneList(), "client"},
	} {
		r := httpsResult(c.origin)
		r.ID, r.Covering = c.id, []match.Cover{{ID: c.id}}
		_, tt, _ := card(t, r, c.list...)
		wantNot(t, tt, "Remote is HTTPS", "Copy command to use SSH")
		if c.id == "client" {
			wantNot(t, tt, "Repository to clone")
		}
	}
}

func TestHTTPSRowSitsUnderTheFindings(t *testing.T) {
	r := httpsResult(httpsURL)
	r.Findings = []match.Finding{{Kind: match.NoEmail}}
	_, tt, _ := card(t, r, cloneList()...)
	finding, _ := tt.Find("Commits have no author")
	row, _ := tt.Find("Remote is HTTPS")
	if row.Y <= finding.Y {
		t.Errorf("the HTTPS row is the last of the block: %v after %v", row.Y, finding.Y)
	}
}

func TestCloneHelperStartsWithTheOrigin(t *testing.T) {
	s, tt, rec := card(t, httpsResult(httpsURL), cloneList()...)
	clock(s)
	want(t, tt, "acme/api", "git clone git@github-work:acme/api.git")
	click(t, tt, "Copy the clone command")
	if !slices.Equal(rec.copied, []string{"git clone git@github-work:acme/api.git"}) {
		t.Fatalf("copied %q", rec.copied)
	}
	if !slices.Equal(rec.cloned, [][2]string{{"work", "acme/"}}) {
		t.Errorf("the prefix is remembered and the use recorded: %v", rec.cloned)
	}
	tt.Frame()
	want(t, tt, "Copied")
}

func TestCloneHelperStartsWithTheRememberedPrefix(t *testing.T) {
	s, tt, rec := screen(t, cloneList()...)
	s.SetPrefixes(map[string]string{"work": "acme/"})
	s.Drop([]string{"/home/ada/work/api"})
	s.SetResult("/home/ada/work/api", httpsResult("git@github.com:solo"), nil)
	tt.Frame()
	// A prefix alone is not a repository: nothing to copy, and a quiet hint instead of an error.
	want(t, tt, "Enter owner/repo or a repository URL")
	wantNot(t, tt, "git clone git@github-work:")
	click(t, tt, "Repository to clone")
	tt.Type("web")
	tt.Key(0, ui.KeyEnter)
	if !slices.Equal(rec.copied, []string{"git clone git@github-work:acme/web.git"}) {
		t.Errorf("the prefix is in the input: %q", rec.copied)
	}
}

func TestCloneHelperTakesWhatTheUserTypes(t *testing.T) {
	for _, c := range []struct{ in, out, prefix string }{
		{"acme/web", "git clone git@github-work:acme/web.git", "acme/"},
		{"https://github.com/acme/web", "git clone git@github-work:acme/web.git", "acme/"},
		{"git clone git@github.com:acme/web.git", "git clone git@github-work:acme/web.git", "acme/"},
	} {
		s, tt, rec := card(t, httpsResult(""), cloneList()...)
		clock(s)
		click(t, tt, "Repository to clone")
		tt.Type(c.in)
		tt.Key(0, ui.KeyEnter)
		if !slices.Equal(rec.copied, []string{c.out}) || !slices.Equal(rec.cloned, [][2]string{{"work", c.prefix}}) {
			t.Errorf("%q: copied %q, cloned %v", c.in, rec.copied, rec.cloned)
		}
	}
}

func TestCloneHelperDoesNotCopyWhatItCannotRead(t *testing.T) {
	for _, in := range []string{"", "acme/", "web"} {
		s, tt, rec := card(t, httpsResult(""), cloneList()...)
		clock(s)
		if in != "" {
			click(t, tt, "Repository to clone")
			tt.Type(in)
			tt.Key(0, ui.KeyEnter)
		}
		if err := tt.Click("Copy the clone command"); err == nil && len(rec.copied) != 0 {
			t.Errorf("%q: copied %q", in, rec.copied)
		}
		if len(rec.copied) != 0 || len(rec.cloned) != 0 {
			t.Errorf("%q: copied %q, cloned %v", in, rec.copied, rec.cloned)
		}
		want(t, tt, "Enter owner/repo or a repository URL")
		wantNot(t, tt, "Copied")
	}
}

func TestIdentityRowsOpenOneHelperAtATime(t *testing.T) {
	s, tt, rec := screen(t, cloneList()...)
	s.SetPrefixes(map[string]string{"work": "acme/", "personal": "me/"})
	tt.Frame()
	wantNot(t, tt, "Repository to clone")
	// Only the identities with an alias can clone.
	if n := count(tt, "Clone with this identity"); n != 2 {
		t.Fatalf("two identities have an alias: %d in %q", n, tt.Texts())
	}
	toggle(t, tt, "work")
	want(t, tt, "Repository to clone")
	if len(rec.opened) != 0 {
		t.Errorf("the disclosure button does not open the accounts window: %v", rec.opened)
	}
	if !tt.Focused("Repository to clone") {
		t.Error("the input has the focus on open")
	}
	if count(tt, "Repository to clone") != 1 {
		t.Fatalf("one helper: %q", tt.Texts())
	}
	// Opening another row closes the first.
	toggle(t, tt, "personal")
	if count(tt, "Repository to clone") != 1 {
		t.Fatalf("the second row replaces the first: %q", tt.Texts())
	}
	tt.Type("web")
	tt.Key(0, ui.KeyEnter)
	if !slices.Equal(rec.copied, []string{"git clone git@github-personal:me/web.git"}) {
		t.Errorf("the second row starts with its own prefix: %q", rec.copied)
	}
	// The same button closes it.
	toggle(t, tt, "personal")
	wantNot(t, tt, "Repository to clone")
	// The body of a row still opens the accounts window.
	click(t, tt, "personal")
	if !slices.Equal(rec.opened, []string{"personal"}) {
		t.Errorf("opened %v", rec.opened)
	}
}

// toggle clicks the disclosure button at the end of the row of an identity.
func toggle(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("no %q in %q", label, tt.Texts())
	}
	tt.ClickAt(340-16-1-8-12, r.Y+r.H/2)
}

func TestCloneHelperOfARowCopiesAndRemembers(t *testing.T) {
	s, tt, rec := screen(t, cloneList()...)
	clock(s)
	toggle(t, tt, "work")
	tt.Type("acme/web")
	tt.Key(0, ui.KeyEnter)
	if !slices.Equal(rec.copied, []string{"git clone git@github-work:acme/web.git"}) ||
		!slices.Equal(rec.cloned, [][2]string{{"work", "acme/"}}) {
		t.Fatalf("copied %q, cloned %v", rec.copied, rec.cloned)
	}
	want(t, tt, "Copied")
}

func TestCloneTextsInChinese(t *testing.T) {
	r := &recorder{}
	s := New(Text("zh-CN"), env, r.actions())
	s.SetAccounts(cloneList(), nil, "/home/ada/.config/idset/accounts.json")
	tt := ui.NewTester(s.View, 340, 480)
	s.Drop([]string{"/home/ada/work/api"})
	s.SetResult("/home/ada/work/api", httpsResult(httpsURL), nil)
	tt.Frame()
	want(t, tt, "远程地址是 HTTPS", "复制改地址命令", "要克隆的仓库", "复制克隆命令", "用这个身份克隆")
}
