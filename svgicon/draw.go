package svgicon

import (
	"golang.org/x/image/math/fixed"
)

// Given a parsed SVG document, implements how to
// draw it on screen.
// This requires a driver implementing the actual draw operations,
// such as a rasterizer to output .png images or a pdf writer.

// Drawer knows how to do the actual draw operations
// but doesn't need any SVG kwowledge
// In particular, tranformations matrix are already applied to the points
// before sending them to the Drawer.
type Drawer interface {
	// Clear must reset the internal state (used before starting a new path painting)
	Clear()

	// Start starts a new path at the given point.
	Start(a fixed.Point26_6)

	// Line Adds a line for the current point to `b`
	Line(b fixed.Point26_6)

	// QuadBezier adds a quadratic bezier curve to the path
	QuadBezier(b, c fixed.Point26_6)

	// CubeBezier adds a cubic bezier curve to the path
	CubeBezier(b, c, d fixed.Point26_6)

	// Closes the path to the start point if `closeLoop` is true
	Stop(closeLoop bool)

	// SetColor set the color for the current path
	SetColor(color Pattern, opacity float64)

	// Draw fills or strokes the accumulated path using the current settings
	// depending on the filling mode
	Draw()
}

type Filler interface {
	Drawer

	// Decide to use or not the NonZeroWinding rule for the current path
	SetWinding(useNonZeroWinding bool)
}

type Stroker interface {
	Drawer

	// Parametrize the stroking style for the current path
	SetStrokeOptions(options StrokeOptions)
}

type Driver interface {
	// SetupDrawers returns the backend painters, and
	// will be called at the begining of every path.
	// If the `willXXX` boolean is false, the returned drawer should be nil
	// to avoid useless operations.
	// When both booleans are true, one can assume that the exact same draw operations
	// will be performed on the Filler first and then on the Stroker.
	// This promise may enable the implementation to avoid duplicating filled and stroked paths
	SetupDrawers(willFill, willStroke bool) (Filler, Stroker)
}

type DashOptions struct {
	Dash       []float64 // values for the dash pattern (nil or an empty slice for no dashes)
	DashOffset float64   // starting offset into the dash array
}

// JoinMode type to specify how segments join.
type JoinMode uint8

// JoinMode constants determine how stroke segments bridge the gap at a join
// ArcClip mode is like MiterClip applied to arcs, and is not part of the SVG2.0
// standard.
const (
	Arc JoinMode = iota // New in SVG2
	Round
	Bevel
	Miter
	MiterClip // New in SVG2
	ArcClip   // Like MiterClip applied to arcs, and is not part of the SVG2.0 standard.
)

func (s JoinMode) String() string {
	switch s {
	case Round:
		return "Round"
	case Bevel:
		return "Bevel"
	case Miter:
		return "Miter"
	case MiterClip:
		return "MiterClip"
	case Arc:
		return "Arc"
	case ArcClip:
		return "ArcClip"
	default:
		return "<unknown JoinMode>"
	}
}

// CapMode defines how to draw caps on the ends of lines
type CapMode uint8

const (
	NilCap CapMode = iota // default value
	ButtCap
	SquareCap
	RoundCap
	CubicCap     // Not part of the SVG2.0 standard.
	QuadraticCap // Not part of the SVG2.0 standard.
)

func (c CapMode) String() string {
	switch c {
	case NilCap:
		return "NilCap"
	case ButtCap:
		return "ButtCap"
	case SquareCap:
		return "SquareCap"
	case RoundCap:
		return "RoundCap"
	case CubicCap:
		return "CubicCap"
	case QuadraticCap:
		return "QuadraticCap"
	default:
		return "<unknown CapMode>"
	}
}

// GapMode defines how to bridge gaps when the miter limit is exceeded,
// and is not part of the SVG2.0 standard.
type GapMode uint8

const (
	NilGap GapMode = iota
	FlatGap
	RoundGap
	CubicGap
	QuadraticGap
)

func (g GapMode) String() string {
	switch g {
	case NilGap:
		return "NilGap"
	case FlatGap:
		return "FlatGap"
	case RoundGap:
		return "RoundGap"
	case CubicGap:
		return "CubicGap"
	case QuadraticGap:
		return "QuadraticGap"
	default:
		return "<unknown GapMode>"
	}
}

type JoinOptions struct {
	MiterLimit   fixed.Int26_6 // he miter cutoff value for miter, arc, miterclip and arcClip joinModes
	LineJoin     JoinMode      // JoinMode for curve segments
	TrailLineCap CapMode       // capping functions for leading and trailing line ends. If one is nil, the other function is used at both ends.

	LeadLineCap CapMode // not part of the standard specification
	LineGap     GapMode // not part of the standard specification. determines how a gap on the convex side of two lines joining is filled
}

type StrokeOptions struct {
	LineWidth fixed.Int26_6 // width of the line
	Join      JoinOptions
	Dash      DashOptions
}

// DefaultStyle sets the default PathStyle to fill black, winding rule,
// full opacity, no stroke, ButtCap line end and Bevel line connect.
var DefaultStyle = PathStyle{
	FillOpacity:       1.0,
	LineOpacity:       1.0,
	LineWidth:         2.0,
	UseNonZeroWinding: true,
	Join: JoinOptions{
		MiterLimit:   fToFixed(4.),
		LineJoin:     Bevel,
		TrailLineCap: ButtCap,
	},
	FillerColor: NewPlainColor(0x00, 0x00, 0x00, 0xff),
	transform:   Identity,
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

	filler, stroker := d.SetupDrawers(svgp.Style.FillerColor != nil, svgp.Style.LinerColor != nil)
	if filler != nil { // nil color disable filling
		filler.Clear()
		filler.SetWinding(svgp.Style.UseNonZeroWinding)

		for _, op := range svgp.Path {
			op.drawTo(filler, svgp.Style.transform)
		}
		filler.Stop(false)

		filler.SetColor(svgp.Style.FillerColor, svgp.Style.FillOpacity*opacity)
		filler.Draw()
		filler.SetWinding(true) // default is true
	}

	if stroker != nil { // nil color disable lining
		stroker.Clear()

		lineGap := svgp.Style.Join.LineGap
		if lineGap == NilGap {
			lineGap = DefaultStyle.Join.LineGap
		}
		lineCap := svgp.Style.Join.TrailLineCap
		if lineCap == NilCap {
			lineCap = DefaultStyle.Join.TrailLineCap
		}
		leadLineCap := lineCap
		if svgp.Style.Join.LeadLineCap != NilCap {
			leadLineCap = svgp.Style.Join.LeadLineCap
		}
		stroker.SetStrokeOptions(StrokeOptions{
			LineWidth: fixed.Int26_6(svgp.Style.LineWidth * 64),
			Join: JoinOptions{
				MiterLimit:   svgp.Style.Join.MiterLimit,
				LineJoin:     svgp.Style.Join.LineJoin,
				LeadLineCap:  leadLineCap,
				TrailLineCap: lineCap,
				LineGap:      lineGap,
			},
			Dash: svgp.Style.Dash,
		})

		for _, op := range svgp.Path {
			op.drawTo(stroker, svgp.Style.transform)
		}
		stroker.Stop(false)

		stroker.SetColor(svgp.Style.LinerColor, svgp.Style.LineOpacity*opacity)
		stroker.Draw()
	}
}
