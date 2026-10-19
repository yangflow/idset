// Package theme is Idset's glass theme: the ui.Theme every window uses, light and dark, and the fills that are not
// theme roles. The values are those of design/tokens.md.
package theme

import (
	"github.com/egoist/mygo/plugins/glass"
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
	// AccentSoft is Git orange as a tint.
	AccentSoft ui.Color
	// Rim is the inner edge of the card menu and dialogs; Scrim is behind a dialog.
	Rim, Scrim ui.Color
}

var (
	lightFills = Fills{
		Solid: ui.Hex("#f2f2f5"), SolidDetail: ui.Hex("#fbfbfd"), Detail: ui.RGBA(255, 255, 255, 0.45),
		Card: ui.RGBA(255, 255, 255, 0.2), Field: ui.RGBA(255, 255, 255, 0.45), AccentSoft: ui.RGBA(240, 80, 51, 0.16),
		Rim: ui.RGBA(255, 255, 255, 0.7), Scrim: ui.RGBA(0, 0, 0, 0.1),
	}
	darkFills = Fills{
		Solid: ui.Hex("#2a2a2e"), SolidDetail: ui.Hex("#323236"), Detail: ui.RGBA(0, 0, 0, 0.18),
		Card: ui.RGBA(255, 255, 255, 0.08), Field: ui.RGBA(0, 0, 0, 0.22), AccentSoft: ui.RGBA(255, 106, 74, 0.22),
		Rim: ui.RGBA(255, 255, 255, 0.18), Scrim: ui.RGBA(0, 0, 0, 0.28),
	}
)

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
		t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = ui.Hex("#ff6a4a"), ui.Hex("#ff8468"), ui.Hex("#ff9a83"), ui.Hex("#1d1d1f")
		t.Danger, t.Warning, t.Success = ui.Hex("#ff5c74"), ui.Hex("#ffb340"), ui.Hex("#3ccf62")
		t.Selection, t.Focus = ui.RGBA(255, 106, 74, 0.35), ui.RGBA(255, 132, 104, 0.6)
		t.Inverse, t.InverseText = ui.Hex("#f5f5f7"), ui.Hex("#1d1d1f")
		t.Scrollbar = ui.RGBA(255, 255, 255, 0.35)
	} else {
		t.Background = lightFills.Solid
		t.Surface, t.SurfaceHover, t.SurfacePressed = ui.RGBA(0, 0, 0, 0.05), ui.RGBA(0, 0, 0, 0.07), ui.RGBA(0, 0, 0, 0.12)
		t.Border = ui.RGBA(60, 60, 67, 0.18)
		t.Text, t.TextMuted = ui.Hex("#1d1d1f"), ui.Hex("#6b6b72")
		t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = ui.Hex("#c63a1c"), ui.Hex("#ad3218"), ui.Hex("#962a14"), ui.Hex("#ffffff")
		t.Danger, t.Warning, t.Success = ui.Hex("#c8102e"), ui.Hex("#b86e00"), ui.Hex("#1e7d36")
		t.Selection, t.Focus = ui.RGBA(240, 80, 51, 0.25), ui.RGBA(198, 58, 28, 0.55)
		t.Inverse, t.InverseText = ui.Hex("#1d1d1f"), ui.Hex("#f5f5f7")
		t.Scrollbar = ui.RGBA(0, 0, 0, 0.32)
	}
	t.Radius = 6
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

// Dialog gives a DialogBase its look: the scrim over the whole window and a pane of Liquid Glass, radius 16 and
// width wide, with its rim and a shadow; its children are Space(3) apart.
func Dialog(c *ui.Context, backdrop, panel ui.Element, width float32) {
	t, f := c.Theme(), Of(c)
	backdrop.Background(f.Scrim)
	panel.Width(width).Padding(t.Space(5)).Gap(t.Space(3)).Radius(16).Material(glass.Glass{}).Border(0.5, f.Rim).
		Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.32))
}

// Menu gives a PopoverBase its look: a pane of Liquid Glass, radius 10 and width wide (as wide as its entries for 0),
// with its rim and a shadow.
func Menu(c *ui.Context, panel ui.Element, width float32) {
	if width > 0 {
		panel.Width(width)
	}
	panel.Padding(c.Theme().Space(1.25)).Radius(10).Material(glass.Glass{}).Border(0.5, Of(c).Rim).
		Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.25))
}

// Select is a drop-down choosing one of options into *selected: a compact button as wide as the choice, and a Menu
// of the options under it, the chosen one checked.
func Select(c *ui.Context, selected *string, options []string) ui.Element {
	t := c.Theme()
	sel := ui.SelectBase(c, selected)
	b := sel.Trigger.Height(t.Space(6.5)).PaddingX(t.Space(2.5)).Gap(t.Space(1.5)).Radius(t.Radius).AlignItems(ui.Center).
		Background(t.Surface)
	if b.Hovered() || sel.Open() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Text(c, *selected).FontSize(t.Rem(0.92)).SingleLine()
		ui.Icon(c, icons.ChevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted)
	})
	sel.Popup(func(panel ui.Element) {
		Menu(c, panel, 0)
		panel.Margin(t.Space(1), 0, 0, 0)
		for _, opt := range options {
			item := sel.Item(opt).Height(t.Space(6.5)).PaddingX(t.Space(1.5)).Gap(t.Space(1)).Radius(t.Radius).AlignItems(ui.Center)
			if item.Highlighted() {
				item.Background(t.SurfaceHover)
			}
			item.Children(func() {
				ui.Row(c).Width(t.Space(3.5)).Children(func() {
					if opt == *selected {
						ui.Icon(c, icons.Check).FontSize(t.Space(3.5)).TextColor(t.Accent)
					}
				})
				ui.Text(c, opt).FontSize(t.Rem(0.92)).SingleLine().Padding(0, t.Space(2), 0, 0)
			})
		}
	})
	return b
}
