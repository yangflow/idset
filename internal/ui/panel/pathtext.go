package panel

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/paths"
)

// pathFace is the type of a path line: all its pieces share it, so that they read as one string.
type pathFace struct {
	mono  bool
	size  float32
	color ui.Color
}

func (f pathFace) apply(e ui.Element) ui.Element {
	if f.mono {
		e = e.Font("monospace")
	}
	return e.FontSize(f.size).TextColor(f.color)
}

// pathLine puts a path and a suffix on one line. When the line is too narrow only the middle of the path gives way,
// to an ellipsis before the last folder ("~/Documents/arch…/payments-gateway/"): the start of the path ("~/", "/"
// or a drive), its last folder and the suffix stay. Only when they do not fit by themselves is the end of the line
// cut off, without an ellipsis. The tooltip is the full text, or tip when it says more. The layout does not trim
// spaces, so the spaces at the start of suffix become padding.
func pathLine(c *ui.Context, path, suffix, tip string, f pathFace) ui.Element {
	space := c.Theme().Rem(0.28)
	lead, mid, last := paths.Split(path)
	piece := func(text string) ui.Element { return f.apply(ui.Text(c, text)).Shrink(0) }
	if tip == "" {
		tip = path + suffix
	}
	return ui.Row(c).MinWidth(0).AlignItems(ui.Center).Tooltip(tip).Children(func() {
		ui.Row(c).MinWidth(0).Clip().AlignItems(ui.Center).Children(func() {
			if lead != "" {
				piece(lead)
			}
			if mid != "" {
				// The middle is clipped inside its own box, so that nothing of it shows when there is no room at all.
				ui.Row(c).MinWidth(0).Clip().Shrink(1e6).Children(func() {
					f.apply(ui.Text(c, mid)).SingleLine().Ellipsis("…/").MinWidth(0)
				})
			}
			if last != "" {
				f.apply(ui.Text(c, last)).SingleLine().MinWidth(0)
			}
		})
		if after := strings.TrimLeft(suffix, " "); after != "" {
			p := piece(after)
			if after != suffix {
				p.Padding(0, 0, 0, space)
			}
		}
	})
}
