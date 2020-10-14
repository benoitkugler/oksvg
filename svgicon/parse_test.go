package svgicon

import (
	"testing"
)

func parseIcon(t *testing.T, iconPath string) {
	_, errSvg := ReadIcon(iconPath, WarnErrorMode)
	if errSvg != nil {
		t.Error(errSvg)
	}
}

func TestLandscapeIcons(t *testing.T) {
	for _, p := range []string{
		"beach", "cape", "iceberg", "island",
		"mountains", "sea", "trees", "village"} {
		parseIcon(t, "testdata/landscapeIcons/"+p+".svg")
	}
}

func TestTestIcons(t *testing.T) {
	for _, p := range []string{
		"astronaut", "jupiter", "lander", "school-bus", "telescope", "content-cut-light", "defs",
		"24px"} {
		parseIcon(t, "testdata/testIcons/"+p+".svg")
	}
}

func TestStrokeIcons(t *testing.T) {
	for _, p := range []string{
		"OpacityStrokeDashTest.svg",
		"OpacityStrokeDashTest2.svg",
		"OpacityStrokeDashTest3.svg",
		"TestShapes.svg",
		"TestShapes2.svg",
		"TestShapes3.svg",
		"TestShapes4.svg",
		"TestShapes5.svg",
		"TestShapes6.svg",
	} {
		parseIcon(t, "testdata/"+p)
	}
}
