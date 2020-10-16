package svgicon

import (
	"encoding/xml"
	"errors"
	"strings"

	"golang.org/x/image/math/fixed"
)

func init() {
	// avoids cyclical static declaration
	// called on package initialization
	drawFuncs["use"] = useF
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
			width, err = parseBasicFloat(attr.Value)
		case "height":
			height, err = parseBasicFloat(attr.Value)
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
			x, err = c.parseUnit(attr.Value, widthPercentage)
		case "y":
			y, err = c.parseUnit(attr.Value, heightPercentage)
		case "width":
			w, err = c.parseUnit(attr.Value, widthPercentage)
		case "height":
			h, err = c.parseUnit(attr.Value, heightPercentage)
		case "rx":
			rx, err = c.parseUnit(attr.Value, widthPercentage)
		case "ry":
			ry, err = c.parseUnit(attr.Value, heightPercentage)
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
			cx, err = c.parseUnit(attr.Value, widthPercentage)
		case "cy":
			cy, err = c.parseUnit(attr.Value, heightPercentage)
		case "r":
			rx, err = c.parseUnit(attr.Value, diagPercentage)
			ry = rx
		case "rx":
			rx, err = c.parseUnit(attr.Value, widthPercentage)
		case "ry":
			ry, err = c.parseUnit(attr.Value, heightPercentage)
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
			x1, err = c.parseUnit(attr.Value, widthPercentage)
		case "x2":
			x2, err = c.parseUnit(attr.Value, widthPercentage)
		case "y1":
			y1, err = c.parseUnit(attr.Value, heightPercentage)
		case "y2":
			y2, err = c.parseUnit(attr.Value, heightPercentage)
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
	c.grad.Direction = direction
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
				stop.Opacity, err = parseBasicFloat(attr.Value)
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
			x, err = c.parseUnit(attr.Value, widthPercentage)
		case "y":
			y, err = c.parseUnit(attr.Value, heightPercentage)
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
			return c.handleError(errStr)
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
