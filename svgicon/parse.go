package svgicon

import (
	"strings"

	"encoding/xml"
	"errors"
	"log"
	"math"

	"golang.org/x/image/math/fixed"
)

func init() {
	// avoids cyclical static declaration
	// called on package initialization
	drawFuncs["use"] = useF
}

type (
	// PathStyle holds the state of the SVG style
	PathStyle struct {
		FillOpacity, LineOpacity float64
		LineWidth                float64
		UseNonZeroWinding        bool

		Join                    JoinOptions
		Dash                    DashOptions
		FillerColor, LinerColor Pattern // either PlainColor or Gradient

		transform Matrix2D // current transform
	}

	// SvgPath binds a style to a path
	SvgPath struct {
		Path  Path
		Style PathStyle
	}

	// SvgIcon holds data from parsed SVGs
	SvgIcon struct {
		ViewBox      struct{ X, Y, W, H float64 }
		Titles       []string // Title elements collect here
		Descriptions []string // Description elements collect here
		SVGPaths     []SvgPath
		Transform    Matrix2D

		grads map[string]*Gradient
		defs  map[string][]definition
	}

	// iconCursor is used while parsing SVG files
	iconCursor struct {
		pathCursor
		icon                                    *SvgIcon
		styleStack                              []PathStyle
		grad                                    *Gradient
		inTitleText, inDescText, inGrad, inDefs bool
		currentDef                              []definition
	}

	// definition is used to store what's given in a def tag
	definition struct {
		ID, Tag string
		Attrs   []xml.Attr
	}
)

func fToFixed(f float64) fixed.Int26_6 {
	return fixed.Int26_6(f * 64)
}

// DefaultStyle sets the default PathStyle to fill black, winding rule,
// full opacity, no stroke, ButtCap line end and Bevel line connect.
var DefaultStyle = PathStyle{
	FillOpacity:       1.0,
	LineOpacity:       1.0,
	LineWidth:         2.0,
	UseNonZeroWinding: true,
	Join: JoinOptions{
		MiterLimit:   fToFixed(4),
		LineJoin:     Bevel,
		TrailLineCap: ButtCap,
	},
	FillerColor: NewPlainColor(0x00, 0x00, 0x00, 0xff),
	transform:   Identity,
}

func (c *iconCursor) readTransformAttr(m1 Matrix2D, k string) (Matrix2D, error) {
	ln := len(c.points)
	switch k {
	case "rotate":
		if ln == 1 {
			m1 = m1.Rotate(c.points[0] * math.Pi / 180)
		} else if ln == 3 {
			m1 = m1.Translate(c.points[1], c.points[2]).
				Rotate(c.points[0]*math.Pi/180).
				Translate(-c.points[1], -c.points[2])
		} else {
			return m1, errParamMismatch
		}
	case "translate":
		if ln == 1 {
			m1 = m1.Translate(c.points[0], 0)
		} else if ln == 2 {
			m1 = m1.Translate(c.points[0], c.points[1])
		} else {
			return m1, errParamMismatch
		}
	case "skewx":
		if ln == 1 {
			m1 = m1.SkewX(c.points[0] * math.Pi / 180)
		} else {
			return m1, errParamMismatch
		}
	case "skewy":
		if ln == 1 {
			m1 = m1.SkewY(c.points[0] * math.Pi / 180)
		} else {
			return m1, errParamMismatch
		}
	case "scale":
		if ln == 1 {
			m1 = m1.Scale(c.points[0], 0)
		} else if ln == 2 {
			m1 = m1.Scale(c.points[0], c.points[1])
		} else {
			return m1, errParamMismatch
		}
	case "matrix":
		if ln == 6 {
			m1 = m1.Mult(Matrix2D{
				A: c.points[0],
				B: c.points[1],
				C: c.points[2],
				D: c.points[3],
				E: c.points[4],
				F: c.points[5]})
		} else {
			return m1, errParamMismatch
		}
	default:
		return m1, errParamMismatch
	}
	return m1, nil
}

func (c *iconCursor) parseTransform(v string) (Matrix2D, error) {
	ts := strings.Split(v, ")")
	m1 := c.styleStack[len(c.styleStack)-1].transform
	for _, t := range ts {
		t = strings.TrimSpace(t)
		if len(t) == 0 {
			continue
		}
		d := strings.Split(t, "(")
		if len(d) != 2 || len(d[1]) < 1 {
			return m1, errParamMismatch // badly formed transformation
		}
		err := c.getPoints(d[1])
		if err != nil {
			return m1, err
		}
		m1, err = c.readTransformAttr(m1, strings.ToLower(strings.TrimSpace(d[0])))
		if err != nil {
			return m1, err
		}
	}
	return m1, nil
}

func (c *iconCursor) readStyleAttr(curStyle *PathStyle, k, v string) error {
	switch k {
	case "fill":
		gradient, ok := c.readGradURL(v, curStyle.FillerColor)
		if ok {
			curStyle.FillerColor = gradient
			break
		}
		optCol, err := parseSVGColor(v)
		curStyle.FillerColor = optCol.asPattern()
		return err
	case "stroke":
		gradient, ok := c.readGradURL(v, curStyle.LinerColor)
		if ok {
			curStyle.LinerColor = gradient
			break
		}
		col, errc := parseSVGColor(v)
		if errc != nil {
			return errc
		}
		curStyle.LinerColor = col.asPattern()
	case "stroke-linegap":
		switch v {
		case "flat":
			curStyle.Join.LineGap = FlatGap
		case "round":
			curStyle.Join.LineGap = RoundGap
		case "cubic":
			curStyle.Join.LineGap = CubicGap
		case "quadratic":
			curStyle.Join.LineGap = QuadraticGap
		}
	case "stroke-leadlinecap":
		switch v {
		case "butt":
			curStyle.Join.LeadLineCap = ButtCap
		case "round":
			curStyle.Join.LeadLineCap = RoundCap
		case "square":
			curStyle.Join.LeadLineCap = SquareCap
		case "cubic":
			curStyle.Join.LeadLineCap = CubicCap
		case "quadratic":
			curStyle.Join.LeadLineCap = QuadraticCap
		}
	case "stroke-linecap":
		switch v {
		case "butt":
			curStyle.Join.TrailLineCap = ButtCap
		case "round":
			curStyle.Join.TrailLineCap = RoundCap
		case "square":
			curStyle.Join.TrailLineCap = SquareCap
		case "cubic":
			curStyle.Join.TrailLineCap = CubicCap
		case "quadratic":
			curStyle.Join.TrailLineCap = QuadraticCap
		}
	case "stroke-linejoin":
		switch v {
		case "miter":
			curStyle.Join.LineJoin = Miter
		case "miter-clip":
			curStyle.Join.LineJoin = MiterClip
		case "arc-clip":
			curStyle.Join.LineJoin = ArcClip
		case "round":
			curStyle.Join.LineJoin = Round
		case "arc":
			curStyle.Join.LineJoin = Arc
		case "bevel":
			curStyle.Join.LineJoin = Bevel
		}
	case "stroke-miterlimit":
		mLimit, err := parseFloat(v, 64)
		if err != nil {
			return err
		}
		curStyle.Join.MiterLimit = fToFixed(mLimit)
	case "stroke-width":
		width, err := parseFloat(v, 64)
		if err != nil {
			return err
		}
		curStyle.LineWidth = width
	case "stroke-dashoffset":
		dashOffset, err := parseFloat(v, 64)
		if err != nil {
			return err
		}
		curStyle.Dash.DashOffset = dashOffset
	case "stroke-dasharray":
		if v != "none" {
			dashes := splitOnCommaOrSpace(v)
			dList := make([]float64, len(dashes))
			for i, dstr := range dashes {
				d, err := parseFloat(strings.TrimSpace(dstr), 64)
				if err != nil {
					return err
				}
				dList[i] = d
			}
			curStyle.Dash.Dash = dList
			break
		}
	case "opacity", "stroke-opacity", "fill-opacity":
		op, err := parseFloat(v, 64)
		if err != nil {
			return err
		}
		if k != "stroke-opacity" {
			curStyle.FillOpacity *= op
		}
		if k != "fill-opacity" {
			curStyle.LineOpacity *= op
		}
	case "transform":
		m, err := c.parseTransform(v)
		if err != nil {
			return err
		}
		curStyle.transform = m
	}
	return nil
}

// pushStyle parses the style element, and push it on the style stack. Only color and opacity are supported
// for fill. Note that this parses both the contents of a style attribute plus
// direct fill and opacity attributes.
func (c *iconCursor) pushStyle(attrs []xml.Attr) error {
	var pairs []string
	for _, attr := range attrs {
		switch strings.ToLower(attr.Name.Local) {
		case "style":
			pairs = append(pairs, strings.Split(attr.Value, ";")...)
		default:
			pairs = append(pairs, attr.Name.Local+":"+attr.Value)
		}
	}
	// Make a copy of the top style
	curStyle := c.styleStack[len(c.styleStack)-1]
	for _, pair := range pairs {
		kv := strings.Split(pair, ":")
		if len(kv) >= 2 {
			k := strings.ToLower(kv[0])
			k = strings.TrimSpace(k)
			v := strings.TrimSpace(kv[1])
			err := c.readStyleAttr(&curStyle, k, v)
			if err != nil {
				return err
			}
		}
	}
	c.styleStack = append(c.styleStack, curStyle) // Push style onto stack
	return nil
}

// splitOnCommaOrSpace returns a list of strings after splitting the input on comma and space delimiters
func splitOnCommaOrSpace(s string) []string {
	return strings.FieldsFunc(s,
		func(r rune) bool {
			return r == ',' || r == ' '
		})
}

func (c *iconCursor) readStartElement(se xml.StartElement) (err error) {
	var skipDef bool
	if se.Name.Local == "radialGradient" || se.Name.Local == "linearGradient" || c.inGrad {
		skipDef = true
	}
	if c.inDefs && !skipDef {
		ID := ""
		for _, attr := range se.Attr {
			if attr.Name.Local == "id" {
				ID = attr.Value
			}
		}
		if ID != "" && len(c.currentDef) > 0 {
			c.icon.defs[c.currentDef[0].ID] = c.currentDef
			c.currentDef = make([]definition, 0)
		}
		c.currentDef = append(c.currentDef, definition{
			ID:    ID,
			Tag:   se.Name.Local,
			Attrs: se.Attr,
		})
		return nil
	}
	df, ok := drawFuncs[se.Name.Local]
	if !ok {
		errStr := "Cannot process svg element " + se.Name.Local
		if c.errorMode == StrictErrorMode {
			return errors.New(errStr)
		} else if c.errorMode == WarnErrorMode {
			log.Println(errStr)
		}
		return nil
	}
	err = df(c, se.Attr)

	if len(c.path) > 0 {
		//The cursor parsed a path from the xml element
		pathCopy := append(Path{}, c.path...)
		c.icon.SVGPaths = append(c.icon.SVGPaths,
			SvgPath{Path: pathCopy, Style: c.styleStack[len(c.styleStack)-1]})
		c.path = c.path[:0]
	}
	return
}

func readFraction(v string) (f float64, err error) {
	v = strings.TrimSpace(v)
	d := 1.0
	if strings.HasSuffix(v, "%") {
		d = 100
		v = strings.TrimSuffix(v, "%")
	}
	f, err = parseFloat(v, 64)
	f /= d
	// Is this is an unnecessary restriction? For now fractions can be all values not just in the range [0,1]
	// if f > 1 {
	// 	f = 1
	// } else if f < 0 {
	// 	f = 0
	// }
	return
}

type svgFunc func(c *iconCursor, attrs []xml.Attr) error

var drawFuncs = map[string]svgFunc{
	"svg":            svgF,
	"g":              gF,
	"line":           lineF,
	"stop":           stopF,
	"rect":           rectF,
	"circle":         circleF,
	"ellipse":        circleF, //circleF handles ellipse also
	"polyline":       polylineF,
	"polygon":        polygonF,
	"path":           pathF,
	"desc":           descF,
	"defs":           defsF,
	"title":          titleF,
	"linearGradient": linearGradientF,
	"radialGradient": radialGradientF,
}

func svgF(c *iconCursor, attrs []xml.Attr) error {
	c.icon.ViewBox.X = 0
	c.icon.ViewBox.Y = 0
	c.icon.ViewBox.W = 0
	c.icon.ViewBox.H = 0
	var width, height float64
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "viewBox":
			err = c.getPoints(attr.Value)
			if len(c.points) != 4 {
				return errParamMismatch
			}
			c.icon.ViewBox.X = c.points[0]
			c.icon.ViewBox.Y = c.points[1]
			c.icon.ViewBox.W = c.points[2]
			c.icon.ViewBox.H = c.points[3]
		case "width":
			width, err = parseFloat(attr.Value, 64)
		case "height":
			height, err = parseFloat(attr.Value, 64)
		}
		if err != nil {
			return err
		}
	}
	if c.icon.ViewBox.W == 0 {
		c.icon.ViewBox.W = width
	}
	if c.icon.ViewBox.H == 0 {
		c.icon.ViewBox.H = height
	}
	return nil
}
func gF(*iconCursor, []xml.Attr) error { return nil } // g does nothing but push the style
func rectF(c *iconCursor, attrs []xml.Attr) error {
	var x, y, w, h, rx, ry float64
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "x":
			x, err = parseFloat(attr.Value, 64)
		case "y":
			y, err = parseFloat(attr.Value, 64)
		case "width":
			w, err = parseFloat(attr.Value, 64)
		case "height":
			h, err = parseFloat(attr.Value, 64)
		case "rx":
			rx, err = parseFloat(attr.Value, 64)
		case "ry":
			ry, err = parseFloat(attr.Value, 64)
		}
		if err != nil {
			return err
		}
	}
	if w == 0 || h == 0 {
		return nil
	}
	c.path.addRoundRect(x+c.curX, y+c.curY, w+x+c.curX, h+y+c.curY, rx, ry, 0)
	return nil
}
func circleF(c *iconCursor, attrs []xml.Attr) error {
	var cx, cy, rx, ry float64
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "cx":
			cx, err = parseFloat(attr.Value, 64)
		case "cy":
			cy, err = parseFloat(attr.Value, 64)
		case "r":
			rx, err = parseFloat(attr.Value, 64)
			ry = rx
		case "rx":
			rx, err = parseFloat(attr.Value, 64)
		case "ry":
			ry, err = parseFloat(attr.Value, 64)
		}
		if err != nil {
			return err
		}
	}
	if rx == 0 || ry == 0 { // not drawn, but not an error
		return nil
	}
	c.ellipseAt(cx+c.curX, cy+c.curY, rx, ry)
	return nil
}
func lineF(c *iconCursor, attrs []xml.Attr) error {
	var x1, x2, y1, y2 float64
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "x1":
			x1, err = parseFloat(attr.Value, 64)
		case "x2":
			x2, err = parseFloat(attr.Value, 64)
		case "y1":
			y1, err = parseFloat(attr.Value, 64)
		case "y2":
			y2, err = parseFloat(attr.Value, 64)
		}
		if err != nil {
			return err
		}
	}
	c.path.Start(fixed.Point26_6{
		X: fixed.Int26_6((x1 + c.curX) * 64),
		Y: fixed.Int26_6((y1 + c.curY) * 64)})
	c.path.Line(fixed.Point26_6{
		X: fixed.Int26_6((x2 + c.curX) * 64),
		Y: fixed.Int26_6((y2 + c.curY) * 64)})
	return nil
}
func polylineF(c *iconCursor, attrs []xml.Attr) error {
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "points":
			err = c.getPoints(attr.Value)
			if len(c.points)%2 != 0 {
				return errors.New("polygon has odd number of points")
			}
		}
		if err != nil {
			return err
		}
	}
	if len(c.points) > 4 {
		c.path.Start(fixed.Point26_6{
			X: fixed.Int26_6((c.points[0] + c.curX) * 64),
			Y: fixed.Int26_6((c.points[1] + c.curY) * 64)})
		for i := 2; i < len(c.points)-1; i += 2 {
			c.path.Line(fixed.Point26_6{
				X: fixed.Int26_6((c.points[i] + c.curX) * 64),
				Y: fixed.Int26_6((c.points[i+1] + c.curY) * 64)})
		}
	}
	return nil
}
func polygonF(c *iconCursor, attrs []xml.Attr) error {
	err := polylineF(c, attrs)
	if len(c.points) > 4 {
		c.path.Stop(true)
	}
	return err
}
func pathF(c *iconCursor, attrs []xml.Attr) error {
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "d":
			err = c.compilePath(attr.Value)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func descF(c *iconCursor, attrs []xml.Attr) error {
	c.inDescText = true
	c.icon.Descriptions = append(c.icon.Descriptions, "")
	return nil
}
func titleF(c *iconCursor, attrs []xml.Attr) error {
	c.inTitleText = true
	c.icon.Titles = append(c.icon.Titles, "")
	return nil
}
func defsF(c *iconCursor, attrs []xml.Attr) error {
	c.inDefs = true
	return nil
}
func linearGradientF(c *iconCursor, attrs []xml.Attr) error {
	var err error
	c.inGrad = true
	direction := Linear{0, 0, 1, 0}
	c.grad = &Gradient{Direction: direction, Bounds: c.icon.ViewBox, Matrix: Identity}
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "id":
			id := attr.Value
			if len(id) >= 0 {
				c.icon.grads[id] = c.grad
			} else {
				return errZeroLengthID
			}
		case "x1":
			direction[0], err = readFraction(attr.Value)
		case "y1":
			direction[1], err = readFraction(attr.Value)
		case "x2":
			direction[2], err = readFraction(attr.Value)
		case "y2":
			direction[3], err = readFraction(attr.Value)
		default:
			err = c.readGradAttr(attr)
		}
		if err != nil {
			return err
		}
	}
	c.grad.Direction = direction
	return nil
}

func radialGradientF(c *iconCursor, attrs []xml.Attr) error {
	c.inGrad = true
	direction := Radial{0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	c.grad = &Gradient{Direction: direction, Bounds: c.icon.ViewBox, Matrix: Identity}
	var setFx, setFy bool
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "id":
			id := attr.Value
			if len(id) >= 0 {
				c.icon.grads[id] = c.grad
			} else {
				return errZeroLengthID
			}
		case "cx":
			direction[0], err = readFraction(attr.Value)
		case "cy":
			direction[1], err = readFraction(attr.Value)
		case "fx":
			setFx = true
			direction[2], err = readFraction(attr.Value)
		case "fy":
			setFy = true
			direction[3], err = readFraction(attr.Value)
		case "r":
			direction[4], err = readFraction(attr.Value)
		case "fr":
			direction[5], err = readFraction(attr.Value)
		default:
			err = c.readGradAttr(attr)
		}
		if err != nil {
			return err
		}
	}
	if !setFx { // set fx to cx by default
		direction[2] = direction[0]
	}
	if !setFy { // set fy to cy by default
		direction[3] = direction[1]
	}
	return nil
}
func stopF(c *iconCursor, attrs []xml.Attr) error {
	var err error
	if c.inGrad {
		stop := GradStop{Opacity: 1.0}
		for _, attr := range attrs {
			switch attr.Name.Local {
			case "offset":
				stop.Offset, err = readFraction(attr.Value)
			case "stop-color":
				//todo: add current color inherit
				var optColor optionnalColor
				optColor, err = parseSVGColor(attr.Value)
				stop.StopColor = optColor.asColor()
			case "stop-opacity":
				stop.Opacity, err = parseFloat(attr.Value, 64)
			}
			if err != nil {
				return err
			}
		}
		c.grad.Stops = append(c.grad.Stops, stop)
	}
	return nil
}
func useF(c *iconCursor, attrs []xml.Attr) error {
	var (
		href string
		x, y float64
		err  error
	)
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "href":
			href = attr.Value
		case "x":
			x, err = parseFloat(attr.Value, 64)
		case "y":
			y, err = parseFloat(attr.Value, 64)
		}
		if err != nil {
			return err
		}
	}
	c.curX, c.curY = x, y
	defer func() {
		c.curX, c.curY = 0, 0
	}()
	if href == "" {
		return errors.New("only use tags with href is supported")
	}
	if !strings.HasPrefix(href, "#") {
		return errors.New("only the ID CSS selector is supported")
	}
	defs, ok := c.icon.defs[href[1:]]
	if !ok {
		return errors.New("href ID in use statement was not found in saved defs")
	}
	for _, def := range defs {
		if def.Tag == "endg" {
			// pop style
			c.styleStack = c.styleStack[:len(c.styleStack)-1]
			continue
		}
		if err = c.pushStyle(def.Attrs); err != nil {
			return err
		}
		df, ok := drawFuncs[def.Tag]
		if !ok {
			errStr := "Cannot process svg element " + def.Tag
			if c.errorMode == StrictErrorMode {
				return errors.New(errStr)
			} else if c.errorMode == WarnErrorMode {
				log.Println(errStr)
			}
			return nil
		}
		if err := df(c, def.Attrs); err != nil {
			return err
		}
		if def.Tag != "g" {
			// pop style
			c.styleStack = c.styleStack[:len(c.styleStack)-1]
		}
	}
	return nil
}
