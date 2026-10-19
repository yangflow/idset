package theme

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestApplyGivesTheFrameIdsetsTheme(t *testing.T) {
	for _, dark := range []bool{false, true} {
		var bg ui.Color
		var fills *Fills
		tester := ui.NewTester(func(c *ui.Context) {
			Apply(c)
			bg, fills = c.Theme().Background, Of(c)
		}, 320, 320)
		tester.SetDark(dark)
		tester.SetVibrancy(true)
		tester.Frame()
		if bg != fills.Solid {
			t.Errorf("dark %v: Background = %v, want the solid fill %v", dark, bg, fills.Solid)
		}
	}
}

func TestBuildFollowsTheAppearance(t *testing.T) {
	light := Build(ui.Theme{}, false)
	dark := Build(ui.Theme{Dark: true}, false)
	if light.Accent != ui.Hex("#c63a1c") || dark.Accent != ui.Hex("#ff6a4a") {
		t.Errorf("Accent = %v / %v", light.Accent, dark.Accent)
	}
	if dark.AccentText != ui.Hex("#1d1d1f") {
		t.Errorf("dark AccentText = %v, want dark text", dark.AccentText)
	}
}

func TestBuildRaisesContrast(t *testing.T) {
	for _, dark := range []bool{false, true} {
		plain, high := Build(ui.Theme{Dark: dark}, false), Build(ui.Theme{Dark: dark}, true)
		if high.Focus.A != 255 || plain.Focus.A == 255 {
			t.Errorf("dark %v: focus alpha %d / %d", dark, plain.Focus.A, high.Focus.A)
		}
		if high.Border.A <= plain.Border.A {
			t.Errorf("dark %v: border not stronger: %v / %v", dark, plain.Border, high.Border)
		}
		if high.TextMuted == plain.TextMuted {
			t.Errorf("dark %v: TextMuted unchanged", dark)
		}
	}
}

func TestInkIsWhiteOnSaturatedColors(t *testing.T) {
	white, black := ui.Hex("#ffffff"), ui.Hex("#1d1d1f")
	for _, tc := range []struct {
		r, g, b uint8
		want    ui.Color
	}{
		{0xf0, 0x50, 0x33, white},
		{0x8e, 0x5c, 0xf7, white},
		{0x2f, 0x7d, 0xf6, white},
		{0xd4, 0xa0, 0x17, white},
		{0xf5, 0xe6, 0x6b, black},
		{0xff, 0xff, 0xff, black},
	} {
		if got := Ink(tc.r, tc.g, tc.b); got != tc.want {
			t.Errorf("Ink(%d, %d, %d) = %v, want %v", tc.r, tc.g, tc.b, got, tc.want)
		}
	}
}

func TestSelectIsAsWideAsItsChoiceAndChoosesFromAMenu(t *testing.T) {
	choice := "GitHub"
	options := []string{"GitHub", "GitLab", "Gitea", "Custom"}
	tester := ui.NewTester(func(c *ui.Context) {
		Apply(c)
		ui.Row(c).Children(func() { Select(c, &choice, options).Label("Platform") })
	}, 320, 320)
	trigger, ok := tester.Find("Platform")
	if !ok || trigger.W > 100 {
		t.Fatalf("the button is %v, want at most 100 wide", trigger)
	}
	if err := tester.Click("Platform"); err != nil {
		t.Fatal(err)
	}
	if got := tester.Menu(); !slices.Equal(got, options) {
		t.Fatalf("the menu is %q, want %q", got, options)
	}
	if err := tester.ChooseMenuItem("Gitea"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if choice != "Gitea" || !tester.HasText("Gitea") {
		t.Fatalf("choose: %q %q", choice, tester.Texts())
	}
}
