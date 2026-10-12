package trayicon

import (
	"bytes"
	"image/png"
	"testing"
)

func TestPNG(t *testing.T) {
	for _, template := range []bool{true, false} {
		img, err := png.Decode(bytes.NewReader(PNG(template)))
		if err != nil {
			t.Fatal(err)
		}
		if b := img.Bounds(); b.Dx() != 32 || b.Dy() != 32 {
			t.Fatalf("size %v", b)
		}
		_, _, _, corner := img.At(0, 0).RGBA()
		_, _, _, edge := img.At(3, 16).RGBA()
		if corner != 0 || edge == 0 {
			t.Fatal("the outline must be drawn on a transparent background")
		}
	}
}
