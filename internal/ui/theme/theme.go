// Package theme is Idset's glass theme: the ui.Theme every window uses, light and dark, and the fills that are not
// theme roles. The values are those of design/tokens.md.
package theme

import (
	"github.com/egoist/mygo/ui"

	"github.com/yangflow/idset/internal/ui/icons"
)

// Fills are the colors of design/tokens.md that ui.Theme has no role for.
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
	// CardRim is the light line along the top edge of a card, inside it, so a card reads as a sheet of glass.
	CardRim ui.Color
}

var (
	lightFills = Fills{
		Solid: ui.Hex("#f2f2f5"), SolidDetail: ui.Hex("#fbfbfd"), Detail: ui.RGBA(255, 255, 255, 0.45),
		Card: ui.RGBA(255, 255, 255, 0.2), Field: ui.RGBA(255, 255, 255, 0.22), AccentSoft: ui.RGBA(20, 184, 150, 0.15),
		CardRim: ui.RGBA(255, 255, 255, 0.45),
	}
	darkFills = Fills{
		Solid: ui.Hex("#2a2a2e"), SolidDetail: ui.Hex("#323236"), Detail: ui.RGBA(0, 0, 0, 0.18),
		Card: ui.RGBA(255, 255, 255, 0.08), Field: ui.RGBA(0, 0, 0, 0.22), AccentSoft: ui.RGBA(79, 220, 184, 0.18),
		CardRim: ui.RGBA(255, 255, 255, 0.06),
	}
)

const (
	// Capsule is the theme radius, so push buttons and drop-downs are capsules; a symbol button is a circle.
	Capsule = 999
	// FieldRadius rounds text inputs, which set it themselves, and the rows of the accounts sidebar.
	FieldRadius = 8
	// CardRadius rounds cards, settings groups, banners and the drop zone.
	CardRadius = 12
)

// Card gives e the card shape: CardRadius, the card fill and the card rim along its top edge, inside it.
func Card(c *ui.Context, e ui.Element) ui.Element {
	f := Of(c)
	return e.Radius(CardRadius).Background(f.Card).InsetShadow(0, 0.5, 0, 0, f.CardRim)
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
		t.Surface, t.SurfaceHover, t.SurfacePressed = ui.RGBA(255, 255, 255, 0.08), ui.RGBA(255, 255, 255, 0.11), ui.RGBA(255, 255, 255, 0.16)
		t.Border = ui.RGBA(255, 255, 255, 0.12)
		t.Text, t.TextMuted = ui.Hex("#f5f5f7"), ui.Hex("#a3a3ab")
		t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = ui.Hex("#4fdcb8"), ui.Hex("#72e5c8"), ui.Hex("#94ecd5"), ui.Hex("#10201c")
		t.Danger, t.Warning, t.Success = ui.Hex("#ff5c74"), ui.Hex("#ffb340"), ui.Hex("#3ccf62")
		t.Selection, t.Focus = ui.RGBA(79, 220, 184, 0.32), ui.RGBA(79, 220, 184, 0.45)
		t.Inverse, t.InverseText = ui.Hex("#f5f5f7"), ui.Hex("#1d1d1f")
		t.Scrollbar = ui.RGBA(255, 255, 255, 0.35)
	} else {
		t.Background = lightFills.Solid
		t.Surface, t.SurfaceHover, t.SurfacePressed = ui.RGBA(0, 0, 0, 0.05), ui.RGBA(0, 0, 0, 0.07), ui.RGBA(0, 0, 0, 0.12)
		t.Border = ui.RGBA(60, 60, 67, 0.18)
		t.Text, t.TextMuted = ui.Hex("#1d1d1f"), ui.Hex("#6b6b72")
		t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = ui.Hex("#0a7a68"), ui.Hex("#086a5a"), ui.Hex("#075a4d"), ui.Hex("#ffffff")
		t.Danger, t.Warning, t.Success = ui.Hex("#c8102e"), ui.Hex("#b86e00"), ui.Hex("#1e7d36")
		t.Selection, t.Focus = ui.RGBA(20, 184, 150, 0.25), ui.RGBA(10, 122, 104, 0.4)
		t.Inverse, t.InverseText = ui.Hex("#1d1d1f"), ui.Hex("#f5f5f7")
		t.Scrollbar = ui.RGBA(0, 0, 0, 0.32)
	}
	t.Radius = Capsule
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
	ui.Row(c).Height(max(bar.Height, t.Space(7))).PaddingX(side).Justify(ui.Center).AlignItems(ui.Center).DragWindow().
		Children(func() {
			ui.Text(c, title).FontSize(t.Rem(0.85)).TextColor(t.TextMuted).Lines(1, 1)
		})
}

// DialogWidth is the width of a dialog's window, in DIPs.
const DialogWidth = 340

// Dialog draws a dialog in a window of its own, a frameless child of the screen's window whose material shows through
// it: a column of children Space(3) apart, named label for assistive technology. Escape sets *open to false. The
// window takes the column's height.
func Dialog(c *ui.Context, open *bool, role ui.Role, label string, body func()) ui.Element {
	t := c.Theme()
	if c.Shortcut(0, ui.KeyEscape) {
		*open = false
	}
	return ui.Column(c).FillWidth().Padding(t.Space(5)).Gap(t.Space(3)).Role(role).Label(label).Children(body)
}

// Select is a drop-down choosing one of options into *selected: a compact button as wide as the choice, which opens
// a menu of the system's with the options, the chosen one checked.
func Select(c *ui.Context, selected *string, options []string) ui.Element {
	t := c.Theme()
	b := ui.Button(c, "").Height(t.Space(6.5)).PaddingX(t.Space(2.5)).Gap(t.Space(1.5)).Radius(t.Radius).AlignItems(ui.Center).
		Background(t.Surface).Border(0, ui.Color{})
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Text(c, *selected).FontSize(t.Rem(0.92)).SingleLine()
		ui.Icon(c, icons.ChevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted)
	})
	return b.Menu(func(m *ui.Menu) {
		for _, opt := range options {
			if m.Item(opt).Checked(opt == *selected).Chosen() {
				*selected = opt
			}
		}
	})
}
