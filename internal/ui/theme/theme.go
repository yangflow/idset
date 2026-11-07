// Package theme is Idset's glass theme: the ui.Theme every window uses, light and dark, the fills that are not
// theme roles, and the tokens of design/tokens.json as constants (tokens_gen.go). The values are those of
// design/tokens.json.
package theme

//go:generate go run ./gen ../../../design/tokens.json tokens_gen.go

import (
	"github.com/egoist/mygo/ui"

	"github.com/yangflow/idset/internal/ui/icons"
)

// Fills are the colors of design/tokens.json that ui.Theme has no role for.
type Fills struct {
	// Solid fills a window where its material does not show; SolidDetail is the accounts detail pane there.
	Solid, SolidDetail ui.Color
	// Detail is the accounts detail pane over the material.
	Detail ui.Color
	// Card fills result cards, the identity list and settings groups.
	Card ui.Color
	// Field fills text inputs.
	Field ui.Color
	// AccentSoft is the mint accent as a tint.
	AccentSoft ui.Color
	// Selected fills the chosen row, tab or chip: a neutral step of the state ladder, between hover and pressed, so
	// the accent only ever marks what can be done.
	Selected ui.Color
	// DangerSoft is the danger color as a tint: under an error banner or notice and the read-only badge.
	DangerSoft ui.Color
	// Fg2 is the second-strongest text, between Text and TextMuted: the body of a notice.
	Fg2 ui.Color
	// CardRim is the light line along the top edge of a card, inside it, so a card reads as a sheet of glass.
	CardRim ui.Color
}

var (
	lightFills = Fills{
		Solid: ui.Hex("#ebebef"), SolidDetail: ui.Hex("#f7f7f9"), Detail: ui.RGBA(255, 255, 255, 0.45),
		Card: ui.RGBA(255, 255, 255, 0.82), Field: ui.RGBA(255, 255, 255, 0.22), AccentSoft: ui.RGBA(20, 184, 150, 0.15),
		Selected: ui.RGBA(0, 0, 0, 0.09), DangerSoft: ui.RGBA(200, 16, 46, 0.13), Fg2: ui.Hex("#3c3c42"),
		CardRim: ui.RGBA(255, 255, 255, 0.45),
	}
	darkFills = Fills{
		Solid: ui.Hex("#232326"), SolidDetail: ui.Hex("#2d2d31"), Detail: ui.RGBA(0, 0, 0, 0.18),
		Card: ui.RGBA(255, 255, 255, 0.055), Field: ui.RGBA(0, 0, 0, 0.22), AccentSoft: ui.RGBA(79, 220, 184, 0.18),
		Selected: ui.RGBA(255, 255, 255, 0.11), DangerSoft: ui.RGBA(255, 107, 128, 0.18), Fg2: ui.Hex("#d6d6dc"),
		CardRim: ui.RGBA(255, 255, 255, 0.06),
	}
)

// Style is a type style of design/tokens.json: Size a multiple of the theme's font size, Weight from 100 to 900,
// Line the line height as a multiple of the size, Tracking the letter spacing in em, Mono the system's monospace
// font. tokens_gen.go holds the styles; text takes its size and weight only from one of them.
type Style struct {
	Size     float32
	Weight   int
	Line     float32
	Tracking float32
	Mono     bool
}

// On gives e the style: its size, weight, line height and tracking, and the monospace font for a mono style.
func (s Style) On(c *ui.Context, e ui.Element) ui.Element {
	size := c.Theme().Rem(s.Size)
	e = e.FontSize(size).FontWeight(s.Weight).LineHeight(s.Line)
	if s.Tracking != 0 {
		e = e.LetterSpacing(s.Tracking * size)
	}
	if s.Mono {
		e = e.Font("monospace")
	}
	return e
}

// Card gives e the card shape: RadiusCard, the card fill and the card rim along its top edge, inside it.
func Card(c *ui.Context, e ui.Element) ui.Element {
	f := Of(c)
	return e.Radius(RadiusCard).Background(f.Card).InsetShadow(0, 0.5, 0, 0, f.CardRim)
}

// Of returns the fills of the frame's appearance.
func Of(c *ui.Context) *Fills {
	if c.Theme().Dark {
		return &darkFills
	}
	return &lightFills
}

// Ink is the letter color on a solid badge of r, g, b: near black on a light color, white otherwise.
func Ink(r, g, b uint8) ui.Color {
	if 0.299*float32(r)+0.587*float32(g)+0.114*float32(b) > 165 {
		return ui.Hex("#1d1d1f")
	}
	return ui.Hex("#ffffff")
}

// Apply gives the frame Idset's theme, built on the default one so that its spacing, font size and the desktop's
// text scale stay, and clears the root where the window's material shows; elsewhere the root draws the solid fill.
// Call it first in a window's view.
func Apply(c *ui.Context) {
	t := Build(*c.Theme(), c.Preferences().HighContrast)
	c.SetTheme(&t)
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}
}

// Build returns t with Idset's colors for its appearance. Under increased contrast, borders and secondary text move
// towards the text color and the focus ring is opaque, as in MyGo's default theme.
func Build(t ui.Theme, highContrast bool) ui.Theme {
	if t.Dark {
		t.Background = darkFills.Solid
		t.Surface, t.SurfaceHover, t.SurfacePressed = ui.RGBA(255, 255, 255, 0.07), ui.RGBA(255, 255, 255, 0.09), ui.RGBA(255, 255, 255, 0.14)
		t.Border = ui.RGBA(255, 255, 255, 0.12)
		t.Text, t.TextMuted = ui.Hex("#f5f5f7"), ui.Hex("#b8b8c0")
		t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = ui.Hex("#4fdcb8"), ui.Hex("#72e5c8"), ui.Hex("#94ecd5"), ui.Hex("#10201c")
		t.Danger, t.Warning, t.Success = ui.Hex("#ffa3af"), ui.Hex("#ffb340"), ui.Hex("#3ccf62")
		t.Selection, t.Focus = ui.RGBA(79, 220, 184, 0.32), ui.RGBA(79, 220, 184, 0.6)
		t.Scrollbar = ui.RGBA(255, 255, 255, 0.35)
	} else {
		t.Background = lightFills.Solid
		t.Surface, t.SurfaceHover, t.SurfacePressed = ui.RGBA(0, 0, 0, 0.05), ui.RGBA(0, 0, 0, 0.07), ui.RGBA(0, 0, 0, 0.12)
		t.Border = ui.RGBA(60, 60, 67, 0.18)
		t.Text, t.TextMuted = ui.Hex("#1d1d1f"), ui.Hex("#58585f")
		t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = ui.Hex("#086b5b"), ui.Hex("#065c4e"), ui.Hex("#075a4d"), ui.Hex("#ffffff")
		t.Danger, t.Warning, t.Success = ui.Hex("#b80d29"), ui.Hex("#a86500"), ui.Hex("#1e7d36")
		t.Selection, t.Focus = ui.RGBA(20, 184, 150, 0.25), ui.RGBA(10, 122, 104, 0.55)
		t.Scrollbar = ui.RGBA(0, 0, 0, 0.32)
	}
	// Push buttons and drop-downs are capsules; a symbol button is a circle.
	t.Radius = RadiusPill
	t.ScrollbarWidth = 6
	if highContrast {
		t.Border = t.Border.Mix(t.Text, 0.45)
		t.TextMuted = t.TextMuted.Mix(t.Text, 0.4)
		t.Focus.A = 255
		t.Scrollbar = t.Scrollbar.Alpha(1.6)
	}
	return t
}

// TitleBar draws the title of a window with a hidden title bar (mygo.TitleBarHidden) in the band its controls sit in,
// over the material: centered on the window, small and muted. It drags the window as a title bar does.
func TitleBar(c *ui.Context, title string) {
	t := c.Theme()
	bar := c.TitleBar()
	side := max(bar.Left, bar.Right)
	ui.Row(c).Height(max(bar.Height, t.Space(Control))).PaddingX(side).Justify(ui.Center).AlignItems(ui.Center).DragWindow().
		Children(func() {
			ui.Text(c, title).FontSize(t.Rem(0.85)).TextColor(t.TextMuted).Lines(1, 1)
		})
}

// Dialog draws a dialog in a window of its own, a frameless child of the screen's window whose material shows through
// it: a column of children space-3 apart, named label for assistive technology. Escape sets *open to false. The
// window takes the column's height.
func Dialog(c *ui.Context, open *bool, role ui.Role, label string, body func()) ui.Element {
	t := c.Theme()
	if c.Shortcut(0, ui.KeyEscape) {
		*open = false
	}
	return ui.Column(c).FillWidth().Padding(t.Space(Space6)).Gap(t.Space(Space3)).Role(role).Label(label).Children(body)
}

// Select is a drop-down choosing one of options into *selected: a compact button as wide as the choice, which opens
// a menu of the system's with the options, the chosen one checked.
func Select(c *ui.Context, selected *string, options []string) ui.Element {
	t := c.Theme()
	b := ui.Button(c, "").Height(t.Space(ControlS)).PaddingX(t.Space(Space2)).Gap(t.Space(Space1_5)).Radius(t.Radius).AlignItems(ui.Center).
		Background(t.Surface).Border(0, ui.Color{})
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Text(c, *selected).FontSize(t.Rem(0.92)).SingleLine()
		ui.Icon(c, icons.ChevronDown).FontSize(t.Space(IconXs)).TextColor(t.TextMuted)
	})
	return b.Menu(func(m *ui.Menu) {
		for _, opt := range options {
			if m.Item(opt).Checked(opt == *selected).Chosen() {
				*selected = opt
			}
		}
	})
}
