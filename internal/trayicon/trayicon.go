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

// The key is drawn in units of a 54×36 box, 36 being the 18 points of the macOS menu bar at 2x.
const keyWidth, keyHeight = 54, 36

// PNG returns the tray icon. A template icon is the key at full size, 54×36, black on transparent for macOS to
// tint to the menu bar, which scales it to 18 points tall. Otherwise it is 32×32, the key scaled to the width and
// centered, in a mid gray that stays visible on light and dark taskbars, since Windows and Linux show a square. A
// badge sits at the lower right, 1.5 apart from the key: in a template its "!" is cut out, elsewhere it is in
// Warning or Danger with a white "!".
func PNG(template bool, badge Badge) []byte {
	ink := color.NRGBA{R: 0x8a, G: 0x8a, B: 0x8a, A: 0xff}
	w, h := 32, 32
	k, dy := float64(w)/keyWidth, (32-keyHeight*32.0/keyWidth)/2
	if template {
		ink = color.NRGBA{A: 0xff}
		w, h, k, dy = keyWidth, keyHeight, 1, 0
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	paint(img, func(x, y float64) bool { return inKey(x/k, (y-dy)/k) }, ink)
	paint(img, func(x, y float64) bool { return inBranch(x/k, (y-dy)/k) }, color.NRGBA{})
	// The badge is laid out for a 32×32 icon; a template scales it to the height and moves it to the right edge.
	bk := float64(h) / 32
	drawBadge(img, template, ink, badge, func(x, y float64) (float64, float64) {
		return (x - float64(w) + 32*bk) / bk, y / bk
	})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// inKey is the product icon's key, solid and pointing right: a rounded diamond bow centered at (18, 18) whose
// corners reach the top and the bottom, a shaft 9.2 thick from the bow to 53, and two bits under the shaft.
func inKey(x, y float64) bool {
	const cy = keyHeight / 2
	return diamond(x-18, y-cy, 17.5, 3.5) <= 0 || roundRect(x, y, 28, cy-4.6, 53, cy+4.6, 2.8) ||
		roundRect(x, y, 39, cy, 44, cy+10.6, 1.6) || roundRect(x, y, 46.5, cy, 51.5, cy+8.6, 1.6)
}

// inBranch is the Git branch cut out of the bow: a trunk with a node at each end, and a curve from it to a third
// node at the upper right.
func inBranch(x, y float64) bool {
	const cy, s, half, node = keyHeight / 2, 8.0, 1.3, 2.5
	trunk, tip := [2]float64{14.2, cy}, [2]float64{20.6, cy - s*0.45}
	p := [2]float64{x, y}
	if toSegment(p, [2]float64{trunk[0], cy - s}, [2]float64{trunk[0], cy + s}) <= half {
		return true
	}
	// The curve is a quadratic from the trunk to the tip, bent towards the upper right.
	q := func(t float64) [2]float64 {
		u := 1 - t
		return [2]float64{u*u*trunk[0] + 2*u*t*tip[0] + t*t*tip[0], u*u*(cy+s*0.35) + 2*u*t*(cy-s*0.25) + t*t*tip[1]}
	}
	for i := range 24 {
		if toSegment(p, q(float64(i)/24), q(float64(i+1)/24)) <= half {
			return true
		}
	}
	return math.Hypot(x-trunk[0], y-cy-s) <= node || math.Hypot(x-trunk[0], y-cy+s) <= node ||
		math.Hypot(x-tip[0], y-tip[1]) <= node
}

// diamond is the signed distance from (x, y) to a diamond centered at the origin, its corners r from the center
// before they are rounded by round: negative inside.
func diamond(x, y, r, round float64) float64 {
	u, v := math.Abs(x+y)/math.Sqrt2, math.Abs(y-x)/math.Sqrt2
	half := r/math.Sqrt2 - round
	qu, qv := u-half, v-half
	return math.Hypot(max(qu, 0), max(qv, 0)) + min(max(qu, qv), 0) - round
}

// roundRect reports whether (x, y) is inside the rectangle from (left, top) to (right, bottom) with corners rounded
// by r.
func roundRect(x, y, left, top, right, bottom, r float64) bool {
	if x < left || x > right || y < top || y > bottom {
		return false
	}
	cx, cy := max(left+r, min(right-r, x)), max(top+r, min(bottom-r, y))
	return math.Hypot(x-cx, y-cy) <= r
}

// drawBadge clears the badge's shape grown by 1.5 so that it stands apart from the key, fills the shape, then
// draws or cuts out the "!";
// at maps a pixel position to the 32×32 layout.
func drawBadge(img *image.NRGBA, template bool, ink color.NRGBA, badge Badge, at func(x, y float64) (float64, float64)) {
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
		x, y = at(x, y)
		return toSegment([2]float64{x, y}, [2]float64{24.5, top}, [2]float64{24.5, top + 3.6}) <= 0.9 ||
			toSegment([2]float64{x, y}, [2]float64{24.5, dot}, [2]float64{24.5, dot + 0.1}) <= 0.9
	}
	paint(img, func(x, y float64) bool { x, y = at(x, y); return shape(x, y, 1.5) }, color.NRGBA{})
	paint(img, func(x, y float64) bool { x, y = at(x, y); return shape(x, y, 0) }, fill)
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
