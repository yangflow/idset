// Package trayicon draws the tray icon, an identity badge, so that no image file has to be maintained by hand.
package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Badge is the mark at the lower right of the icon: the worst entry of the diagnostics summary.
type Badge int

const (
	// None: the card alone.
	None Badge = iota
	// Warning: a triangle with "!".
	Warning
	// Error: a circle with "!".
	Error
)

// The badge colors off macOS, Warning and Danger, readable on light and dark taskbars.
var (
	warningInk = color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}
	dangerInk  = color.NRGBA{R: 0xdc, G: 0x26, B: 0x26, A: 0xff}
)

// PNG returns a 32×32 icon. A template icon is black on transparent for macOS to tint to the menu bar; otherwise
// it is a mid gray that stays visible on light and dark taskbars. A badge sits at the lower right, 1.5 apart from
// the card: in a template its "!" is cut out, elsewhere it is in Warning or Danger with a white "!".
func PNG(template bool, badge Badge) []byte {
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
	drawBadge(img, template, ink, badge)
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// drawBadge clears the badge's shape grown by 1.5 so that it stands apart from the card, fills the shape, then
// draws or cuts out the "!".
func drawBadge(img *image.NRGBA, template bool, ink color.NRGBA, badge Badge) {
	var shape func(x, y, grow float64) bool
	top, dot := 21.2, 27.9
	fill := dangerInk
	switch badge {
	case Error:
		shape = func(x, y, grow float64) bool { return math.Hypot(x-24.5, y-24.5) <= 6.5+grow }
	case Warning:
		a, b, c := [2]float64{24.5, 17.2}, [2]float64{31.5, 30}, [2]float64{17.5, 30}
		shape = func(x, y, grow float64) bool {
			p := [2]float64{x, y}
			return inTriangle(p, a, b, c) || grow > 0 && min(toSegment(p, a, b), toSegment(p, b, c), toSegment(p, c, a)) <= grow
		}
		top, dot, fill = 21.8, 28.2, warningInk
	default:
		return
	}
	if template {
		fill = ink
	}
	mark := func(x, y, _ float64) bool {
		return toSegment([2]float64{x, y}, [2]float64{24.5, top}, [2]float64{24.5, top + 3.6}) <= 0.9 ||
			toSegment([2]float64{x, y}, [2]float64{24.5, dot}, [2]float64{24.5, dot + 0.1}) <= 0.9
	}
	paint(img, func(x, y float64) bool { return shape(x, y, 1.5) }, color.NRGBA{})
	paint(img, func(x, y float64) bool { return shape(x, y, 0) }, fill)
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	if template {
		white = color.NRGBA{}
	}
	paint(img, func(x, y float64) bool { return mark(x, y, 0) }, white)
}

// paint covers the pixels inside a shape with c, by how much of each pixel the shape covers, from 4×4 samples. A
// transparent c clears instead.
func paint(img *image.NRGBA, inside func(x, y float64) bool, c color.NRGBA) {
	const n = 4
	b := img.Bounds()
	for py := b.Min.Y; py < b.Max.Y; py++ {
		for px := b.Min.X; px < b.Max.X; px++ {
			hits := 0
			for sy := range n {
				for sx := range n {
					if inside(float64(px)+(float64(sx)+0.5)/n, float64(py)+(float64(sy)+0.5)/n) {
						hits++
					}
				}
			}
			if hits == 0 {
				continue
			}
			cov := float64(hits) / (n * n)
			old := img.NRGBAAt(px, py)
			oa := float64(old.A) / 255
			if c.A == 0 {
				old.A = uint8(math.Round(255 * oa * (1 - cov)))
				img.SetNRGBA(px, py, old)
				continue
			}
			a := cov + oa*(1-cov)
			mix := func(v, w uint8) uint8 { return uint8(math.Round((float64(v)*cov + float64(w)*oa*(1-cov)) / a)) }
			img.SetNRGBA(px, py, color.NRGBA{R: mix(c.R, old.R), G: mix(c.G, old.G), B: mix(c.B, old.B), A: uint8(math.Round(255 * a))})
		}
	}
}

func inTriangle(p, a, b, c [2]float64) bool {
	side := func(p, q, r [2]float64) float64 { return (p[0]-r[0])*(q[1]-r[1]) - (q[0]-r[0])*(p[1]-r[1]) }
	d1, d2, d3 := side(p, a, b), side(p, b, c), side(p, c, a)
	neg, pos := d1 < 0 || d2 < 0 || d3 < 0, d1 > 0 || d2 > 0 || d3 > 0
	return !(neg && pos)
}

// toSegment is the distance from p to the segment from a to b.
func toSegment(p, a, b [2]float64) float64 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	t := ((p[0]-a[0])*dx + (p[1]-a[1])*dy) / (dx*dx + dy*dy)
	t = max(0, min(1, t))
	return math.Hypot(p[0]-a[0]-t*dx, p[1]-a[1]-t*dy)
}
