package svgpath

import (
	"math"

	"golang.org/x/image/math/fixed"
)

// ToFixedP converts two floats to a fixed point.
func ToFixedP(x, y float64) (p fixed.Point26_6) {
	p.X = fixed.Int26_6(x * 64)
	p.Y = fixed.Int26_6(y * 64)
	return
}

// AddRect adds a rectangle of the indicated size, rotated
// around the center by rot degrees.
func AddRect(minX, minY, maxX, maxY, rot float64, p Adder) {
	rot *= math.Pi / 180
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	m := Identity.Translate(cx, cy).Rotate(rot).Translate(-cx, -cy)
	q := &MatrixAdder{M: m, Adder: p}
	q.Start(ToFixedP(minX, minY))
	q.Line(ToFixedP(maxX, minY))
	q.Line(ToFixedP(maxX, maxY))
	q.Line(ToFixedP(minX, maxY))
	q.Stop(true)
}

// AddRoundRect adds a rectangle of the indicated size, rotated
// around the center by rot degrees with rounded corners of radius
// rx in the x axis and ry in the y axis. gf specifes the shape of the
// filleting function.
func AddRoundRect(minX, minY, maxX, maxY, rx, ry, rot float64, p Adder) {
	if rx <= 0 || ry <= 0 {
		AddRect(minX, minY, maxX, maxY, rot, p)
		return
	}
	rot *= math.Pi / 180

	w := maxX - minX
	if w < rx*2 {
		rx = w / 2
	}
	h := maxY - minY
	if h < ry*2 {
		ry = h / 2
	}
	stretch := rx / ry
	midY := minY + h/2
	m := Identity.Translate(minX+w/2, midY).Rotate(rot).Scale(1, 1/stretch).Translate(-minX-w/2, -minY-h/2)
	maxY = midY + h/2*stretch
	minY = midY - h/2*stretch

	q := &MatrixAdder{M: m, Adder: p}

	q.Start(ToFixedP(minX+rx, minY))
	q.Line(ToFixedP(maxX-rx, minY))
	gf(q, ToFixedP(maxX-rx, minY+rx), ToFixedP(0, -rx), ToFixedP(rx, 0))
	q.Line(ToFixedP(maxX, maxY-rx))
	gf(q, ToFixedP(maxX-rx, maxY-rx), ToFixedP(rx, 0), ToFixedP(0, rx))
	q.Line(ToFixedP(minX+rx, maxY))
	gf(q, ToFixedP(minX+rx, maxY-rx), ToFixedP(0, rx), ToFixedP(-rx, 0))
	q.Line(ToFixedP(minX, minY+rx))
	gf(q, ToFixedP(minX+rx, minY+rx), ToFixedP(-rx, 0), ToFixedP(0, -rx))
	q.Stop(true)
}
