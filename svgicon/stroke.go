package svgicon

import (
	"math"

	"golang.org/x/image/math/fixed"
)

const (
	cubicsPerHalfCircle = 8 // Number of cubic beziers to approx half a circle

	// fixed point t paramaterization shift factor;
	// (2^this)/64 is the max length of t for fixed.Int26_6
	tStrokeShift = 14
)

// invert  returns the point inverted around the origin
func invert(v fixed.Point26_6) fixed.Point26_6 {
	return fixed.Point26_6{X: -v.X, Y: -v.Y}
}

// turnStarboard90 returns the vector 90 degrees starboard (right in direction heading)
func turnStarboard90(v fixed.Point26_6) fixed.Point26_6 {
	return fixed.Point26_6{X: -v.Y, Y: v.X}
}

// turnPort90 returns the vector 90 degrees port (left in direction heading)
func turnPort90(v fixed.Point26_6) fixed.Point26_6 {
	return fixed.Point26_6{X: v.Y, Y: -v.X}
}

// length is the distance from the origin of the point
func length(v fixed.Point26_6) fixed.Int26_6 {
	vx, vy := float64(v.X), float64(v.Y)
	return fixed.Int26_6(math.Sqrt(vx*vx + vy*vy))
}

// strokeArc strokes a circular arc by approximation with bezier curves
func strokeArc(p *matrixAdder, a, s1, s2 fixed.Point26_6, clockwise bool, trimStart,
	trimEnd fixed.Int26_6, firstPoint func(p fixed.Point26_6)) (ps1, ds1, ps2, ds2 fixed.Point26_6) {
	// Approximate the circular arc using a set of cubic bezier curves by the method of
	// L. Maisonobe, "Drawing an elliptical arc using polylines, quadratic
	// or cubic Bezier curves", 2003
	// https://www.spaceroots.org/documents/elllipse/elliptical-arc.pdf
	// The method was simplified for circles.
	theta1 := math.Atan2(float64(s1.Y-a.Y), float64(s1.X-a.X))
	theta2 := math.Atan2(float64(s2.Y-a.Y), float64(s2.X-a.X))
	if !clockwise {
		for theta1 < theta2 {
			theta1 += math.Pi * 2
		}
	} else {
		for theta2 < theta1 {
			theta2 += math.Pi * 2
		}
	}
	deltaTheta := theta2 - theta1
	if trimStart > 0 {
		ds := (deltaTheta * float64(trimStart)) / float64(1<<tStrokeShift)
		deltaTheta -= ds
		theta1 += ds
	}
	if trimEnd > 0 {
		ds := (deltaTheta * float64(trimEnd)) / float64(1<<tStrokeShift)
		deltaTheta -= ds
	}

	segs := int(math.Abs(deltaTheta)/(math.Pi/cubicsPerHalfCircle)) + 1
	dTheta := deltaTheta / float64(segs)
	tde := math.Tan(dTheta / 2)
	alpha := fixed.Int26_6(math.Sin(dTheta) * (math.Sqrt(4+3*tde*tde) - 1) * (64.0 / 3.0)) // Math is fun!
	r := float64(length(s1.Sub(a)))                                                        // Note r is *64
	ldp := fixed.Point26_6{X: -fixed.Int26_6(r * math.Sin(theta1)), Y: fixed.Int26_6(r * math.Cos(theta1))}
	ds1 = ldp
	ps1 = fixed.Point26_6{X: a.X + ldp.Y, Y: a.Y - ldp.X}
	firstPoint(ps1)
	s1 = ps1
	for i := 1; i <= segs; i++ {
		eta := theta1 + dTheta*float64(i)
		ds2 = fixed.Point26_6{X: -fixed.Int26_6(r * math.Sin(eta)), Y: fixed.Int26_6(r * math.Cos(eta))}
		ps2 = fixed.Point26_6{X: a.X + ds2.Y, Y: a.Y - ds2.X} // Using deriviative to calc new pt, because circle
		p1 := s1.Add(ldp.Mul(alpha))
		p2 := ps2.Sub(ds2.Mul(alpha))
		p.CubeBezier(p1, p2, ps2)
		s1, ldp = ps2, ds2
	}
	return
}
