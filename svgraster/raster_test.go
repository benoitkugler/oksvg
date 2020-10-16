package svgraster

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func toPngBytes(m image.Image) ([]byte, error) {
	// Create Writer from file
	var b bytes.Buffer
	// Write the image into the buffer
	err := png.Encode(&b, m)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func saveToPngFile(filePath string, m image.Image) error {
	b, err := toPngBytes(m)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, b, os.ModePerm)
	return err
}

func renderIcon(t *testing.T, filename string) {
	filename = filepath.Join("..", "svgicon", filename)
	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("can't open svg source: %s", err)
	}
	img, err := RasterSVGIconToImage(f, nil)
	if err != nil {
		t.Fatalf("can't raster image: %s", err)
	}

	name := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	err = saveToPngFile(fmt.Sprintf("testdata_out/%s.png", name), img)
	if err != nil {
		t.Fatalf("can't saved rasterized image: %s", err)
	}

	got, err := toPngBytes(img)
	if err != nil {
		t.Fatalf("can't retrieve binary from image: %s", err)
	}

	// comparison with oksvg, requires to run its test first
	ref, err := ioutil.ReadFile(fmt.Sprintf("../../../srwiley/oksvg/testdata/%s.svg.png", name))
	if err != nil {
		t.Fatalf("can't load reference image: %s", err)
	}

	if !bytes.Equal(got, ref) {
		t.Errorf("image %s is different from expectation", filename)
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
