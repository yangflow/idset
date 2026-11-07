// Package trayicon draws the tray icon, the product icon's folder with its keyhole, so that no image file has to be
// maintained by hand.
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
	// None: the folder alone.
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

// The template is 40×36, 36 being the 18 points of the macOS menu bar at 2x, and its folder is 28 tall, the 14
// points of the other icons in the menu bar.
const templateWidth, templateHeight, templateFolder = 40, 36, 28

// The folder and its keyholes are drawn on the 100 grid of design/icon.svg, where the folder spans x 10 to 90 and
// y 20 to 84.
const folderLeft, folderTop, folderWidth, folderHeight = 10, 20, 80, 64

// PNG returns the tray icon. A template icon is 40×36, black on transparent for macOS to tint to the menu bar, which
// scales it to 18 points tall; the folder is 14 points tall and centered. Otherwise it is 32×32 in a mid gray that
// stays visible on light and dark taskbars, since Windows and Linux show a square: the folder is as wide as the icon,
// or, with a badge, smaller at the upper left so that the badge stays clear of the keyhole. A badge sits at the
// lower right, 1.5 apart from the folder, in a template centered on the folder's corner at (80, 78) of the grid: in a
// template its "!" is cut out, elsewhere it is in Warning or Danger with a white "!".
func PNG(template bool, badge Badge) []byte {
	ink := color.NRGBA{R: 0x8a, G: 0x8a, B: 0x8a, A: 0xff}
	if template {
		ink = color.NRGBA{A: 0xff}
	}
	w, h, grid, bx, by := place(template, badge)
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	paint(img, func(x, y float64) bool { return inside(folder, grid, x, y) }, ink)
	paint(img, func(x, y float64) bool { gx, gy := grid(x, y); return keyholeCut.inside(gx, gy) }, color.NRGBA{})
	paint(img, func(x, y float64) bool { gx, gy := grid(x, y); return keyholeCore.inside(gx, gy) }, ink)
	// The badge is laid out for a 32×32 icon, centered on (24.5, 24.5); a template moves its center to (bx, by).
	drawBadge(img, template, ink, badge, func(x, y float64) (float64, float64) {
		return x - bx + badgeCenter, y - by + badgeCenter
	})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// badgeCenter is the center of the badge in its 32×32 layout, and badgeX, badgeY the point of the 100 grid that a
// template centers it on, the folder's lower right corner.
const badgeCenter, badgeX, badgeY = 24.5, 80, 78

// place returns the icon's size, its grid, which maps a point of the icon to the 100 grid of design/icon.svg, and
// the badge's center.
func place(template bool, badge Badge) (w, h int, grid func(x, y float64) (float64, float64), bx, by float64) {
	w, h = 32, 32
	s, ox, oy := 32.0/folderWidth, 0.0, (32-folderHeight*32.0/folderWidth)/2
	if badge != None {
		s, ox, oy = 0.35, 0, 1
	}
	bx, by = badgeCenter, badgeCenter
	if template {
		w, h = templateWidth, templateHeight
		s = float64(templateFolder) / folderHeight
		ox, oy = (templateWidth-folderWidth*s)/2, (templateHeight-templateFolder)/2
		bx, by = ox+(badgeX-folderLeft)*s, oy+(badgeY-folderTop)*s
	}
	return w, h, func(x, y float64) (float64, float64) { return (x-ox)/s + folderLeft, (y-oy)/s + folderTop }, bx, by
}

// folder is the outline of design/icon.svg's folder, its tab at the upper left, flattened to a polygon.
var folder = newOutline(10, 33).
	quad(10, 20, 23, 20).line(35, 20).quad(40, 20, 43, 24).line(45.5, 27.5).quad(47.5, 30, 51, 30).line(77, 30).
	quad(90, 30, 90, 43).line(90, 71).quad(90, 84, 77, 84).line(23, 84).quad(10, 84, 10, 71).points

// The keyhole cut through the folder, and the smaller mint one lit inside it.
var (
	keyholeCut  = keyhole{cx: 50, cy: 53, r: 11.5, top: 57, bottom: 75.5, neck: 13, foot: 17.5}
	keyholeCore = keyhole{cx: 50, cy: 53, r: 7, top: 57, bottom: 71, neck: 6.5, foot: 9.5}
)

// keyhole is a round head and a foot that widens from neck at top to foot at bottom, its bottom corners rounded by
// 2.2.
type keyhole struct{ cx, cy, r, top, bottom, neck, foot float64 }

func (k keyhole) inside(x, y float64) bool {
	if math.Hypot(x-k.cx, y-k.cy) <= k.r {
		return true
	}
	const c = 2.2
	l, r := k.cx-k.foot/2, k.cx+k.foot/2
	foot := newOutline(k.cx-k.neck/2, k.top).line(k.cx+k.neck/2, k.top).line(r, k.bottom-c).quad(r, k.bottom, r-c, k.bottom).
		line(l+c, k.bottom).quad(l, k.bottom, l, k.bottom-c).points
	return inPolygon([2]float64{x, y}, foot)
}

// outline builds a closed polygon from lines and quadratic curves, each curve cut into 12 segments.
type outline struct{ points [][2]float64 }

func newOutline(x, y float64) *outline { return &outline{points: [][2]float64{{x, y}}} }

func (o *outline) line(x, y float64) *outline {
	o.points = append(o.points, [2]float64{x, y})
	return o
}

func (o *outline) quad(cx, cy, x, y float64) *outline {
	p := o.points[len(o.points)-1]
	for i := 1; i <= 12; i++ {
		t := float64(i) / 12
		u := 1 - t
		o.points = append(o.points, [2]float64{u*u*p[0] + 2*u*t*cx + t*t*x, u*u*p[1] + 2*u*t*cy + t*t*y})
	}
	return o
}

// inside reports whether the pixel position (x, y), mapped to the grid, is inside the polygon.
func inside(poly [][2]float64, grid func(x, y float64) (float64, float64), x, y float64) bool {
	gx, gy := grid(x, y)
	return inPolygon([2]float64{gx, gy}, poly)
}

// inPolygon reports whether p is inside the closed polygon, by the even-odd rule.
func inPolygon(p [2]float64, poly [][2]float64) bool {
	in := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		a, b := poly[i], poly[j]
		if (a[1] > p[1]) != (b[1] > p[1]) && p[0] < (b[0]-a[0])*(p[1]-a[1])/(b[1]-a[1])+a[0] {
			in = !in
		}
	}
	return in
}

// drawBadge clears the badge's shape grown by 1.5 so that it stands apart from the folder, fills the shape, then
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
