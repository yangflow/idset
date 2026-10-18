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
		_, _, _, bow := img.At(3, 13).RGBA()
		_, _, _, hole := img.At(9, 11).RGBA()
		_, _, _, shaft := img.At(20, 13).RGBA()
		if corner != 0 || bow == 0 || shaft == 0 {
			t.Fatal("the key must be drawn on a transparent background")
		}
		if hole != 0 {
			t.Fatal("the bow must be an outline")
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
		gap   [2]int
	}{{Warning, warningInk, [2]int{21, 28}, [2]int{25, 17}}, {Error, dangerInk, [2]int{21, 24}, [2]int{22, 17}}} {
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
			// The bit nearest the badge stops short of it: the gap is transparent.
			if a := nrgba(img, c.gap[0], c.gap[1]).A; a != 0 {
				t.Errorf("badge %d template %v: no gap at the bit", c.badge, template)
			}
			// The bow and the shaft are untouched.
			if nrgba(img, 3, 13).A != 0xff || nrgba(img, 20, 13).A != 0xff {
				t.Errorf("badge %d template %v: the key lost its bow or shaft", c.badge, template)
			}
		}
	}
}
