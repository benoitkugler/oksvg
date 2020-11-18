// Implements a PDF backend to render SVG images,
// by wrapping github.com/jung-kurt/gofpdf.
package svgpdf

import (
	"github.com/inkeliz/oksvg/svgicon"
	"github.com/jung-kurt/gofpdf"
	"golang.org/x/image/math/fixed"
)

// assert interface conformance
// var (
// 	_ svgicon.Driver  = Renderer{}
// 	_ svgicon.Filler  = filler{}
// 	_ svgicon.Stroker = stroker{}
// 	_ svgicon.Stroker = patherStroker{}
// )

type Renderer struct {
	pdf *gofpdf.Fpdf
}

// implements the common path commands,
// shared by the filler and the stroker
type pather struct {
	pdf         *gofpdf.Fpdf
	a           fixed.Point26_6     // current point, used to compute boundingBox
	boundingBox fixed.Rectangle26_6 // bouding box for the current path
}

// implements the filling operation
type filler struct {
	pather
	useNonZeroWinding bool
}

// implements the stroking operation, while
// also writing the path
type patherStroker struct {
	pather
}

// only stroke the current path, doesnt add point to it
type stroker struct{}

// NewRenderer return a renderer which will
// write to the given `pdf`.
func NewRenderer(pdf *gofpdf.Fpdf) Renderer {
	return Renderer{pdf: pdf}
}

func fixedTof(a fixed.Point26_6) (float64, float64) {
	return float64(a.X) / 64, float64(a.Y) / 64
}

func fToFixed(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
}

func (p *pather) Clear() {
	p.boundingBox = fixed.Rectangle26_6{}
	p.a = fixed.Point26_6{}
}

func (p *pather) Start(a fixed.Point26_6) {
	p.pdf.MoveTo(fixedTof(a))
	p.a = a
	p.boundingBox = fixed.Rectangle26_6{Min: a, Max: a} // degenerate case
}

func (p *pather) Line(b fixed.Point26_6) {
	p.pdf.LineTo(fixedTof(b))
	p.boundingBox = p.boundingBox.Union(computeBoundingBox(line{p.a, b}))
	p.a = b
}

func (p *pather) QuadBezier(b fixed.Point26_6, c fixed.Point26_6) {
	cx, cy := fixedTof(b)
	x, y := fixedTof(c)
	p.pdf.CurveTo(cx, cy, x, y)
	p.boundingBox = p.boundingBox.Union(computeBoundingBox(quadBezier{p.a, b, c}))
	p.a = c
}

func (p *pather) CubeBezier(b fixed.Point26_6, c fixed.Point26_6, d fixed.Point26_6) {
	cx0, cy0 := fixedTof(b)
	cx1, cy1 := fixedTof(c)
	x, y := fixedTof(d)
	p.pdf.CurveBezierCubicTo(cx0, cy0, cx1, cy1, x, y)
	p.boundingBox = p.boundingBox.Union(computeBoundingBox(cubicBezier{p.a, b, c, d}))
	p.a = d
}

func (p *pather) Stop(closeLoop bool) {
	if closeLoop {
		p.pdf.ClosePath()
	}
}

// TODO: support gradient
func (f filler) SetColor(color svgicon.Pattern, opacity float64) {
	switch color := color.(type) {
	case svgicon.PlainColor:
		f.pdf.SetFillColor(int(color.R), int(color.G), int(color.B))
		opacity *= float64(color.A) / 255.
	}
	f.pdf.SetAlpha(opacity, "")
}

func (f filler) Draw() {
	styleStr := "f*"
	if f.useNonZeroWinding {
		styleStr = "f"
	}
	f.pdf.DrawPath(styleStr)
}

func (f *filler) SetWinding(useNonZeroWinding bool) {
	f.useNonZeroWinding = useNonZeroWinding
}
