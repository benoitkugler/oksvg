package svgpath

import (
	"math"

	"golang.org/x/image/math/fixed"
)

// This file implements the transformation from
// high level shapes to their path equivalent

// maxDx is the maximum radians a cubic splice is allowed to span
// in ellipse parametric when approximating an off-axis ellipse.
const maxDx float64 = math.Pi / 8

// toFixedP converts two floats to a fixed point.
func toFixedP(x, y float64) (p fixed.Point26_6) {
	p.X = fixed.Int26_6(x * 64)
	p.Y = fixed.Int26_6(y * 64)
	return
}

// addRect adds a rectangle of the indicated size, rotated
// around the center by rot degrees.
func (p *Path) addRect(minX, minY, maxX, maxY, rot float64) {
	rot *= math.Pi / 180
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	m := Identity.Translate(cx, cy).Rotate(rot).Translate(-cx, -cy)
	q := &matrixAdder{M: m, path: p}
	q.Start(toFixedP(minX, minY))
	q.Line(toFixedP(maxX, minY))
	q.Line(toFixedP(maxX, maxY))
	q.Line(toFixedP(minX, maxY))
	q.path.Stop(true)
}

// addRoundRect adds a rectangle of the indicated size, rotated
// around the center by rot degrees with rounded corners of radius
// rx in the x axis and ry in the y axis. gf specifes the shape of the
// filleting function.
func (p *Path) addRoundRect(minX, minY, maxX, maxY, rx, ry, rot float64) {
	if rx <= 0 || ry <= 0 {
		p.addRect(minX, minY, maxX, maxY, rot)
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

	q := &matrixAdder{M: m, path: p}

	q.Start(toFixedP(minX+rx, minY))
	q.Line(toFixedP(maxX-rx, minY))
	RoundGap(q, toFixedP(maxX-rx, minY+rx), toFixedP(0, -rx), toFixedP(rx, 0))
	q.Line(toFixedP(maxX, maxY-rx))
	RoundGap(q, toFixedP(maxX-rx, maxY-rx), toFixedP(rx, 0), toFixedP(0, rx))
	q.Line(toFixedP(minX+rx, maxY))
	RoundGap(q, toFixedP(minX+rx, maxY-rx), toFixedP(0, rx), toFixedP(-rx, 0))
	q.Line(toFixedP(minX, minY+rx))
	RoundGap(q, toFixedP(minX+rx, minY+rx), toFixedP(-rx, 0), toFixedP(0, -rx))
	q.path.Stop(true)
}

// addArc adds an arc to the adder p
func (p *Path) addArc(points []float64, cx, cy, px, py float64) (lx, ly float64) {
	rotX := points[2] * math.Pi / 180 // Convert degress to radians
	largeArc := points[3] != 0
	sweep := points[4] != 0
	startAngle := math.Atan2(py-cy, px-cx) - rotX
	endAngle := math.Atan2(points[6]-cy, points[5]-cx) - rotX
	deltaTheta := endAngle - startAngle
	arcBig := math.Abs(deltaTheta) > math.Pi

	// Approximate ellipse using cubic bezeir splines
	etaStart := math.Atan2(math.Sin(startAngle)/points[1], math.Cos(startAngle)/points[0])
	etaEnd := math.Atan2(math.Sin(endAngle)/points[1], math.Cos(endAngle)/points[0])
	deltaEta := etaEnd - etaStart
	if (arcBig && !largeArc) || (!arcBig && largeArc) { // Go has no boolean XOR
		if deltaEta < 0 {
			deltaEta += math.Pi * 2
		} else {
			deltaEta -= math.Pi * 2
		}
	}
	// This check might be needed if the center point of the elipse is
	// at the midpoint of the start and end lines.
	if deltaEta < 0 && sweep {
		deltaEta += math.Pi * 2
	} else if deltaEta >= 0 && !sweep {
		deltaEta -= math.Pi * 2
	}

	// Round up to determine number of cubic splines to approximate bezier curve
	segs := int(math.Abs(deltaEta)/maxDx) + 1
	dEta := deltaEta / float64(segs) // span of each segment
	// Approximate the ellipse using a set of cubic bezier curves by the method of
	// L. Maisonobe, "Drawing an elliptical arc using polylines, quadratic
	// or cubic Bezier curves", 2003
	// https://www.spaceroots.org/documents/elllipse/elliptical-arc.pdf
	tde := math.Tan(dEta / 2)
	alpha := math.Sin(dEta) * (math.Sqrt(4+3*tde*tde) - 1) / 3 // Math is fun!
	lx, ly = px, py
	sinTheta, cosTheta := math.Sin(rotX), math.Cos(rotX)
	ldx, ldy := ellipsePrime(points[0], points[1], sinTheta, cosTheta, etaStart, cx, cy)
	for i := 1; i <= segs; i++ {
		eta := etaStart + dEta*float64(i)
		var px, py float64
		if i == segs {
			px, py = points[5], points[6] // Just makes the end point exact; no roundoff error
		} else {
			px, py = ellipsePointAt(points[0], points[1], sinTheta, cosTheta, eta, cx, cy)
		}
		dx, dy := ellipsePrime(points[0], points[1], sinTheta, cosTheta, eta, cx, cy)
		p.CubeBezier(toFixedP(lx+alpha*ldx, ly+alpha*ldy),
			toFixedP(px-alpha*dx, py-alpha*dy), toFixedP(px, py))
		lx, ly, ldx, ldy = px, py, dx, dy
	}
	return lx, ly
}

// ellipsePrime gives tangent vectors for parameterized elipse; a, b, radii, eta parameter, center cx, cy
func ellipsePrime(a, b, sinTheta, cosTheta, eta, cx, cy float64) (px, py float64) {
	bCosEta := b * math.Cos(eta)
	aSinEta := a * math.Sin(eta)
	px = -aSinEta*cosTheta - bCosEta*sinTheta
	py = -aSinEta*sinTheta + bCosEta*cosTheta
	return
}

// ellipsePointAt gives points for parameterized elipse; a, b, radii, eta parameter, center cx, cy
func ellipsePointAt(a, b, sinTheta, cosTheta, eta, cx, cy float64) (px, py float64) {
	aCosEta := a * math.Cos(eta)
	bSinEta := b * math.Sin(eta)
	px = cx + aCosEta*cosTheta - bSinEta*sinTheta
	py = cy + aCosEta*sinTheta + bSinEta*cosTheta
	return
}

// findEllipseCenter locates the center of the Ellipse if it exists. If it does not exist,
// the radius values will be increased minimally for a solution to be possible
// while preserving the ra to rb ratio.  ra and rb arguments are pointers that can be
// checked after the call to see if the values changed. This method uses coordinate transformations
// to reduce the problem to finding the center of a circle that includes the origin
// and an arbitrary point. The center of the circle is then transformed
// back to the original coordinates and returned.
func findEllipseCenter(ra, rb *float64, rotX, startX, startY, endX, endY float64, sweep, smallArc bool) (cx, cy float64) {
	cos, sin := math.Cos(rotX), math.Sin(rotX)

	// Move origin to start point
	nx, ny := endX-startX, endY-startY

	// Rotate ellipse x-axis to coordinate x-axis
	nx, ny = nx*cos+ny*sin, -nx*sin+ny*cos
	// Scale X dimension so that ra = rb
	nx *= *rb / *ra // Now the ellipse is a circle radius rb; therefore foci and center coincide

	midX, midY := nx/2, ny/2
	midlenSq := midX*midX + midY*midY

	var hr float64
	if *rb**rb < midlenSq {
		// Requested ellipse does not exist; scale ra, rb to fit. Length of
		// span is greater than max width of ellipse, must scale *ra, *rb
		nrb := math.Sqrt(midlenSq)
		if *ra == *rb {
			*ra = nrb // prevents roundoff
		} else {
			*ra = *ra * nrb / *rb
		}
		*rb = nrb
	} else {
		hr = math.Sqrt(*rb**rb-midlenSq) / math.Sqrt(midlenSq)
	}
	// Notice that if hr is zero, both answers are the same.
	if (sweep && smallArc) || (!sweep && !smallArc) {
		cx = midX + midY*hr
		cy = midY - midX*hr
	} else {
		cx = midX - midY*hr
		cy = midY + midX*hr
	}

	// reverse scale
	cx *= *ra / *rb
	//Reverse rotate and translate back to original coordinates
	return cx*cos - cy*sin + startX, cx*sin + cy*cos + startY
}
