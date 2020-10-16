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
// In particular, tranformations matrix are already applied to the points
// before sending them to the Driver.
type Driver interface {
	// Clear must reset the internal state (used before starting a new path painting)
	Clear()

	// Decide to use or not the NonZeroWinding rule.
	SetWinding(useNonZeroWinding bool)

	// Set the filling color (plain color or gradient)
	SetFillColor(color Pattern, opacity float64)

	// Set the stroking color (plain color or gradient)
	SetStrokeColor(color Pattern, opacity float64)

	// Parametrize the stroking style
	SetStrokeOptions(options StrokeOptions)

	// SetFillingMode switch between filling or stroking
	SetFillingMode(fill bool)

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

	// Draw fills or strokes the accumulated path using the current settings
	// depending on the filling mode
	Draw()
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

	if svgp.Style.FillerColor != nil { // nil color disable filling
		d.Clear()
		d.SetWinding(svgp.Style.UseNonZeroWinding)

		d.SetFillingMode(true)
		for _, op := range svgp.Path {
			op.drawTo(d, svgp.Style.transform)
		}
		d.Stop(false)

		d.SetFillColor(svgp.Style.FillerColor, svgp.Style.FillOpacity*opacity)
		d.Draw()
		d.SetWinding(true) // default is true
	}

	if svgp.Style.LinerColor != nil { // nil color disable lining
		d.Clear()

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
		d.SetStrokeOptions(StrokeOptions{
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

		d.SetFillingMode(false)
		for _, op := range svgp.Path {
			op.drawTo(d, svgp.Style.transform)
		}
		d.Stop(false)

		d.SetStrokeColor(svgp.Style.LinerColor, svgp.Style.LineOpacity*opacity)
		d.Draw()
	}
}
