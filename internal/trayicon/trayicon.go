// Package trayicon draws the tray icon, an identity badge, so that no image file has to be maintained by hand.
package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// PNG returns a 32×32 icon. A template icon is black on transparent for macOS to tint to the menu bar; otherwise
// it is a mid gray that stays visible on light and dark taskbars.
func PNG(template bool) []byte {
	ink := color.NRGBA{R: 0x8a, G: 0x8a, B: 0x8a, A: 0xff}
	if template {
		ink = color.NRGBA{A: 0xff}
	}
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	set := func(x, y int) { img.SetNRGBA(x, y, ink) }
	// Card outline with rounded corners: 3 to 28 wide, 7 to 25 high, 2 pixels thick.
	for x := 5; x <= 26; x++ {
		for _, y := range []int{7, 8, 24, 25} {
			set(x, y)
		}
	}
	for y := 9; y <= 23; y++ {
		for _, x := range []int{3, 4, 27, 28} {
			set(x, y)
		}
	}
	for _, p := range [][2]int{{4, 8}, {27, 8}, {4, 24}, {27, 24}} {
		set(p[0], p[1])
	}
	// Head and shoulders on the left.
	for y := 11; y <= 22; y++ {
		for x := 7; x <= 15; x++ {
			dx, dy := float64(x)-11, float64(y)-13.5
			head := dx*dx+dy*dy <= 6.5
			shoulders := y >= 18 && (dx*dx)/16+(float64(y)-22)*(float64(y)-22)/16 <= 1
			if head || shoulders {
				set(x, y)
			}
		}
	}
	// Two text lines on the right.
	for _, row := range [][3]int{{18, 24, 13}, {18, 22, 18}} {
		for x := row[0]; x <= row[1]; x++ {
			set(x, row[2])
			set(x, row[2]+1)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}
