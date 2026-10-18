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

// layout is where to look in each kind of icon: in the bow, in the branch cut out of it, in the shaft, in each
// badge's body and "!", and at a pixel of the key that each badge's gap clears.
type layout struct {
	w, h               int
	bow, branch, shaft [2]int
	body, mark, gap    map[Badge][2]int
}

var layouts = map[bool]layout{
	true: {w: 54, h: 36, bow: [2]int{8, 18}, branch: [2]int{14, 18}, shaft: [2]int{35, 17},
		body: map[Badge][2]int{Warning: {42, 31}, Error: {42, 29}},
		mark: map[Badge][2]int{Warning: {45, 25}, Error: {45, 25}},
		gap:  map[Badge][2]int{Warning: {45, 18}, Error: {39, 22}}},
	false: {w: 32, h: 32, bow: [2]int{4, 16}, branch: [2]int{8, 16}, shaft: [2]int{20, 15},
		body: map[Badge][2]int{Warning: {21, 28}, Error: {21, 24}},
		mark: map[Badge][2]int{Warning: {24, 23}, Error: {24, 23}},
		gap:  map[Badge][2]int{Warning: {24, 16}, Error: {20, 18}}},
}

func TestPNG(t *testing.T) {
	for template, l := range layouts {
		img := decode(t, PNG(template, None), l.w, l.h)
		if nrgba(img, [2]int{0, 0}).A != 0 || nrgba(img, l.bow).A != 0xff || nrgba(img, l.shaft).A != 0xff {
			t.Errorf("template %v: the key must be solid on a transparent background", template)
		}
		if nrgba(img, l.branch).A != 0 {
			t.Errorf("template %v: the branch must be cut out of the bow", template)
		}
		for _, badge := range []Badge{Warning, Error} {
			if p := l.gap[badge]; nrgba(img, p).A == 0 {
				t.Errorf("template %v: %v is not part of the key", template, p)
			}
			if p := l.body[badge]; nrgba(img, p).A != 0 {
				t.Errorf("template %v: no badge without a severity, but %v is drawn", template, p)
			}
		}
	}
}

func TestBadges(t *testing.T) {
	for _, c := range []struct {
		badge Badge
		ink   color.NRGBA
	}{{Warning, warningInk}, {Error, dangerInk}} {
		for template, l := range layouts {
			img := decode(t, PNG(template, c.badge), l.w, l.h)
			body, mark := nrgba(img, l.body[c.badge]), nrgba(img, l.mark[c.badge])
			want := c.ink
			if template {
				want = color.NRGBA{A: 0xff}
			}
			if body != want {
				t.Errorf("badge %d template %v: body %v, want %v", c.badge, template, body, want)
			}
			if template && mark.A != 0 || !template && mark != (color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}) {
				t.Errorf("badge %d template %v: mark %v", c.badge, template, mark)
			}
			// The key stops short of the badge: the gap is transparent.
			if a := nrgba(img, l.gap[c.badge]).A; a != 0 {
				t.Errorf("badge %d template %v: no gap between the key and the badge", c.badge, template)
			}
			// The rest of the key is untouched.
			if nrgba(img, l.bow).A != 0xff || nrgba(img, l.shaft).A != 0xff {
				t.Errorf("badge %d template %v: the key lost its bow or shaft", c.badge, template)
			}
		}
	}
}
