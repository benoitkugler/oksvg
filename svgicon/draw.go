package svgicon

import (
	"golang.org/x/image/math/fixed"
)

// Given a parsed SVG document, implements how to
// draw it on screen.
// This requires a driver implementing the actual draw operations,
// such as a rasterizer to output .png images or a pdf writer.

// Driver knows how to do the actual draw operations
// but doesn't need any SVG kwowledge
type Driver interface {
	// Clear must reset the internal state (used before starting a new path painting)
	Clear()

	// Decide to use or not the NonZeroWinding rule.
	SetWinding(useNonZeroWinding bool)

	// Set the filling color (plain color or gradient)
	SetFillColor(color Pattern, opacity float64)

	// Start starts a new path at the given point.
	Start(a fixed.Point26_6)

	// Line Adds a line for the current point to `b`
	Line(b fixed.Point26_6)

	// QuadBezier adds a quadratic bezier curve to the path
	QuadBezier(b, c fixed.Point26_6)

	// CubeBezier adds a cubic bezier curve to the path
	CubeBezier(b, c, d fixed.Point26_6)

	// Draw draws the accumulated path using the current fill and stroke color
	Draw()
}

// SetTarget sets the Transform matrix to draw within the bounds of the rectangle arguments
func (s *SvgIcon) SetTarget(x, y, w, h float64) {
	scaleW := w / s.ViewBox.W
	scaleH := h / s.ViewBox.H
	s.Transform = Identity.Translate(x-s.ViewBox.X, y-s.ViewBox.Y).Scale(scaleW, scaleH)
}

// Draw the compiled SVG icon into the driver `d`.
// All elements should be contained by the Bounds rectangle of the SvgIcon.
func (s *SvgIcon) Draw(d Driver, opacity float64) {
	for _, svgp := range s.SVGPaths {
		svgp.drawTransformed(d, opacity, s.Transform)
	}
}

// drawTransformed draws the compiled SvgPath into the driver while applying transform t.
func (svgp *SvgPath) drawTransformed(d Driver, opacity float64, t Matrix2D) {
	m := svgp.Style.transform
	svgp.Style.transform = t.Mult(m)
	defer func() { svgp.Style.transform = m }() // Restore untransformed matrix

	svgp.drawFill(d, opacity)

	if svgp.Style.LinerColor != nil { // // nil color disable lining
		svgp.drawLine(d, opacity)
	}
}
