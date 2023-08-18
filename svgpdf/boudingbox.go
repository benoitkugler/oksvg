package svgpdf

import (
	"math"

	"github.com/benoitkugler/pdf/model"
	"golang.org/x/image/math/fixed"
)

// compute the bouding box of a path, needed when using gradient with objectBoudingBox

type line [2]fixed.Point26_6

func (l line) criticalPoints() (tX, tY []model.Fl) {
	return nil, nil
}

func (l line) evaluateCurve(t model.Fl) (x, y model.Fl) {
	p0x, p0y := fixedTof(l[0])
	p1x, p1y := fixedTof(l[1])
	return bezierLine(p0x, p1x, t), bezierLine(p0y, p1y, t)
}

func bezierLine(p0, p1, t model.Fl) model.Fl {
	return (p1-p0)*t + p0
}

type quadBezier [3]fixed.Point26_6

// quadratic polinomial
// x = At^2 + Bt + C
// where
// A = p0 + p2 - 2p1
// B = 2(p1 - p0)
// C = p0
func bezierQuad(p0, p1, p2, t model.Fl) model.Fl {
	return (p0+p2-2*p1)*t*t + 2*(p1-p0)*t + p0
}

// derivative as at + b where a,b :
func quadraticDerivative(p0, p1, p2 model.Fl) (a, b model.Fl) {
	return 2 * (p2 - p1 - (p1 - p0)), 2 * (p1 - p0)
}

// handle the case where a = 0
func linearRoots(a, b model.Fl) []model.Fl {
	if a == 0 {
		return nil
	}
	return []model.Fl{-b / a}
}

func (cu quadBezier) criticalPoints() (tX, tY []model.Fl) {
	p0x, p0y := fixedTof(cu[0])
	p1x, p1y := fixedTof(cu[1])
	p2x, p2y := fixedTof(cu[2])

	aX, bX := quadraticDerivative(p0x, p1x, p2x)
	aY, bY := quadraticDerivative(p0y, p1y, p2y)

	return linearRoots(aX, bX), linearRoots(aY, bY)
}

func (cu quadBezier) evaluateCurve(t model.Fl) (x, y model.Fl) {
	p0x, p0y := fixedTof(cu[0])
	p1x, p1y := fixedTof(cu[1])
	p2x, p2y := fixedTof(cu[2])
	return bezierQuad(p0x, p1x, p2x, t), bezierQuad(p0y, p1y, p2y, t)
}

type cubicBezier [4]fixed.Point26_6

func (cu cubicBezier) criticalPoints() (tX, tY []model.Fl) {
	p1x, p1y := fixedTof(cu[0])
	c1x, c1y := fixedTof(cu[1])
	c2x, c2y := fixedTof(cu[2])
	p2x, p2y := fixedTof(cu[3])

	aX, bX, cX := cubicDerivative(p1x, c1x, c2x, p2x)
	aY, bY, cY := cubicDerivative(p1y, c1y, c2y, p2y)

	return quadraticRoots(aX, bX, cX), quadraticRoots(aY, bY, cY)
}

func (cu cubicBezier) evaluateCurve(t model.Fl) (x, y model.Fl) {
	p0x, p0y := fixedTof(cu[0])
	p1x, p1y := fixedTof(cu[1])
	p2x, p2y := fixedTof(cu[2])
	p3x, p3y := fixedTof(cu[3])
	return bezierSpline(p0x, p1x, p2x, p3x, t), bezierSpline(p0y, p1y, p2y, p3y, t)
}

// cubic polinomial
// x = At^3 + Bt^2 + Ct + D
// where A,B,C,D:
// A = p3 -3 * p2 + 3 * p1 - p0
// B = 3 * p2 - 6 * p1 +3 * p0
// C = 3 * p1 - 3 * p0
// D = p0
func bezierSpline(p0, p1, p2, p3, t model.Fl) model.Fl {
	return (p3-3*p2+3*p1-p0)*t*t*t +
		(3*p2-6*p1+3*p0)*t*t +
		(3*p1-3*p0)*t +
		(p0)
}

// We would like to know the values of t where X = 0
// X  = (p3-3*p2+3*p1-p0)t^3 + (3*p2-6*p1+3*p0)t^2 + (3*p1-3*p0)t + (p0)
// Derivative :
// X' = 3(p3-3*p2+3*p1-p0)t^(3-1) + 2(6*p2-12*p1+6*p0)t^(2-1) + 1(3*p1-3*p0)t^(1-1)
// simplified:
// X' = (3*p3-9*p2+9*p1-3*p0)t^2 + (6*p2-12*p1+6*p0)t + (3*p1-3*p0)
// taken as aX^2 + bX + c  a,b and c are:
func cubicDerivative(p0, p1, p2, p3 model.Fl) (a, b, c model.Fl) {
	return 3*p3 - 9*p2 + 9*p1 - 3*p0, 6*p2 - 12*p1 + 6*p0, 3*p1 - 3*p0
}

// b^2 - 4ac = Determinant
func determinant(a, b, c model.Fl) model.Fl { return b*b - 4*a*c }

func solve(a, b, c model.Fl, s bool) model.Fl {
	var sign model.Fl = 1.
	if !s {
		sign = -1.
	}
	return (-b + (model.Fl(math.Sqrt(float64((b*b)-(4*a*c)))) * sign)) / (2 * a)
}

func quadraticRoots(a, b, c model.Fl) []model.Fl {
	d := determinant(a, b, c)
	if d < 0 {
		return nil
	}

	if a == 0 {
		// aX^2 + bX + c well then then this is a simple line
		// x= -c / b
		return []model.Fl{-c / b}
	}

	if d == 0 {
		return []model.Fl{solve(a, b, c, true)}
	} else {
		return []model.Fl{
			solve(a, b, c, true),
			solve(a, b, c, false),
		}
	}
}

type bezier interface {
	// compute the t zeroing the derivative
	criticalPoints() (tX, tY []model.Fl)
	// compute the point a time t
	evaluateCurve(t model.Fl) (x, y model.Fl)
}

func computeBoundingBox(curve bezier) fixed.Rectangle26_6 {
	resX, resY := curve.criticalPoints()

	// draw min and max
	var bbox [][2]model.Fl

	// add begin and end point
	for _, t := range append(append(resX, 0, 1), resY...) {
		// filter invalid value
		if !(0 <= t && t <= 1) {
			continue
		}
		x, y := curve.evaluateCurve(t)

		bbox = append(bbox, [2]model.Fl{x, y})
	}

	minX := math.Inf(1)
	minY := math.Inf(1)
	maxX := math.Inf(-1)
	maxY := math.Inf(-1)

	for _, e := range bbox {
		minX = math.Min(float64(e[0]), minX)
		minY = math.Min(float64(e[1]), minY)
		maxX = math.Max(float64(e[0]), maxX)
		maxY = math.Max(float64(e[1]), maxY)
	}
	return fixed.Rectangle26_6{Min: fToFixed(minX, minY), Max: fToFixed(maxX, maxY)}
}

// BoundingBox stores the current bounding box
// and exposes method to update it
type BoundingBox struct {
	BBox fixed.Rectangle26_6
	a    fixed.Point26_6 // current point, used to compute the next boundingBox
}

func (p *BoundingBox) Start(a fixed.Point26_6) {
	p.a = a
	p.BBox = fixed.Rectangle26_6{Min: a, Max: a} // degenerate case
}

func (p *BoundingBox) Line(b fixed.Point26_6) {
	p.BBox = p.BBox.Union(computeBoundingBox(line{p.a, b}))
	p.a = b
}

func (p *BoundingBox) QuadBezier(b fixed.Point26_6, c fixed.Point26_6) {
	p.BBox = p.BBox.Union(computeBoundingBox(quadBezier{p.a, b, c}))
	p.a = c
}

func (p *BoundingBox) CubeBezier(b fixed.Point26_6, c fixed.Point26_6, d fixed.Point26_6) {
	p.BBox = p.BBox.Union(computeBoundingBox(cubicBezier{p.a, b, c, d}))
	p.a = d
}
