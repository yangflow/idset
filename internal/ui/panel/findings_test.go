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
	return tt.Image().RGBAAt(12, int(r.Y+r.H/2))
}

func TestFindingRowsOnTheMatchedCard(t *testing.T) {
	for _, c := range []struct {
		name     string
		findings []match.Finding
		shown    []string
		tips     []string
	}{
		{"no email", []match.Finding{noEmail}, []string{"Commits have no author"},
			[]string{"No user.email in this repository. Git refuses to commit or guesses an address."}},
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
	details, _ := tt.Find("bob@example.com")
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
	a, _ := plain.Find("Identities")
	b, _ := with.Find("Identities")
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
	if got := edge(t, warning, "bob@example.com"); got != warn {
		t.Errorf("warnings use the Warning outline: %v, want %v", got, warn)
	}
	plainEdge := edge(t, plain, "bob@example.com")
	if plainEdge == warn {
		t.Fatal("a card without findings has no outline")
	}
	dangerEdge := edge(t, danger, "bob@example.com")
	if dangerEdge == warn || dangerEdge == plainEdge {
		t.Errorf("an error finding uses the Danger outline: %v (warning %v, plain %v)", dangerEdge, warn, plainEdge)
	}
	if got := edge(t, dangerOnly, "bob@example.com"); got != dangerEdge {
		t.Errorf("one error is enough: %v, want %v", got, dangerEdge)
	}
}

func TestFindingsInChinese(t *testing.T) {
	list := findingList()
	for _, c := range []struct {
		f     match.Finding
		shown []string
		tip   string
		name  string
	}{
		{noEmail, []string{"提交没有作者"}, "这个仓库没有 user.email，Git 会拒绝提交或自己猜一个地址。", "提交没有作者。这个仓库没有 user.email，Git 会拒绝提交或自己猜一个地址。"},
		{otherEmail, []string{"提交会署名为", "carol@example.com", "（client）"}, "仓库的 user.email 是 carol@example.com，属于 client，不是 work。", "提交会署名为 carol@example.com（client）。仓库的 user.email 是 carol@example.com，属于 client，不是 work。"},
		{unknownEmail, []string{"提交会署名为", "someone@example.org"}, "仓库的 user.email 是 someone@example.org，没有身份用这个邮箱。", "提交会署名为 someone@example.org。仓库的 user.email 是 someone@example.org，没有身份用这个邮箱。"},
		{otherKey, []string{"推送会用", "~/.ssh/id_other"}, "core.sshCommand 指定了 ~/.ssh/id_other，不是 work 的密钥。", "推送会用 ~/.ssh/id_other。core.sshCommand 指定了 ~/.ssh/id_other，不是 work 的密钥。"},
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
		// The Chinese full stop joins the sentence and its reason in the row's name, not an ASCII one.
		if !slices.Contains(tt.Texts(), c.name) {
			t.Errorf("the row's name should be %q: %q", c.name, tt.Texts())
		}
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

func TestFindingsAreAnnouncedInChinese(t *testing.T) {
	s := New(Text("zh-CN"), env, (&recorder{}).actions())
	s.SetAccounts(findingList(), nil, "")
	tt := ui.NewTester(s.View, 340, 480)
	res := withFindings(otherEmail, otherKey)
	s.Drop([]string{res.Root})
	res.Dir = res.Root
	s.SetResult(res.Root, res, nil)
	tt.Frame()
	got := tt.Announcements()
	if want := []string{"Git 在这里用 work。提交会署名为 carol@example.com（client）。推送会用 ~/.ssh/id_other"}; !slices.Equal(got, want) {
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

// blankRun is the widest stretch, in pixels, between two inked columns of the box r of the frame.
func blankRun(tt *ui.Tester, r ui.Rect, scale float32) int {
	img := tt.Image()
	x0, x1 := int(r.X*scale), int((r.X+r.W)*scale)
	y0, y1 := int(r.Y*scale), int((r.Y+r.H)*scale)
	bg := img.RGBAAt(x1+int(scale), y0)
	inked := func(x int) bool {
		for y := y0; y < y1; y++ {
			c := img.RGBAAt(x, y)
			if d := int(c.R) - int(bg.R) + int(c.G) - int(bg.G) + int(c.B) - int(bg.B); d > 60 || d < -60 {
				return true
			}
		}
		return false
	}
	widest, run, seen := 0, 0, false
	for x := x0; x < x1; x++ {
		switch {
		case inked(x):
			widest, run, seen = max(widest, run), 0, true
		case seen:
			run++
		}
	}
	return widest
}

// The middle of a path or of an email gives way to an ellipsis, and what follows it comes right after: the width a
// box has left over after the cut, less than one glyph, is at the end of the line, not in the line. This measures
// the pixels of the frame, from the end of a row's words to the end of its line, at every width in a range, against
// the widest stretch between two glyphs of the same line with room to spare.
func TestATruncatedLineHasNoGapAfterItsEllipsis(t *testing.T) {
	long := match.Finding{Kind: match.UnknownEmail, Value: "robert.fitzgerald-baker.the-third.of-many@client.example"}
	key := match.Finding{Kind: match.OtherKey, Value: "~/.ssh/clients/northwind-traders/id_ed25519_deploy"}
	const scale = 4
	runs := func(w int) []int {
		_, tt, _ := card(t, withFindings(long, key), findingList()...)
		tt.SetScale(scale)
		tt.SetSize(w, 480)
		tt.Frame()
		var out []int
		for _, words := range []string{"Commits use", "Pushes use"} {
			r, ok := tt.Find(words)
			if !ok {
				t.Fatalf("no %q in %q", words, tt.Texts())
			}
			// From the words to the end of the line, short of the card's border.
			line := ui.Rect{X: r.X + r.W + 4, Y: r.Y, W: float32(w) - 34 - (r.X + r.W + 4), H: r.H}
			if w > 600 {
				line.W = 300
			}
			out = append(out, blankRun(tt, line, scale))
		}
		return out
	}
	natural := runs(800)
	for w := 300; w <= 340; w++ {
		for i, gap := range runs(w) {
			if gap > natural[i]+4 {
				t.Errorf("width %d, line %d: %d pixels of nothing in a line whose glyphs are at most %d apart", w, i, gap, natural[i])
			}
		}
	}
}

func TestLongValuesGiveWayButTheDomainAndTheNameStay(t *testing.T) {
	list := findingList()
	list[1].Label = "Northwind"
	long := match.Finding{Kind: match.OtherEmail, Value: "robert.fitzgerald-baker.the-third.of-many@client.example", Owner: "client"}
	key := match.Finding{Kind: match.OtherKey, Value: "~/.ssh/clients/northwind-traders/id_ed25519_deploy"}
	_, tt, _ := card(t, withFindings(long, key), list...)
	want(t, tt, "Commits use", "@client.example (Northwind)", "Pushes use", "~/", "id_ed25519_deploy")
	within(t, tt, "robert.fitzgerald-baker.the-third.of-many@client.example (Northwind)", ".ssh/clients/northwind-traders/id_ed25519_deploy")
	if slices.ContainsFunc(tt.Texts(), func(s string) bool { return s == " id_ed25519_deploy" || s == "/id_ed25519_deploy" }) {
		t.Errorf("the last folder is a piece of its own: %q", tt.Texts())
	}
}
