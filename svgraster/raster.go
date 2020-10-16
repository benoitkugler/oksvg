// Implements a raster backend to render SVG images,
// by wrapping rasterx.
package svgraster

import (
	"image"
	"io"

	"github.com/benoitkugler/oksvg/svgicon"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/math/fixed"
)

var _ svgicon.Driver = (*Renderer)(nil) // assert interface conformance

type Renderer struct {
	dasher    *rasterx.Dasher
	isFilling bool
}

// NewRenderer returns a renderer with default values.
// In addition to rasterizing lines like a Scanner,
// it can also rasterize quadratic and cubic bezier curves.
// If scanner is nil, a default scanner rasterx.ScannerGV is used
func NewRenderer(width, height int, scanner rasterx.Scanner) *Renderer {
	return &Renderer{dasher: rasterx.NewDasher(width, height, scanner)}
}

// RasterSVGIconToImage uses a ScannerGV instance to renderer the
// icon into an image and returns it
func RasterSVGIconToImage(icon io.Reader) (*image.RGBA, error) {
	parsedIcon, err := svgicon.ReadIconStream(icon, svgicon.WarnErrorMode)
	if err != nil {
		return nil, err
	}
	w, h := int(parsedIcon.ViewBox.W), int(parsedIcon.ViewBox.H)
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	renderer := NewRenderer(w, h, scanner)
	parsedIcon.Draw(renderer, 1.0)
	return img, nil
}

func (rd Renderer) Clear() {
	rd.dasher.Clear()
}

func (rd *Renderer) SetFillingMode(fill bool) {
	rd.isFilling = fill
}

func (rd Renderer) SetWinding(useNonZeroWinding bool) {
	rd.dasher.SetWinding(useNonZeroWinding)
}

func toRasterxGradient(grad svgicon.Gradient) rasterx.Gradient {
	var (
		points   [5]float64
		isRadial bool
	)
	switch dir := grad.Direction.(type) {
	case svgicon.Linear:
		points[0], points[1], points[2], points[3] = dir[0], dir[1], dir[2], dir[3]
		isRadial = false
	case svgicon.Radial:
		points[0], points[1], points[2], points[3], points[4], _ = dir[0], dir[1], dir[2], dir[3], dir[4], dir[5] // in rasterx fr is ignored
		isRadial = true
	}
	stops := make([]rasterx.GradStop, len(grad.Stops))
	for i := range grad.Stops {
		stops[i] = rasterx.GradStop(grad.Stops[i])
	}
	return rasterx.Gradient{
		Points:   points,
		Stops:    stops,
		Bounds:   grad.Bounds,
		Matrix:   rasterx.Matrix2D(grad.Matrix),
		Spread:   rasterx.SpreadMethod(grad.Spread),
		Units:    rasterx.GradientUnits(grad.Units),
		IsRadial: isRadial,
	}
}

// resolve gradient color
func setColorFromPattern(color svgicon.Pattern, opacity float64, scanner rasterx.Scanner) {
	switch color := color.(type) {
	case svgicon.PlainColor:
		scanner.SetColor(rasterx.ApplyOpacity(color, opacity))
	case svgicon.Gradient:
		if color.Units == svgicon.ObjectBoundingBox {
			fRect := scanner.GetPathExtent()
			mnx, mny := float64(fRect.Min.X)/64, float64(fRect.Min.Y)/64
			mxx, mxy := float64(fRect.Max.X)/64, float64(fRect.Max.Y)/64
			color.Bounds.X, color.Bounds.Y = mnx, mny
			color.Bounds.W, color.Bounds.H = mxx-mnx, mxy-mny
		}
		rasterxGradient := toRasterxGradient(color)
		scanner.SetColor(rasterxGradient.GetColorFunction(opacity))
	}
}

func (rd Renderer) SetFillColor(color svgicon.Pattern, opacity float64) {
	setColorFromPattern(color, opacity, rd.dasher.Scanner)
}

func (rd Renderer) SetStrokeColor(color svgicon.Pattern, opacity float64) {
	setColorFromPattern(color, opacity, rd.dasher.Scanner)
}

var (
	joinToJoin = [...]rasterx.JoinMode{
		svgicon.Round:     rasterx.Round,
		svgicon.Bevel:     rasterx.Bevel,
		svgicon.Miter:     rasterx.Miter,
		svgicon.MiterClip: rasterx.MiterClip,
		svgicon.Arc:       rasterx.Arc,
		svgicon.ArcClip:   rasterx.ArcClip,
	}

	capToFunc = [...]rasterx.CapFunc{
		svgicon.ButtCap:      rasterx.ButtCap,
		svgicon.SquareCap:    rasterx.SquareCap,
		svgicon.RoundCap:     rasterx.RoundCap,
		svgicon.CubicCap:     rasterx.CubicCap,
		svgicon.QuadraticCap: rasterx.QuadraticCap,
	}

	gapToFunc = [...]rasterx.GapFunc{
		svgicon.FlatGap:      rasterx.FlatGap,
		svgicon.RoundGap:     rasterx.RoundGap,
		svgicon.CubicGap:     rasterx.CubicGap,
		svgicon.QuadraticGap: rasterx.QuadraticGap,
	}
)

func (rd Renderer) SetStrokeOptions(options svgicon.StrokeOptions) {
	rd.dasher.SetStroke(
		options.LineWidth, options.Join.MiterLimit, capToFunc[options.Join.LeadLineCap],
		capToFunc[options.Join.TrailLineCap], gapToFunc[options.Join.LineGap],
		joinToJoin[options.Join.LineJoin], options.Dash.Dash, options.Dash.DashOffset,
	)
}

func (rd Renderer) Start(a fixed.Point26_6) {
	if rd.isFilling {
		rd.dasher.Filler.Start(a)
	} else {
		rd.dasher.Start(a)
	}
}

func (rd Renderer) Line(b fixed.Point26_6) {
	if rd.isFilling {
		rd.dasher.Filler.Line(b)
	} else {
		rd.dasher.Line(b)
	}
}

func (rd Renderer) QuadBezier(b fixed.Point26_6, c fixed.Point26_6) {
	if rd.isFilling {
		rd.dasher.Filler.QuadBezier(b, c)
	} else {
		rd.dasher.QuadBezier(b, c)
	}
}

func (rd Renderer) CubeBezier(b fixed.Point26_6, c fixed.Point26_6, d fixed.Point26_6) {
	if rd.isFilling {
		rd.dasher.Filler.CubeBezier(b, c, d)
	} else {
		rd.dasher.CubeBezier(b, c, d)
	}
}

func (rd Renderer) Stop(closeLoop bool) {
	if rd.isFilling {
		rd.dasher.Filler.Stop(closeLoop)
	} else {
		rd.dasher.Stop(closeLoop)
	}
}

func (rd Renderer) Draw() {
	if rd.isFilling {
		rd.dasher.Filler.Draw()
	} else {
		rd.dasher.Draw()
	}
}
