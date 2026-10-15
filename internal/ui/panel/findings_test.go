package panel

import (
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
)

const strayKey = "~/.ssh/id_other"

// findingList is the matched identity, which has a key, and another one whose email the repository may carry.
func findingList() []account.Account {
	work := identity("work", "github-work", "bob@example.com", account.GitHub, 0, "~/work/")
	work.KeyPath = "~/.ssh/id_work"
	other := identity("client", "", "carol@example.com", account.GitHub, 0, "~/client/")
	return []account.Account{work, other}
}

func withFindings(fs ...match.Finding) match.Result {
	r := matched("/home/ada/work/api", "work", false)
	r.Findings = fs
	return r
}

var (
	noEmail      = match.Finding{Kind: match.NoEmail}
	otherEmail   = match.Finding{Kind: match.OtherEmail, Value: "carol@example.com", Owner: "client"}
	unknownEmail = match.Finding{Kind: match.UnknownEmail, Value: "someone@example.org"}
	otherKey     = match.Finding{Kind: match.OtherKey, Value: strayKey}
)

// edge is the color of the card's left border, a pixel in from the panel's padding, level with the row of text.
func edge(t *testing.T, tt *ui.Tester, text string) color.RGBA {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %q in %q", text, tt.Texts())
	}
	return tt.Image().RGBAAt(16, int(r.Y+r.H/2))
}

func TestFindingRowsOnTheMatchedCard(t *testing.T) {
	for _, c := range []struct {
		name     string
		findings []match.Finding
		shown    []string
		tips     []string
	}{
		{"no email", []match.Finding{noEmail}, []string{"Commits have no author"},
			[]string{"This repository has no user.email, so Git will stop at the commit or guess an address."}},
		{"another identity's email", []match.Finding{otherEmail}, []string{"Commits use", "carol@example.com", "(client)"},
			[]string{"This repository’s user.email is carol@example.com, which belongs to client, not work."}},
		{"an unknown email", []match.Finding{unknownEmail}, []string{"Commits use", "someone@example.org"},
			[]string{"This repository’s user.email is someone@example.org, which no identity uses."}},
		{"another key", []match.Finding{otherKey}, []string{"Pushes use", "~/.ssh/id_other"},
			[]string{"core.sshCommand picks ~/.ssh/id_other, not work’s key."}},
	} {
		_, tt, _ := card(t, withFindings(c.findings...), findingList()...)
		want(t, tt, c.shown...)
		for _, tip := range c.tips {
			// The reason is the accessible name of the row, and what hovering the row says.
			if !slices.ContainsFunc(tt.Texts(), func(s string) bool { return len(s) > len(tip) && s[len(s)-len(tip):] == tip }) {
				t.Errorf("%s: the row's name should end with %q: %q", c.name, tip, tt.Texts())
			}
			if tipShown := hover(t, tt, c.shown[0]); !tipShown(tip) {
				t.Errorf("%s: the tooltip is %q: %q", c.name, tip, tt.Texts())
			}
		}
	}
}

func TestFindingRowsComeEmailFirstThenKey(t *testing.T) {
	_, tt, _ := card(t, withFindings(otherEmail, otherKey), findingList()...)
	email, _ := tt.Find("Commits use")
	key, _ := tt.Find("Pushes use")
	details, _ := tt.Find("Folders")
	if !(details.Y < email.Y && email.Y < key.Y) {
		t.Errorf("details %v, email %v, key %v", details.Y, email.Y, key.Y)
	}
	// Two 28 DIP rows, one under the other.
	if !same(key.Y-email.Y, 28) {
		t.Errorf("rows are 28 high: %v", key.Y-email.Y)
	}
}

func TestCardWithoutFindingsHasNoRows(t *testing.T) {
	_, plain, _ := card(t, withFindings(), findingList()...)
	_, with, _ := card(t, withFindings(noEmail), findingList()...)
	wantNot(t, plain, "Commits have no author")
	a, _ := plain.Find("Drop a folder, or")
	b, _ := with.Find("Drop a folder, or")
	// A row is 28 high with 4 above and below it, and a hairline.
	if got := b.Y - a.Y; !same(got, 28+8+1) {
		t.Errorf("one finding adds %v DIPs", got)
	}
}

func TestCardBorderTakesTheWorstSeverity(t *testing.T) {
	list := findingList()
	_, plain, _ := card(t, withFindings(), list...)
	_, warning, _ := card(t, withFindings(noEmail, unknownEmail), list...)
	_, danger, _ := card(t, withFindings(noEmail, otherKey), list...)
	_, dangerOnly, _ := card(t, withFindings(otherEmail), list...)
	_, noMatch, _ := card(t, match.Result{Kind: match.NoMatch, Root: "/home/ada/scratch"}, list...)
	warn := edge(t, noMatch, "No identity matches this folder")
	if got := edge(t, warning, "Alias"); got != warn {
		t.Errorf("warnings use the Warning outline: %v, want %v", got, warn)
	}
	plainEdge := edge(t, plain, "Alias")
	if plainEdge == warn {
		t.Fatal("a card without findings keeps the plain outline")
	}
	dangerEdge := edge(t, danger, "Alias")
	if dangerEdge == warn || dangerEdge == plainEdge {
		t.Errorf("an error finding uses the Danger outline: %v (warning %v, plain %v)", dangerEdge, warn, plainEdge)
	}
	if got := edge(t, dangerOnly, "Alias"); got != dangerEdge {
		t.Errorf("one error is enough: %v, want %v", got, dangerEdge)
	}
}

func TestFindingsInChinese(t *testing.T) {
	list := findingList()
	for _, c := range []struct {
		f     match.Finding
		shown []string
		tip   string
	}{
		{noEmail, []string{"提交没有作者"}, "这个仓库读不到 user.email，提交时 Git 会报错或自己猜一个地址。"},
		{otherEmail, []string{"提交会署名为", "carol@example.com", "（client）"}, "仓库的 user.email 是 carol@example.com，属于 client，不是 work。"},
		{unknownEmail, []string{"提交会署名为", "someone@example.org"}, "仓库的 user.email 是 someone@example.org，没有身份用这个邮箱。"},
		{otherKey, []string{"推送会用", "~/.ssh/id_other"}, "core.sshCommand 指定了 ~/.ssh/id_other，不是 work 的密钥。"},
	} {
		s := New(Text("zh-CN"), env, (&recorder{}).actions())
		s.SetAccounts(list, nil, "")
		tt := ui.NewTester(s.View, 340, 480)
		res := withFindings(c.f)
		s.Drop([]string{res.Root})
		res.Dir = res.Root
		s.SetResult(res.Root, res, nil)
		tt.Frame()
		want(t, tt, c.shown...)
		if tipShown := hover(t, tt, c.shown[0]); !tipShown(c.tip) {
			t.Errorf("%q: tooltip %q in %q", c.shown[0], c.tip, tt.Texts())
		}
	}
}

func TestFindingsAreAnnounced(t *testing.T) {
	_, tt, _ := card(t, withFindings(otherEmail, otherKey), findingList()...)
	got := tt.Announcements()
	if want := []string{"Git uses work here. Commits use carol@example.com (client). Pushes use ~/.ssh/id_other"}; !slices.Equal(got, want) {
		t.Errorf("announced %q, want %q", got, want)
	}
}

func TestAnEmailOfAGoneIdentityIsJustUnknown(t *testing.T) {
	gone := match.Finding{Kind: match.OtherEmail, Value: "dan@example.com", Owner: "removed"}
	_, tt, _ := card(t, withFindings(gone), findingList()...)
	want(t, tt, "Commits use", "dan@example.com")
	wantNot(t, tt, "(removed)")
	if tipShown := hover(t, tt, "Commits use"); !tipShown("This repository’s user.email is dan@example.com, which no identity uses.") {
		t.Errorf("tooltip: %q", tt.Texts())
	}
}

func TestLongValuesGiveWayButTheDomainAndTheNameStay(t *testing.T) {
	list := findingList()
	list[1].Label = "Northwind Traders"
	long := match.Finding{Kind: match.OtherEmail, Value: "robert.fitzgerald-baker.the-third.of-many@client.example", Owner: "client"}
	key := match.Finding{Kind: match.OtherKey, Value: "~/.ssh/clients/northwind-traders/id_ed25519_deploy"}
	_, tt, _ := card(t, withFindings(long, key), list...)
	want(t, tt, "Commits use", "@client.example", "(Northwind Traders)", "Pushes use", "~/", "id_ed25519_deploy")
	within(t, tt, "@client.example", "(Northwind Traders)", "id_ed25519_deploy")
	local, _ := tt.Find("robert.fitzgerald-baker.the-third.of-many")
	domain, _ := tt.Find("@client.example")
	owner, _ := tt.Find("(Northwind Traders)")
	if local.X+local.W > domain.X+0.5 || domain.X+domain.W > owner.X+0.5 {
		t.Errorf("the local part, the domain and the name sit side by side: %+v %+v %+v", local, domain, owner)
	}
}

// The middle of a key path gives way to an ellipsis and the last folder follows it with no gap.
func TestTruncatedKeyPathHasNoGapBeforeItsLastFolder(t *testing.T) {
	key := match.Finding{Kind: match.OtherKey, Value: "~/.ssh/clients/northwind-traders/id_ed25519_deploy"}
	_, tt, _ := card(t, withFindings(key), findingList()...)
	mid, ok := tt.Find(".ssh/clients/northwind-traders/")
	last, ok2 := tt.Find("id_ed25519_deploy")
	if !ok || !ok2 {
		t.Fatalf("pieces of the path: %q", tt.Texts())
	}
	if gap := last.X - (mid.X + mid.W); gap < -0.01 || gap > 0.01 {
		t.Errorf("the last folder starts %v DIPs after the middle ends", gap)
	}
	if slices.Contains(tt.Texts(), " id_ed25519_deploy") || slices.Contains(tt.Texts(), "/id_ed25519_deploy") {
		t.Errorf("the last folder is a piece of its own: %q", tt.Texts())
	}
}
