// Alternative implementation of PDF rendering (experimental)
package alt

import (
	"io"

	"github.com/benoitkugler/oksvg/svgicon"
	"github.com/benoitkugler/oksvg/svgpdf"
	"github.com/benoitkugler/pdf/contentstream"
	"github.com/benoitkugler/pdf/model"
	"golang.org/x/image/math/fixed"
)

// assert interface conformance
var (
	_ svgicon.Driver  = Renderer{}
	_ svgicon.Filler  = (*filler)(nil)
	_ svgicon.Stroker = (*stroker)(nil)
	_ svgicon.Stroker = (*patherStroker)(nil)
)

type Renderer struct {
	pdf                 *contentstream.Appearance
	fillOpacityStates   map[float64]*model.GraphicState
	strokeOpacityStates map[float64]*model.GraphicState
}

// implements the common path commands,
// shared by the filler and the stroker
type pather struct {
	pdf         *contentstream.Appearance
	boundingBox svgpdf.BoundingBox
}

// implements the filling operation
type filler struct {
	pather
	useNonZeroWinding bool
	fillOpacityStates map[float64]*model.GraphicState
}

// implements the stroking operation, while
// also writing the path
type patherStroker struct {
	pather
	strokeOpacityStates map[float64]*model.GraphicState
}

// only stroke the current path, established by
// the filler
type stroker struct {
	patherStroker
}

// RenderSVGIconToPDF reads the given icon and renders it
// into the given file.
func RenderSVGIconToPDF(icon io.Reader, pdfName string) error {
	parsedIcon, err := svgicon.ReadIconStream(icon, svgicon.WarnErrorMode)
	if err != nil {
		return err
	}
	pdf := contentstream.NewAppearance(595.28, 841.89)
	// pdf.TransformBegin()
	// pdf.TransformScale(10000/parsedIcon.ViewBox.W, 10000/parsedIcon.ViewBox.H, 0, 0)
	renderer := NewRenderer(&pdf)
	pdf.Ops(
		contentstream.OpSave{},
		contentstream.OpConcat{Matrix: model.Matrix{1, 0, 0, -1, 0, 841.89}},
	)
	parsedIcon.Draw(renderer, 1.0)
	pdf.Ops(contentstream.OpRestore{})

	var doc model.Document
	doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids, pdf.ToPageObject(true))
	return doc.WriteFile(pdfName, nil)
}

// NewRenderer return a renderer which will
// write to the given `pdf`.
func NewRenderer(cs *contentstream.Appearance) Renderer {
	return Renderer{pdf: cs,
		fillOpacityStates:   make(map[float64]*model.GraphicState),
		strokeOpacityStates: make(map[float64]*model.GraphicState),
	}
}

func (r Renderer) SetupDrawers(willFill, willDraw bool) (f svgicon.Filler, s svgicon.Stroker) {
	if willFill { //
		f = &filler{pather: pather{pdf: r.pdf}, fillOpacityStates: r.fillOpacityStates}
		if willDraw { // dont write the same path twice
			s = &stroker{patherStroker: patherStroker{pather: pather{pdf: r.pdf}, strokeOpacityStates: r.strokeOpacityStates}}
		} // else s = nil
	} else {
		if willDraw { // write the path
			s = &patherStroker{pather: pather{pdf: r.pdf}, strokeOpacityStates: r.strokeOpacityStates}
		}
	}
	return f, s
}

func fixedTof(a fixed.Point26_6) (float64, float64) {
	return float64(a.X) / 64, float64(a.Y) / 64
}

func fToFixed(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
}

func (p *pather) Clear() {
	p.boundingBox = svgpdf.BoundingBox{}
}

func (p *pather) Start(a fixed.Point26_6) {
	x, y := fixedTof(a)
	p.pdf.Ops(contentstream.OpMoveTo{X: x, Y: y})
	p.boundingBox.Start(a)
}

func (p *pather) Line(b fixed.Point26_6) {
	x, y := fixedTof(b)
	p.pdf.Ops(contentstream.OpLineTo{X: x, Y: y})
	p.boundingBox.Line(b)
}

func (p *pather) QuadBezier(b fixed.Point26_6, c fixed.Point26_6) {
	cx, cy := fixedTof(b)
	x, y := fixedTof(c)
	p.pdf.Ops(contentstream.OpCurveTo1{X2: cx, Y2: cy, X3: x, Y3: y})
	p.boundingBox.QuadBezier(b, c)
}

func (p *pather) CubeBezier(b fixed.Point26_6, c fixed.Point26_6, d fixed.Point26_6) {
	cx0, cy0 := fixedTof(b)
	cx1, cy1 := fixedTof(c)
	x, y := fixedTof(d)
	p.pdf.Ops(contentstream.OpCubicTo{X1: cx0, Y1: cy0, X2: cx1, Y2: cy1, X3: x, Y3: y})
	p.boundingBox.CubeBezier(b, c, d)
}

func (p *pather) Stop(closeLoop bool) {
	if closeLoop {
		p.pdf.Ops(contentstream.OpClosePath{})
	}
}

// TODO: support gradient
func (f filler) Draw(color svgicon.Pattern, opacity float64) {
	switch color := color.(type) {
	case svgicon.PlainColor:
		f.pdf.SetColorFill(color)
		opacity *= float64(color.A) / 255.
		// cache the opacity states
		gs, ok := f.fillOpacityStates[opacity]
		if !ok {
			gs = &model.GraphicState{Ca: model.ObjFloat(opacity), BM: []model.Name{"Normal"}}
			f.fillOpacityStates[opacity] = gs
		}
		name := f.pdf.AddExtGState(gs)
		f.pdf.Ops(contentstream.OpSetExtGState{Dict: name})
	case svgicon.Gradient:
		// mat := color.ApplyPathExtent(f.boundingBox.BBox)

	}

	if f.useNonZeroWinding {
		f.pdf.Ops(contentstream.OpFill{})
	} else {
		f.pdf.Ops(contentstream.OpEOFill{})
	}
}

func (f *filler) SetWinding(useNonZeroWinding bool) {
	f.useNonZeroWinding = useNonZeroWinding
}

func (f *patherStroker) SetStrokeOptions(options svgicon.StrokeOptions) {
	var capStyle, joinStyle uint8
	switch options.Join.TrailLineCap {
	case svgicon.ButtCap:
		capStyle = 0
	case svgicon.RoundCap:
		capStyle = 1
	case svgicon.SquareCap:
		capStyle = 2
	}
	switch options.Join.LineJoin {
	case svgicon.Bevel:
		joinStyle = 2
	case svgicon.Miter:
		joinStyle = 0
	case svgicon.Round:
		joinStyle = 1
	}

	f.pdf.Ops(
		contentstream.OpSetDash{Dash: model.DashPattern{
			Array: options.Dash.Dash,
			Phase: options.Dash.DashOffset,
		}},
		contentstream.OpSetLineWidth{W: float64(options.LineWidth) / 64},
		contentstream.OpSetLineCap{Style: capStyle},
		contentstream.OpSetLineJoin{Style: joinStyle},
		contentstream.OpSetMiterLimit{Limit: float64(options.Join.MiterLimit) / 64},
	)
}

// TODO: support gradient
func (f patherStroker) Draw(color svgicon.Pattern, opacity float64) {
	switch color := color.(type) {
	case svgicon.PlainColor:
		f.pdf.SetColorStroke(color)
		opacity *= float64(color.A) / 255.
		// cache the opacity states
		gs, ok := f.strokeOpacityStates[opacity]
		if !ok {
			gs = &model.GraphicState{CA: model.ObjFloat(opacity), BM: []model.Name{"Normal"}}
			f.strokeOpacityStates[opacity] = gs
		}
		name := f.pdf.AddExtGState(gs)
		f.pdf.Ops(contentstream.OpSetExtGState{Dict: name})
	}
	f.pdf.Ops(contentstream.OpStroke{})
}

// the stroker doesnt write the path again

func (p stroker) Clear() {}

func (p stroker) Start(a fixed.Point26_6) {}

func (p stroker) Line(b fixed.Point26_6) {}

func (p stroker) QuadBezier(b fixed.Point26_6, c fixed.Point26_6) {}

func (p stroker) CubeBezier(b fixed.Point26_6, c fixed.Point26_6, d fixed.Point26_6) {}

func (p stroker) Stop(closeLoop bool) {}
