package svgicon

import (
	"fmt"
	"strings"

	"golang.org/x/image/math/fixed"
)

// This file defines the basic path structure

// Operation groups the different SVG commands
type Operation interface {
	// add itself on the driver `d`, after aplying the transform `M`
	drawTo(d Drawer, M Matrix2D)
}

type MoveTo fixed.Point26_6

type LineTo fixed.Point26_6

type QuadTo [2]fixed.Point26_6

type CubicTo [3]fixed.Point26_6

type Close struct{}

// starts a new path at the given point.
func (op MoveTo) drawTo(d Drawer, M Matrix2D) {
	d.Stop(false) // implicit close if currently in path.
	d.Start(M.trMove(op))
}

// draw a line
func (op LineTo) drawTo(d Drawer, M Matrix2D) {
	d.Line(M.trLine(op))
}

// draw a quadratic bezier curve
func (op QuadTo) drawTo(d Drawer, M Matrix2D) {
	b, c := M.trQuad(op)
	d.QuadBezier(b, c)
}

// draw a cubic bezier curve
func (op CubicTo) drawTo(d Drawer, M Matrix2D) {
	b, c, d_ := M.trCubic(op)
	d.CubeBezier(b, c, d_)
}

func (op Close) drawTo(d Drawer, _ Matrix2D) {
	d.Stop(true)
}

// Path describes a sequence of basic SVG operations, which should not be nil
// Higher-level shapes may be reduced to a path.
type Path []Operation

// ToSVGPath returns a string representation of the path
func (p Path) ToSVGPath() string {
	chunks := make([]string, len(p))
	for i, op := range p {
		switch op := op.(type) {
		case MoveTo:
			chunks[i] = fmt.Sprintf("M%4.3f,%4.3f", float32(op.X)/64, float32(op.Y)/64)
		case LineTo:
			chunks[i] = fmt.Sprintf("L%4.3f,%4.3f", float32(op.X)/64, float32(op.Y)/64)
		case QuadTo:
			chunks[i] = fmt.Sprintf("Q%4.3f,%4.3f,%4.3f,%4.3f", float32(op[0].X)/64, float32(op[0].Y)/64,
				float32(op[1].X)/64, float32(op[1].Y)/64)
		case CubicTo:
			chunks[i] = "C" + fmt.Sprintf("C%4.3f,%4.3f,%4.3f,%4.3f,%4.3f,%4.3f", float32(op[0].X)/64, float32(op[0].Y)/64,
				float32(op[1].X)/64, float32(op[1].Y)/64, float32(op[2].X)/64, float32(op[2].Y)/64)
		case Close:
			chunks[i] = "Z"
		}
	}
	return strings.Join(chunks, " ")
}

// String returns a readable representation of a Path.
func (p Path) String() string {
	return p.ToSVGPath()
}

// Clear zeros the path slice
func (p *Path) Clear() {
	*p = (*p)[:0]
}

// Start starts a new curve at the given point.
func (p *Path) Start(a fixed.Point26_6) {
	*p = append(*p, MoveTo{a.X, a.Y})
}

// Line adds a linear segment to the current curve.
func (p *Path) Line(b fixed.Point26_6) {
	*p = append(*p, LineTo{b.X, b.Y})
}

// QuadBezier adds a quadratic segment to the current curve.
func (p *Path) QuadBezier(b, c fixed.Point26_6) {
	*p = append(*p, QuadTo{b, c})
}

// CubeBezier adds a cubic segment to the current curve.
func (p *Path) CubeBezier(b, c, d fixed.Point26_6) {
	*p = append(*p, CubicTo{b, c, d})
}

// Stop joins the ends of the path
func (p *Path) Stop(closeLoop bool) {
	if closeLoop {
		*p = append(*p, Close{})
	}
}
