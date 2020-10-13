package svgpath

import (
	"image/color"
)

// GradientUnits is the type for gradient units
type GradientUnits byte

// SVG bounds paremater constants
const (
	ObjectBoundingBox GradientUnits = iota
	UserSpaceOnUse
)

// SpreadMethod is the type for spread parameters
type SpreadMethod byte

// SVG spread parameter constants
const (
	PadSpread SpreadMethod = iota
	ReflectSpread
	RepeatSpread
)

const epsilonF = 1e-5

// GradStop represents a stop in the SVG 2.0 gradient specification
type GradStop struct {
	StopColor color.Color
	Offset    float64
	Opacity   float64
}

// Gradient holds a description of an SVG 2.0 gradient
type Gradient struct {
	// Points   [5]float64
	Direction gradientDirecter
	Stops     []GradStop
	Bounds    struct{ X, Y, W, H float64 }
	Matrix    Matrix2D
	Spread    SpreadMethod
	Units     GradientUnits
}

// radial or linear
type gradientDirecter interface {
	isRadial() bool
}

// x1, y1, x2, y2
type Linear [4]float64

func (Linear) isRadial() bool { return false }

// cx, cy, fx, fy, r, fr
type Radial [6]float64

func (Radial) isRadial() bool { return true }
