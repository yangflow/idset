package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func decode(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 32 || b.Dy() != 32 {
		t.Fatalf("size %v", b)
	}
	return img
}

func TestPNG(t *testing.T) {
	for _, template := range []bool{true, false} {
		img := decode(t, PNG(template, None))
		_, _, _, corner := img.At(0, 0).RGBA()
		_, _, _, edge := img.At(3, 16).RGBA()
		if corner != 0 || edge == 0 {
			t.Fatal("the outline must be drawn on a transparent background")
		}
		_, _, _, badge := img.At(24, 22).RGBA()
		if badge != 0 {
			t.Fatal("no badge without a severity")
		}
	}
}

func TestBadges(t *testing.T) {
	nrgba := func(img image.Image, x, y int) color.NRGBA {
		return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	}
	for _, c := range []struct {
		badge Badge
		ink   color.NRGBA
		body  [2]int
	}{{Warning, warningInk, [2]int{21, 28}}, {Error, dangerInk, [2]int{21, 24}}} {
		for _, template := range []bool{true, false} {
			img := decode(t, PNG(template, c.badge))
			body, mark := nrgba(img, c.body[0], c.body[1]), nrgba(img, 24, 23)
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
			// The card's right edge stops short of the badge: the gap is transparent.
			if a := nrgba(img, 28, 18).A; a != 0 && c.badge == Error {
				t.Errorf("badge %d template %v: no gap at the card's edge", c.badge, template)
			}
			// The rest of the card is untouched.
			if nrgba(img, 3, 16).A != 0xff {
				t.Errorf("badge %d template %v: the card lost its left edge", c.badge, template)
			}
		}
	}
}
