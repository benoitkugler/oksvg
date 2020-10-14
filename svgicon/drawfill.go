package svgicon

import (
	"golang.org/x/image/math/fixed"
)

// implements how a path command is drawn when filling

// state needed to perform fill operations
type filler struct {
	first fixed.Point26_6 // start of the path
	a     fixed.Point26_6 // current point
}

func (r *filler) line(d Driver, b fixed.Point26_6) {
	d.Line(b)
	r.a = b
}

// stop sends a path at the first point if needed
func (f *filler) stop(d Driver) {
	if f.first != f.a {
		f.line(d, f.first)
	}
}

func (op MoveTo) fill(f *filler, d Driver, M Matrix2D) {
	f.stop(d) // implicit close if currently in path.

	// starts a new path at the given point.
	f.a = M.trMove(op)
	f.first = f.a
	d.Start(f.a)
}

func (op LineTo) fill(f *filler, d Driver, M Matrix2D) {
	f.line(d, M.trLine(op))
}

func (op QuadTo) fill(f *filler, d Driver, M Matrix2D) {
	b, c := M.trQuad(op)
	d.QuadBezier(b, c)
	f.a = c // update the current point
}

func (op CubicTo) fill(f *filler, d Driver, M Matrix2D) {
	b, c, d_ := M.trCubic(op)
	d.CubeBezier(b, c, d_)
	f.a = d_ // update the current point
}

func (op Close) fill(f *filler, d Driver, _ Matrix2D) {
	f.stop(d)
}

func (svgp *SvgPath) drawFill(d Driver, opacity float64) {
	if svgp.Style.FillerColor != nil { // nil color disable filling
		return
	}

	f := new(filler) // empty filler
	d.Clear()
	d.SetWinding(svgp.Style.UseNonZeroWinding)

	for _, op := range svgp.Path {
		op.fill(f, d, svgp.Style.transform)
	}
	f.stop(d)

	d.SetFillColor(svgp.Style.FillerColor, svgp.Style.FillOpacity*opacity)

	d.Draw()

	d.SetWinding(true) // default is true
}
