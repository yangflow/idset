package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func decode(t *testing.T, b []byte, w, h int) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != w || b.Dy() != h {
		t.Fatalf("size %v, want %d×%d", b, w, h)
	}
	return img
}

func nrgba(img image.Image, p [2]int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(p[0], p[1])).(color.NRGBA)
}

// layout is where to look in one placement of the folder: in its body, in the keyhole cut through it, and in the
// head and the foot of the keyhole lit inside the cut.
type layout struct {
	w, h                  int
	body, cut, head, foot [2]int
}

var (
	templateLayout = layout{w: 54, h: 36, body: [2]int{8, 18}, cut: [2]int{21, 18}, head: [2]int{26, 18}, foot: [2]int{26, 25}}
	// Off macOS the folder is as wide as the icon, and smaller at the upper left when a badge is shown.
	squareLayout  = layout{w: 32, h: 32, body: [2]int{4, 15}, cut: [2]int{12, 16}, head: [2]int{15, 16}, foot: [2]int{15, 20}}
	badgedLayout  = layout{w: 32, h: 32, body: [2]int{4, 11}, cut: [2]int{10, 12}, head: [2]int{13, 12}, foot: [2]int{13, 15}}
	white         = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	templateInk   = color.NRGBA{A: 0xff}
	squareInk     = color.NRGBA{R: 0x8a, G: 0x8a, B: 0x8a, A: 0xff}
	badgeBodies   = map[bool]map[Badge][2]int{true: {Warning: {42, 31}, Error: {42, 29}}, false: {Warning: {21, 28}, Error: {21, 24}}}
	badgeMarks    = map[bool]map[Badge][2]int{true: {Warning: {45, 25}, Error: {45, 25}}, false: {Warning: {24, 23}, Error: {24, 23}}}
	badgeGaps     = map[bool]map[Badge][2]int{true: {Warning: {45, 18}, Error: {39, 22}}, false: {Warning: {24, 16}, Error: {22, 17}}}
	badges        = []Badge{Warning, Error}
	badgeInks     = map[Badge]color.NRGBA{Warning: warningInk, Error: dangerInk}
	allPlacements = []struct {
		template bool
		badge    Badge
		l        layout
		ink      color.NRGBA
	}{
		{true, None, templateLayout, templateInk}, {true, Warning, templateLayout, templateInk}, {true, Error, templateLayout, templateInk},
		{false, None, squareLayout, squareInk}, {false, Warning, badgedLayout, squareInk}, {false, Error, badgedLayout, squareInk},
	}
)

func TestFolder(t *testing.T) {
	for _, c := range allPlacements {
		img := decode(t, PNG(c.template, c.badge), c.l.w, c.l.h)
		if nrgba(img, [2]int{0, c.l.h - 1}).A != 0 {
			t.Errorf("template %v badge %d: the background must be transparent", c.template, c.badge)
		}
		if nrgba(img, c.l.body) != c.ink {
			t.Errorf("template %v badge %d: the folder must be solid", c.template, c.badge)
		}
		if nrgba(img, c.l.cut).A != 0 {
			t.Errorf("template %v badge %d: the keyhole must be cut through the folder", c.template, c.badge)
		}
		if nrgba(img, c.l.head) != c.ink || nrgba(img, c.l.foot) != c.ink {
			t.Errorf("template %v badge %d: the inner keyhole must be solid", c.template, c.badge)
		}
	}
}

// Without a severity there is no badge: every pixel is the folder's ink, at some coverage, and a template keeps the
// folder whole where a badge would cut its mark.
func TestNoBadge(t *testing.T) {
	for _, c := range allPlacements {
		if c.badge != None {
			continue
		}
		img := decode(t, PNG(c.template, None), c.l.w, c.l.h)
		for y := range c.l.h {
			for x := range c.l.w {
				if p := nrgba(img, [2]int{x, y}); p.A != 0 && (p.R != c.ink.R || p.G != c.ink.G || p.B != c.ink.B) {
					t.Fatalf("template %v: %v at %d,%d is not the folder's ink", c.template, p, x, y)
				}
			}
		}
		if c.template && nrgba(img, badgeMarks[true][Warning]) != templateInk {
			t.Errorf("template: the folder is cut where no badge is shown")
		}
	}
}

func TestBadges(t *testing.T) {
	for _, badge := range badges {
		for _, template := range []bool{true, false} {
			w, h, grid := place(template, badge)
			img := decode(t, PNG(template, badge), w, h)
			body, mark := nrgba(img, badgeBodies[template][badge]), nrgba(img, badgeMarks[template][badge])
			want := badgeInks[badge]
			if template {
				want = templateInk
			}
			if body != want {
				t.Errorf("badge %d template %v: body %v, want %v", badge, template, body, want)
			}
			if template && mark.A != 0 || !template && mark != white {
				t.Errorf("badge %d template %v: mark %v", badge, template, mark)
			}
			// The badge's gap cuts into the folder: a point of the folder, clear of the keyhole, is transparent.
			p := badgeGaps[template][badge]
			gx, gy := grid(float64(p[0])+0.5, float64(p[1])+0.5)
			if !inside(folder, func(x, y float64) (float64, float64) { return x, y }, gx, gy) || keyholeCut.inside(gx, gy) {
				t.Fatalf("badge %d template %v: %v is not on the folder", badge, template, p)
			}
			if a := nrgba(img, p).A; a != 0 {
				t.Errorf("badge %d template %v: no gap between the folder and the badge", badge, template)
			}
		}
	}
}
