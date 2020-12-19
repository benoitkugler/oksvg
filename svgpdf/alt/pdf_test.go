package alt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func renderIcon(t *testing.T, filename string) {
	filename = filepath.Join("..", "..", "svgicon", filename)
	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("can't open svg source: %s", err)
	}

	name := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	err = RenderSVGIconToPDF(f, fmt.Sprintf("testdata_out/%s.pdf", name))
	if err != nil {
		t.Fatalf("can't raster image: %s", err)
	}
}

func TestLandscapeIcons(t *testing.T) {
	for _, p := range [...]string{
		"beach", "cape", "iceberg", "island",
		"mountains", "sea", "trees", "village"} {
		renderIcon(t, "testdata/landscapeIcons/"+p+".svg")
	}
}

func TestSportsIcons(t *testing.T) {
	for _, p := range [...]string{
		"archery", "fencing", "rugby_sevens",
		"artistic_gymnastics", "football", "sailing",
		"athletics", "golf", "shooting",
		"badminton", "handball", "swimming",
		"basketball", "hockey", "synchronised_swimming",
		"beach_volleyball", "judo", "table_tennis",
		"boxing", "marathon_swimming", "taekwondo",
		"canoe_slalom", "modern_pentathlon", "tennis",
		"canoe_sprint", "olympic_medal_bronze", "trampoline_gymnastics",
		"cycling_bmx", "olympic_medal_gold", "triathlon",
		"cycling_mountain_bike", "olympic_medal_silver", "trophy",
		"cycling_road", "olympic_torch", "volleyball",
		"cycling_track", "water_polo",
		"diving", "rhythmic_gymnastics", "weightlifting",
		"equestrian", "rowing", "wrestling"} {
		renderIcon(t, "testdata/sportsIcons/"+p+".svg")
	}
}

func TestTestIcons(t *testing.T) {
	for _, p := range [...]string{
		"astronaut", "jupiter", "lander", "school-bus", "telescope", "content-cut-light", "defs",
		"24px"} {
		renderIcon(t, "testdata/testIcons/"+p+".svg")
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
		renderIcon(t, "testdata/"+p)
	}
}
