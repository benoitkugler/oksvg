// Implements an abstract representation of
// svg paths, which can then be consumed
// by painting driver
package svgpath

import (
	"fmt"
	"strings"

	"golang.org/x/image/math/fixed"
)

// // Adder interface for types that can accumlate path commands
// type Adder interface {
// 	// Start starts a new curve at the given point.
// 	Start(a fixed.Point26_6)
// 	// Line adds a line segment to the path
// 	Line(b fixed.Point26_6)
// 	// QuadBezier adds a quadratic bezier curve to the path
// 	QuadBezier(b, c fixed.Point26_6)
// 	// CubeBezier adds a cubic bezier curve to the path
// 	CubeBezier(b, c, d fixed.Point26_6)
// 	// Closes the path to the start point if closeLoop is true
// 	Stop(closeLoop bool)
// }

type pathCommand uint8

// Human readable path constants
const (
	pathMoveTo pathCommand = iota
	pathLineTo
	pathQuadTo
	pathCubicTo
	pathClose
)

// Operation groups the different SVG commands
type Operation interface {
	command() pathCommand
}

type MoveTo fixed.Point26_6

type LineTo fixed.Point26_6

type QuadTo [2]fixed.Point26_6

type CubicTo [3]fixed.Point26_6

type Close struct{}

func (MoveTo) command() pathCommand  { return pathMoveTo }
func (LineTo) command() pathCommand  { return pathLineTo }
func (QuadTo) command() pathCommand  { return pathQuadTo }
func (CubicTo) command() pathCommand { return pathCubicTo }
func (Close) command() pathCommand   { return pathClose }

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

// // AddTo adds the Path p to q.
// func (p Path) AddTo(q Adder) {
// 	for _, op := range p {
// 		switch op := op.(type) {
// 		case MoveTo:
// 			q.Stop(false) // Fixes issues #1 by described by Djadala; implicit close if currently in path.
// 			q.Start(fixed.Point26_6(op))
// 		case LineTo:
// 			q.Line(fixed.Point26_6(op))
// 		case QuadTo:
// 			q.QuadBezier(op[0], op[1])
// 		case CubicTo:
// 			q.CubeBezier(op[0], op[1], op[2])
// 		case Close:
// 			q.Stop(true)
// 		}
// 	}
// 	q.Stop(false)
// }
