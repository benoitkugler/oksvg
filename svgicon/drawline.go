package svgicon

import (
	"image/color"

	"github.com/srwiley/rasterx"
	"golang.org/x/image/math/fixed"
)

// implements how a path command is drawn when stroking

type stroker struct{}

func (op MoveTo) stroke(s *stroker, d Driver, M Matrix2D) {
	f.stop(d) // implicit close if currently in path.

	// starts a new path at the given point.
	f.a = M.trMove(op)
	f.first = f.a
	d.Start(f.a)
}

func (op LineTo) stroke(s *stroker, d Driver, M Matrix2D) {
	f.line(d, M.trLine(op))
}

func (op QuadTo) stroke(s *stroker, d Driver, M Matrix2D) {
	b, c := M.trQuad(op)
	d.QuadBezier(b, c)
	f.a = c // update the current point
}

func (op CubicTo) stroke(s *stroker, d Driver, M Matrix2D) {
	b, c, d_ := M.trCubic(op)
	d.CubeBezier(b, c, d_)
	f.a = d_ // update the current point
}

func (op Close) stroke(s *stroker, d Driver, _ Matrix2D) {
	f.stop(d)
}

func (svgp *SvgPath) drawLine(d Driver, opacity float64) {
	r.Clear()
	svgp.mAdder.Adder = r
	lineGap := svgp.LineGap
	if lineGap == nil {
		lineGap = DefaultStyle.LineGap
	}
	lineCap := svgp.LineCap
	if lineCap == nil {
		lineCap = DefaultStyle.LineCap
	}
	leadLineCap := lineCap
	if svgp.LeadLineCap != nil {
		leadLineCap = svgp.LeadLineCap
	}
	r.SetStroke(fixed.Int26_6(svgp.LineWidth*64),
		fixed.Int26_6(svgp.MiterLimit*64), leadLineCap, lineCap,
		lineGap, svgp.LineJoin, svgp.Dash, svgp.DashOffset)
	svgp.Path.AddTo(&svgp.mAdder)
	switch LinerColor := svgp.LinerColor.(type) {
	case color.Color:
		r.SetColor(rasterx.ApplyOpacity(LinerColor, svgp.LineOpacity*opacity))
	case Gradient:
		if LinerColor.Units == rasterx.ObjectBoundingBox {
			fRect := r.Scanner.GetPathExtent()
			mnx, mny := float64(fRect.Min.X)/64, float64(fRect.Min.Y)/64
			mxx, mxy := float64(fRect.Max.X)/64, float64(fRect.Max.Y)/64
			LinerColor.Bounds.X, LinerColor.Bounds.Y = mnx, mny
			LinerColor.Bounds.W, LinerColor.Bounds.H = mxx-mnx, mxy-mny
		}
		r.SetColor(LinerColor.GetColorFunction(svgp.LineOpacity * opacity))
	}
	r.Draw()
}
