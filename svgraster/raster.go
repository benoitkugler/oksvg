// Implements a raster backend to render SVG images,
// by wrapping github.com/srwiley/rasterx.
package svgraster

import (
	"image"
	"io"

	"github.com/benoitkugler/oksvg/svgicon"
	"github.com/srwiley/rasterx"
)

// assert interface conformance
var (
	_ svgicon.Driver  = (*Renderer)(nil)
	_ svgicon.Filler  = filler{}
	_ svgicon.Stroker = stroker{}
)

type Renderer struct {
	dasher    *rasterx.Dasher
	isFilling bool
}

type filler struct {
	*rasterx.Filler
}

type stroker struct {
	*rasterx.Dasher
}

// NewRenderer returns a renderer with default values.
// In addition to rasterizing lines like a Scanner,
// it can also rasterize quadratic and cubic bezier curves.
// If scanner is nil, a default scanner rasterx.ScannerGV is used
func NewRenderer(width, height int, scanner rasterx.Scanner) *Renderer {
	return &Renderer{dasher: rasterx.NewDasher(width, height, scanner)}
}

func (rd Renderer) SetupDrawers(willFill, willStroke bool) (f svgicon.Filler, s svgicon.Stroker) {
	if willFill {
		f = filler{Filler: &rd.dasher.Filler}
	}
	if willStroke {
		s = stroker{Dasher: rd.dasher}
	}
	return f, s
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

func (f filler) SetColor(color svgicon.Pattern, opacity float64) {
	setColorFromPattern(color, opacity, f.Scanner)
}

func (s stroker) SetColor(color svgicon.Pattern, opacity float64) {
	setColorFromPattern(color, opacity, s.Scanner)
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

func (s stroker) SetStrokeOptions(options svgicon.StrokeOptions) {
	s.SetStroke(
		options.LineWidth, options.Join.MiterLimit, capToFunc[options.Join.LeadLineCap],
		capToFunc[options.Join.TrailLineCap], gapToFunc[options.Join.LineGap],
		joinToJoin[options.Join.LineJoin], options.Dash.Dash, options.Dash.DashOffset,
	)
}
