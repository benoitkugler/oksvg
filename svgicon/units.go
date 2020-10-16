package svgicon

import (
	"math"
	"strconv"
	"strings"
)

var root2 = math.Sqrt(2)

type Unite uint8

// absoluteUnits are suffixes sometimes applied to the width and height attributes
// of the svg element.
const (
	Px Unite = iota
	Cm
	Mm
	Pt
	In
	Q
	Pc
	Perc
)

var absoluteUnits = [...]string{Px: "px", Cm: "cm", Mm: "mm", Pt: "pt", In: "in", Q: "Q", Pc: "pc", Perc: "%"}

var toPx = [...]float64{Px: 1, Cm: 96. / 2.54, Mm: 9.6 / 2.54, Pt: 96. / 72., In: 96., Q: 96. / 40. / 2.54, Pc: 96. / 6., Perc: 1}

// look for an absolute unit, or nothing (considered as pixels)
// % is also supported
func findUnit(s string) (unite Unite, value string) {
	s = strings.TrimSpace(s)
	for unite, suffix := range absoluteUnits {
		if strings.HasSuffix(s, suffix) {
			valueS := strings.TrimSpace(strings.TrimSuffix(s, suffix))
			return Unite(unite), valueS
		}
	}
	return Px, s
}

// convert the unite to pixels. Return true if it is a %
func parseUnit(s string) (float64, bool, error) {
	unite, value := findUnit(s)
	out, err := strconv.ParseFloat(value, 64)
	return out * toPx[unite], unite == Perc, err
}

type percentageReference uint8

const (
	widthPercentage percentageReference = iota
	heightPercentage
	diagPercentage
)

// parseUnit converts a length with a unit into its value in 'px'
// percentage are supported, and refer to the current ViewBox
func (c *iconCursor) parseUnit(s string, asPerc percentageReference) (float64, error) {
	value, isPercentage, err := parseUnit(s)
	if err != nil {
		return 0, err
	}
	if isPercentage {
		w, h := c.icon.ViewBox.W, c.icon.ViewBox.H
		switch asPerc {
		case widthPercentage:
			return value / 100 * w, nil
		case heightPercentage:
			return value / 100 * h, nil
		case diagPercentage:
			normalizedDiag := math.Sqrt(w*w+h*h) / root2
			return value / 100 * normalizedDiag, nil
		}
	}
	return value, nil
}

func parseBasicFloat(s string) (float64, error) {
	value, _, err := parseUnit(s)
	return value, err
}
