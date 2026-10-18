// Package trayicon draws the tray icon, the product icon's key, so that no image file has to be maintained by hand.
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
	// None: the key alone.
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
// the key: in a template its "!" is cut out, elsewhere it is in Warning or Danger with a white "!".
func PNG(template bool, badge Badge) []byte {
	ink := color.NRGBA{R: 0x8a, G: 0x8a, B: 0x8a, A: 0xff}
	if template {
		ink = color.NRGBA{A: 0xff}
	}
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	paint(img, inKey, ink)
	drawBadge(img, template, ink, badge)
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// inKey is the product icon's key as an outline, pointing right: a rounded diamond bow, 2 thick around a dot,
// centered at (9, 14); a shaft 3 thick from the bow to 29.5; two bits 3 wide under the shaft. It sits 2 above the
// middle so that a badge leaves the shaft whole.
func inKey(x, y float64) bool {
	const cy = 14
	bow := diamond(x-9, y-cy, 7.5, 1.5)
	switch {
	case bow <= 0 && bow >= -2, math.Hypot(x-9, y-cy) <= 1.6:
		return true
	case y >= cy-1.5 && y <= cy+1.5 && x >= 15 && x <= 28:
		return true
	case math.Hypot(x-28, y-cy) <= 1.5:
		return true
	}
	return bit(x, y, 20, 23, cy, cy+6.5) || bit(x, y, 25, 28, cy, cy+5)
}

// diamond is the signed distance from (x, y) to a diamond centered at the origin, its corners r from the center
// before they are rounded by round: negative inside.
func diamond(x, y, r, round float64) float64 {
	u, v := math.Abs(x+y)/math.Sqrt2, math.Abs(y-x)/math.Sqrt2
	half := r/math.Sqrt2 - round
	qu, qv := u-half, v-half
	return math.Hypot(max(qu, 0), max(qv, 0)) + min(max(qu, qv), 0) - round
}

// bit is a bit hanging from the shaft at top between left and right, rounded at the bottom.
func bit(x, y, left, right, top, bottom float64) bool {
	r := (right - left) / 2
	return x >= left && x <= right && y >= top && y <= bottom-r || math.Hypot(x-left-r, y-bottom+r) <= r
}

// drawBadge clears the badge's shape grown by 1.5 so that it stands apart from the key, fills the shape, then
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
