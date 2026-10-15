package panel

import (
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
// cut off, without an ellipsis. The tooltip is the full text, or tip when it says more.
//
// Everything after the start is one text, and the ellipsis it ends a cut with carries the last folder and the
// suffix: a box shows whole glyphs only, so the width it has left over after the cut sits at the end of the line
// and not between the ellipsis and the last folder.
func pathLine(c *ui.Context, path, suffix, tip string, f pathFace) ui.Element {
	lead, mid, last := paths.Split(path)
	if tip == "" {
		tip = path + suffix
	}
	mark := "…" + suffix
	if mid != "" {
		mark = "…/" + last + suffix
	}
	return ui.Row(c).MinWidth(0).AlignItems(ui.Center).Tooltip(tip).Children(func() {
		ui.Row(c).MinWidth(0).Clip().AlignItems(ui.Center).Children(func() {
			if lead != "" {
				f.apply(ui.Text(c, lead)).Shrink(0)
			}
			f.apply(ui.Text(c, mid+last+suffix)).SingleLine().Ellipsis(mark).MinWidth(0)
		})
	})
}
