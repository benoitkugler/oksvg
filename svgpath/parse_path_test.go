package svgpath

import "testing"

func TestReadFloat(t *testing.T) {
	c := new(pathCursor)

	fStr := "23.4.56"
	err := c.readFloat(fStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.points) != 2 || c.points[0] != 23.4 || c.points[1] != 0.56 {
		t.Error("read float failed", fStr, c.points)
	}

	c.points = nil

	fStr = "23.4"
	err = c.readFloat(fStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.points) != 1 || c.points[0] != 23.4 {
		t.Error("read float failed", fStr)
	}
	c.points = nil

	fStr = "23"
	err = c.readFloat(fStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.points) != 1 || c.points[0] != 23.0 {
		t.Error("read float failed", fStr, c.points)
	}

	c.points = nil
	fStr = ".4"
	err = c.readFloat(fStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.points) != 1 || c.points[0] != 0.4 {
		t.Error("read float failed", fStr, c.points)
	}

	c.points = nil
	fStr = "23.4.56.67.32"
	err = c.readFloat(fStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.points) != 4 || c.points[0] != 23.4 || c.points[1] != 0.56 ||
		c.points[2] != 0.67 || c.points[3] != 0.32 {
		t.Error("read float failed", fStr, c.points)
	}

}
