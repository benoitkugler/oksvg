package svgpath

import (
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html/charset"

	"encoding/xml"
	"errors"
	"image/color"
	"log"
	"math"

	"golang.org/x/image/colornames"
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
		FillOpacity, LineOpacity          float64
		LineWidth, DashOffset, MiterLimit float64
		Dash                              []float64
		UseNonZeroWinding                 bool
		fillerColor, linerColor           interface{} // either color.Color or Gradient
		LineGap                           GapFunc
		LeadLineCap                       CapFunc // This is used if different than LineCap
		LineCap                           CapFunc
		LineJoin                          JoinMode
		transform                         Matrix2D // current transform
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
		Grads        map[string]*Gradient
		Defs         map[string][]definition
		SVGPaths     []SvgPath
		Transform    Matrix2D
	}

	// IconCursor is used while parsing SVG files
	IconCursor struct {
		PathCursor
		icon                                    *SvgIcon
		StyleStack                              []PathStyle
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

// DefaultStyle sets the default PathStyle to fill black, winding rule,
// full opacity, no stroke, ButtCap line end and Bevel line connect.
var DefaultStyle = PathStyle{1.0, 1.0, 2.0, 0.0, 4.0, nil, true,
	color.NRGBA{0x00, 0x00, 0x00, 0xff}, nil,
	nil, nil, ButtCap, Bevel, Identity}

// // Draw the compiled SVG icon into the GraphicContext.
// // All elements should be contained by the Bounds rectangle of the SvgIcon.
// func (s *SvgIcon) Draw(r *rasterx.Dasher, opacity float64) {
// 	for _, svgp := range s.SVGPaths {
// 		svgp.DrawTransformed(r, opacity, s.Transform)
// 	}
// }

// SetTarget sets the Transform matrix to draw within the bounds of the rectangle arguments
func (s *SvgIcon) SetTarget(x, y, w, h float64) {
	scaleW := w / s.ViewBox.W
	scaleH := h / s.ViewBox.H
	s.Transform = Identity.Translate(x-s.ViewBox.X, y-s.ViewBox.Y).Scale(scaleW, scaleH)
}

// // Draw the compiled SvgPath into the Dasher.
// func (svgp *SvgPath) Draw(r *rasterx.Dasher, opacity float64) {
// 	svgp.DrawTransformed(r, opacity, Identity)
// }

// // DrawTransformed draws the compiled SvgPath into the Dasher while applying transform t.
// func (svgp *SvgPath) DrawTransformed(r *rasterx.Dasher, opacity float64, t Matrix2D) {
// 	m := svgp.transform
// 	svgp.transform = t.Mult(m)
// 	defer func() { svgp.transform = m }() // Restore untransformed matrix
// 	if svgp.fillerColor != nil {
// 		r.Clear()
// 		rf := &r.Filler
// 		rf.SetWinding(svgp.UseNonZeroWinding)
// 		svgp.mAdder.Adder = rf // This allows transformations to be applied
// 		svgp.Path.AddTo(&svgp.mAdder)

// 		switch fillerColor := svgp.fillerColor.(type) {
// 		case color.Color:
// 			rf.SetColor(rasterx.ApplyOpacity(fillerColor, svgp.FillOpacity*opacity))
// 		case Gradient:
// 			if fillerColor.Units == rasterx.ObjectBoundingBox {
// 				fRect := rf.Scanner.GetPathExtent()
// 				mnx, mny := float64(fRect.Min.X)/64, float64(fRect.Min.Y)/64
// 				mxx, mxy := float64(fRect.Max.X)/64, float64(fRect.Max.Y)/64
// 				fillerColor.Bounds.X, fillerColor.Bounds.Y = mnx, mny
// 				fillerColor.Bounds.W, fillerColor.Bounds.H = mxx-mnx, mxy-mny
// 			}
// 			rf.SetColor(fillerColor.GetColorFunction(svgp.FillOpacity * opacity))
// 		}
// 		rf.Draw()
// 		// default is true
// 		rf.SetWinding(true)
// 	}
// 	if svgp.linerColor != nil {
// 		r.Clear()
// 		svgp.mAdder.Adder = r
// 		lineGap := svgp.LineGap
// 		if lineGap == nil {
// 			lineGap = DefaultStyle.LineGap
// 		}
// 		lineCap := svgp.LineCap
// 		if lineCap == nil {
// 			lineCap = DefaultStyle.LineCap
// 		}
// 		leadLineCap := lineCap
// 		if svgp.LeadLineCap != nil {
// 			leadLineCap = svgp.LeadLineCap
// 		}
// 		r.SetStroke(fixed.Int26_6(svgp.LineWidth*64),
// 			fixed.Int26_6(svgp.MiterLimit*64), leadLineCap, lineCap,
// 			lineGap, svgp.LineJoin, svgp.Dash, svgp.DashOffset)
// 		svgp.Path.AddTo(&svgp.mAdder)
// 		switch linerColor := svgp.linerColor.(type) {
// 		case color.Color:
// 			r.SetColor(rasterx.ApplyOpacity(linerColor, svgp.LineOpacity*opacity))
// 		case Gradient:
// 			if linerColor.Units == rasterx.ObjectBoundingBox {
// 				fRect := r.Scanner.GetPathExtent()
// 				mnx, mny := float64(fRect.Min.X)/64, float64(fRect.Min.Y)/64
// 				mxx, mxy := float64(fRect.Max.X)/64, float64(fRect.Max.Y)/64
// 				linerColor.Bounds.X, linerColor.Bounds.Y = mnx, mny
// 				linerColor.Bounds.W, linerColor.Bounds.H = mxx-mnx, mxy-mny
// 			}
// 			r.SetColor(linerColor.GetColorFunction(svgp.LineOpacity * opacity))
// 		}
// 		r.Draw()
// 	}
// }

// ParseSVGColorNum reads the SFG color string e.g. #FBD9BD
func ParseSVGColorNum(colorStr string) (r, g, b uint8, err error) {
	colorStr = strings.TrimPrefix(colorStr, "#")
	var t uint64
	if len(colorStr) != 6 {
		// SVG specs say duplicate characters in case of 3 digit hex number
		colorStr = string([]byte{colorStr[0], colorStr[0],
			colorStr[1], colorStr[1], colorStr[2], colorStr[2]})
	}
	for _, v := range []struct {
		c *uint8
		s string
	}{
		{&r, colorStr[0:2]},
		{&g, colorStr[2:4]},
		{&b, colorStr[4:6]}} {
		t, err = strconv.ParseUint(v.s, 16, 8)
		if err != nil {
			return
		}
		*v.c = uint8(t)
	}
	return
}

// ParseSVGColor parses an SVG color string in all forms
// including all SVG1.1 names, obtained from the colornames package
func ParseSVGColor(colorStr string) (color.Color, error) {
	//_, _, _, a := curColor.RGBA()
	v := strings.ToLower(colorStr)
	if strings.HasPrefix(v, "url") { // We are not handling urls
		// and gradients and stuff at this point
		return color.NRGBA{0, 0, 0, 255}, nil
	}
	switch v {
	case "none":
		// nil signals that the function (fill or stroke) is off;
		// not the same as black
		return nil, nil
	default:
		cn, ok := colornames.Map[v]
		if ok {
			r, g, b, a := cn.RGBA()
			return color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
		}
	}
	cStr := strings.TrimPrefix(colorStr, "rgb(")
	if cStr != colorStr {
		cStr := strings.TrimSuffix(cStr, ")")
		vals := strings.Split(cStr, ",")
		if len(vals) != 3 {
			return color.NRGBA{}, errParamMismatch
		}
		var cvals [3]uint8
		var err error
		for i := range cvals {
			cvals[i], err = parseColorValue(vals[i])
			if err != nil {
				return nil, err
			}
		}
		return color.NRGBA{cvals[0], cvals[1], cvals[2], 0xFF}, nil
	}
	if colorStr[0] == '#' {
		r, g, b, err := ParseSVGColorNum(colorStr)
		if err != nil {
			return nil, err
		}
		return color.NRGBA{r, g, b, 0xFF}, nil
	}
	return nil, errParamMismatch
}

func parseColorValue(v string) (uint8, error) {
	if v[len(v)-1] == '%' {
		n, err := strconv.Atoi(strings.TrimSpace(v[:len(v)-1]))
		if err != nil {
			return 0, err
		}
		return uint8(n * 0xFF / 100), nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if n > 255 {
		n = 255
	}
	return uint8(n), err
}

func (c *IconCursor) readTransformAttr(m1 Matrix2D, k string) (Matrix2D, error) {
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

func (c *IconCursor) parseTransform(v string) (Matrix2D, error) {
	ts := strings.Split(v, ")")
	m1 := c.StyleStack[len(c.StyleStack)-1].transform
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

func (c *IconCursor) readStyleAttr(curStyle *PathStyle, k, v string) error {
	switch k {
	case "fill":
		gradient, ok := c.ReadGradURL(v, curStyle.fillerColor)
		if ok {
			curStyle.fillerColor = gradient
			break
		}
		var err error
		curStyle.fillerColor, err = ParseSVGColor(v)
		return err
	case "stroke":
		gradient, ok := c.ReadGradURL(v, curStyle.linerColor)
		if ok {
			curStyle.linerColor = gradient
			break
		}
		col, errc := ParseSVGColor(v)
		if errc != nil {
			return errc
		}
		if col != nil {
			curStyle.linerColor = col.(color.NRGBA)
		} else {
			curStyle.linerColor = nil
		}
	case "stroke-linegap":
		switch v {
		case "flat":
			curStyle.LineGap = FlatGap
		case "round":
			curStyle.LineGap = RoundGap
		case "cubic":
			curStyle.LineGap = CubicGap
		case "quadratic":
			curStyle.LineGap = QuadraticGap
		}
	case "stroke-leadlinecap":
		switch v {
		case "butt":
			curStyle.LeadLineCap = ButtCap
		case "round":
			curStyle.LeadLineCap = RoundCap
		case "square":
			curStyle.LeadLineCap = SquareCap
		case "cubic":
			curStyle.LeadLineCap = CubicCap
		case "quadratic":
			curStyle.LeadLineCap = QuadraticCap
		}
	case "stroke-linecap":
		switch v {
		case "butt":
			curStyle.LineCap = ButtCap
		case "round":
			curStyle.LineCap = RoundCap
		case "square":
			curStyle.LineCap = SquareCap
		case "cubic":
			curStyle.LineCap = CubicCap
		case "quadratic":
			curStyle.LineCap = QuadraticCap
		}
	case "stroke-linejoin":
		switch v {
		case "miter":
			curStyle.LineJoin = Miter
		case "miter-clip":
			curStyle.LineJoin = MiterClip
		case "arc-clip":
			curStyle.LineJoin = ArcClip
		case "round":
			curStyle.LineJoin = Round
		case "arc":
			curStyle.LineJoin = Arc
		case "bevel":
			curStyle.LineJoin = Bevel
		}
	case "stroke-miterlimit":
		mLimit, err := parseFloat(v, 64)
		if err != nil {
			return err
		}
		curStyle.MiterLimit = mLimit
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
		curStyle.DashOffset = dashOffset
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
			curStyle.Dash = dList
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

// PushStyle parses the style element, and push it on the style stack. Only color and opacity are supported
// for fill. Note that this parses both the contents of a style attribute plus
// direct fill and opacity attributes.
func (c *IconCursor) PushStyle(attrs []xml.Attr) error {
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
	curStyle := c.StyleStack[len(c.StyleStack)-1]
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
	c.StyleStack = append(c.StyleStack, curStyle) // Push style onto stack
	return nil
}

// splitOnCommaOrSpace returns a list of strings after splitting the input on comma and space delimiters
func splitOnCommaOrSpace(s string) []string {
	return strings.FieldsFunc(s,
		func(r rune) bool {
			return r == ',' || r == ' '
		})
}

func (c *IconCursor) readStartElement(se xml.StartElement) (err error) {
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
			c.icon.Defs[c.currentDef[0].ID] = c.currentDef
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
		if c.ErrorMode == StrictErrorMode {
			return errors.New(errStr)
		} else if c.ErrorMode == WarnErrorMode {
			log.Println(errStr)
		}
		return nil
	}
	err = df(c, se.Attr)

	if len(c.path) > 0 {
		//The cursor parsed a path from the xml element
		pathCopy := append(Path{}, c.path...)
		c.icon.SVGPaths = append(c.icon.SVGPaths,
			SvgPath{Path: pathCopy, Style: c.StyleStack[len(c.StyleStack)-1]})
		c.path = c.path[:0]
	}
	return
}

// ReadIconStream reads the Icon from the given io.Reader
// This only supports a sub-set of SVG, but
// is enough to draw many icons. If errMode is provided,
// the first value determines if the icon ignores, errors out, or logs a warning
// if it does not handle an element found in the icon file. Ignore warnings is
// the default if no ErrorMode value is provided.
func ReadIconStream(stream io.Reader, errMode ...ErrorMode) (*SvgIcon, error) {
	icon := &SvgIcon{Defs: make(map[string][]definition), Grads: make(map[string]*Gradient), Transform: Identity}
	cursor := &IconCursor{StyleStack: []PathStyle{DefaultStyle}, icon: icon}
	if len(errMode) > 0 {
		cursor.ErrorMode = errMode[0]
	}
	decoder := xml.NewDecoder(stream)
	decoder.CharsetReader = charset.NewReaderLabel
	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return icon, err
		}
		// Inspect the type of the XML token
		switch se := t.(type) {
		case xml.StartElement:
			// Reads all recognized style attributes from the start element
			// and places it on top of the styleStack
			err = cursor.PushStyle(se.Attr)
			if err != nil {
				return icon, err
			}
			err = cursor.readStartElement(se)
			if err != nil {
				return icon, err
			}
		case xml.EndElement:
			// pop style
			cursor.StyleStack = cursor.StyleStack[:len(cursor.StyleStack)-1]
			switch se.Name.Local {
			case "g":
				if cursor.inDefs {
					cursor.currentDef = append(cursor.currentDef, definition{
						Tag: "endg",
					})
				}
			case "title":
				cursor.inTitleText = false
			case "desc":
				cursor.inDescText = false
			case "defs":
				if len(cursor.currentDef) > 0 {
					cursor.icon.Defs[cursor.currentDef[0].ID] = cursor.currentDef
					cursor.currentDef = make([]definition, 0)
				}
				cursor.inDefs = false
			case "radialGradient", "linearGradient":
				cursor.inGrad = false
			}
		case xml.CharData:
			if cursor.inTitleText == true {
				icon.Titles[len(icon.Titles)-1] += string(se)
			}
			if cursor.inDescText == true {
				icon.Descriptions[len(icon.Descriptions)-1] += string(se)
			}
		}
	}
	return icon, nil
}

// ReadIcon reads the Icon from the named file
// This only supports a sub-set of SVG, but
// is enough to draw many icons. If errMode is provided,
// the first value determines if the icon ignores, errors out, or logs a warning
// if it does not handle an element found in the icon file. Ignore warnings is
// the default if no ErrorMode value is provided.
func ReadIcon(iconFile string, errMode ...ErrorMode) (*SvgIcon, error) {
	fin, errf := os.Open(iconFile)
	if errf != nil {
		return nil, errf
	}
	defer fin.Close()
	return ReadIconStream(fin, errMode...)
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

// getColor is a helper function to get the background color
// if ReadGradUrl needs it.
func getColor(clr interface{}) color.Color {
	switch c := clr.(type) {
	case Gradient: // This is a bit lazy but oh well
		for _, s := range c.Stops {
			if s.StopColor != nil {
				return s.StopColor
			}
		}
	case color.NRGBA:
		return c
	}
	return colornames.Black
}

func localizeGradIfStopClrNil(g *Gradient, defaultColor interface{}) Gradient {
	grad := *g
	for _, s := range grad.Stops {
		if s.StopColor == nil { // This means we need copy the gradient's Stop slice
			// and fill in the default color

			// Copy the stops
			stops := append([]GradStop{}, grad.Stops...)
			grad.Stops = stops
			// Use the background color when a stop color is nil
			clr := getColor(defaultColor)
			for i, s := range stops {
				if s.StopColor == nil {
					grad.Stops[i].StopColor = clr
				}
			}
			break // Only need to do this once
		}
	}
	return grad
}

// ReadGradURL reads an SVG format gradient url
// Since the context of the gradient can affect the colors
// the current fill or line color is passed in and used in
// the case of a nil stopClor value
func (c *IconCursor) ReadGradURL(v string, defaultColor interface{}) (grad Gradient, ok bool) {
	if strings.HasPrefix(v, "url(") && strings.HasSuffix(v, ")") {
		urlStr := strings.TrimSpace(v[4 : len(v)-1])
		if strings.HasPrefix(urlStr, "#") {
			var g *Gradient
			g, ok = c.icon.Grads[urlStr[1:]]
			if ok {
				grad = localizeGradIfStopClrNil(g, defaultColor)
			}
		}
	}
	return
}

// ReadGradAttr reads an SVG gradient attribute
func (c *IconCursor) ReadGradAttr(attr xml.Attr) (err error) {
	switch attr.Name.Local {
	case "gradientTransform":
		c.grad.Matrix, err = c.parseTransform(attr.Value)
	case "gradientUnits":
		switch strings.TrimSpace(attr.Value) {
		case "userSpaceOnUse":
			c.grad.Units = UserSpaceOnUse
		case "objectBoundingBox":
			c.grad.Units = ObjectBoundingBox
		}
	case "spreadMethod":
		switch strings.TrimSpace(attr.Value) {
		case "pad":
			c.grad.Spread = PadSpread
		case "reflect":
			c.grad.Spread = ReflectSpread
		case "repeat":
			c.grad.Spread = RepeatSpread
		}
	}
	return
}

type svgFunc func(c *IconCursor, attrs []xml.Attr) error

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

func svgF(c *IconCursor, attrs []xml.Attr) error {
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
func gF(*IconCursor, []xml.Attr) error { return nil } // g does nothing but push the style
func rectF(c *IconCursor, attrs []xml.Attr) error {
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
func circleF(c *IconCursor, attrs []xml.Attr) error {
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
func lineF(c *IconCursor, attrs []xml.Attr) error {
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
func polylineF(c *IconCursor, attrs []xml.Attr) error {
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
func polygonF(c *IconCursor, attrs []xml.Attr) error {
	err := polylineF(c, attrs)
	if len(c.points) > 4 {
		c.path.Stop(true)
	}
	return err
}
func pathF(c *IconCursor, attrs []xml.Attr) error {
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "d":
			err = c.CompilePath(attr.Value)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func descF(c *IconCursor, attrs []xml.Attr) error {
	c.inDescText = true
	c.icon.Descriptions = append(c.icon.Descriptions, "")
	return nil
}
func titleF(c *IconCursor, attrs []xml.Attr) error {
	c.inTitleText = true
	c.icon.Titles = append(c.icon.Titles, "")
	return nil
}
func defsF(c *IconCursor, attrs []xml.Attr) error {
	c.inDefs = true
	return nil
}
func linearGradientF(c *IconCursor, attrs []xml.Attr) error {
	var err error
	c.inGrad = true
	direction := Linear{0, 0, 1, 0}
	c.grad = &Gradient{Direction: direction, Bounds: c.icon.ViewBox, Matrix: Identity}
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "id":
			id := attr.Value
			if len(id) >= 0 {
				c.icon.Grads[id] = c.grad
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
			err = c.ReadGradAttr(attr)
		}
		if err != nil {
			return err
		}
	}
	c.grad.Direction = direction
	return nil
}

func radialGradientF(c *IconCursor, attrs []xml.Attr) error {
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
				c.icon.Grads[id] = c.grad
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
			err = c.ReadGradAttr(attr)
		}
		if err != nil {
			return err
		}
	}
	if setFx == false { // set fx to cx by default
		direction[2] = direction[0]
	}
	if setFy == false { // set fy to cy by default
		direction[3] = direction[1]
	}
	return nil
}
func stopF(c *IconCursor, attrs []xml.Attr) error {
	var err error
	if c.inGrad {
		stop := GradStop{Opacity: 1.0}
		for _, attr := range attrs {
			switch attr.Name.Local {
			case "offset":
				stop.Offset, err = readFraction(attr.Value)
			case "stop-color":
				//todo: add current color inherit
				stop.StopColor, err = ParseSVGColor(attr.Value)
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
func useF(c *IconCursor, attrs []xml.Attr) error {
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
	defs, ok := c.icon.Defs[href[1:]]
	if !ok {
		return errors.New("href ID in use statement was not found in saved defs")
	}
	for _, def := range defs {
		if def.Tag == "endg" {
			// pop style
			c.StyleStack = c.StyleStack[:len(c.StyleStack)-1]
			continue
		}
		if err = c.PushStyle(def.Attrs); err != nil {
			return err
		}
		df, ok := drawFuncs[def.Tag]
		if !ok {
			errStr := "Cannot process svg element " + def.Tag
			if c.ErrorMode == StrictErrorMode {
				return errors.New(errStr)
			} else if c.ErrorMode == WarnErrorMode {
				log.Println(errStr)
			}
			return nil
		}
		if err := df(c, def.Attrs); err != nil {
			return err
		}
		if def.Tag != "g" {
			// pop style
			c.StyleStack = c.StyleStack[:len(c.StyleStack)-1]
		}
	}
	return nil
}
