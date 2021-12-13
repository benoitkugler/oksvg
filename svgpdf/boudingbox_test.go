package svgpdf

import (
	"image/color"
	"math/rand"
	"testing"

	"github.com/benoitkugler/pdf/contentstream"
	"golang.org/x/image/math/fixed"
)

func randPoint(offsetx, offsety int) fixed.Point26_6 {
	x, y := rand.Intn(1100), rand.Intn(1000)
	return fixed.Point26_6{X: fixed.Int26_6(x + offsetx), Y: fixed.Int26_6(y + offsety)}
}

func generateDrawCurve(p pather, order int, offsetx, offsety int) bezier {
	a := randPoint(offsetx, offsety)
	b := randPoint(offsetx, offsety)
	p.Start(a)
	switch order {
	case 1:
		p.Line(b)
		return line{a, b}
	case 2:
		c := randPoint(offsetx, offsety)
		p.QuadBezier(b, c)
		return quadBezier{a, b, c}
	case 3:
		c := randPoint(offsetx, offsety)
		d := randPoint(offsetx, offsety)
		p.CubeBezier(b, c, d)
		return cubicBezier{a, b, c, d}
	default:
		return nil
	}
}

func drawOneBox(p pather, order int, offsetx, offsety int) {
	p.pdf.SetStrokeAlpha(1)

	curve := generateDrawCurve(p, order, offsetx, offsety)
	p.Stop(true)
	p.pdf.Ops(contentstream.OpCloseStroke{})

	rect := computeBoundingBox(curve)
	p.pdf.SetFillAlpha(0.2)
	drawRectange(p.pdf, rect)
}

func TestBoudindBox(t *testing.T) {
	ap := contentstream.NewAppearance(500, 500)
	p := pather{pdf: &ap}
	p.pdf.SetColorFill(color.RGBA{50, 50, 50, 255})
	p.pdf.SetColorStroke(color.RGBA{50, 0, 50, 255})
	p.pdf.Ops(contentstream.OpSetLineWidth{W: 0.1})

	for i := range [10]int{} {
		for j := range [10]int{} {
			drawOneBox(p, 1+rand.Intn(3), i<<10+500, j<<10+500)
		}
	}

	if err := saveApperanceToFile(&ap, "testdata_out/bezier_bbox.pdf"); err != nil {
		t.Error(err)
	}
}

func drawRectange(p *contentstream.Appearance, rect fixed.Rectangle26_6) {
	xmin, ymin := fixedTof(rect.Min)
	xmax, ymax := fixedTof(rect.Max)
	p.Ops(
		contentstream.OpRectangle{X: xmin, Y: ymin, W: xmax - xmin, H: ymax - ymin},
		contentstream.OpFill{},
	)
}

func TestAggregateBoxes(t *testing.T) {
	pdf := contentstream.NewAppearance(100, 100)
	for i := range [6]int{} {
		for j := range [9]int{} {
			min1 := randPoint(i<<11+100, j<<11+100)
			diff1 := randPoint(100, 100)
			max1 := fixed.Point26_6{X: min1.X + diff1.X, Y: min1.Y + diff1.Y}
			rect1 := fixed.Rectangle26_6{Min: min1, Max: max1}

			min2 := randPoint(i<<11+100, j<<11+100)
			diff2 := randPoint(100, 100)
			max2 := fixed.Point26_6{X: min2.X + diff2.X, Y: min2.Y + diff2.Y}
			rect2 := fixed.Rectangle26_6{Min: min2, Max: max2}

			res := rect1.Union(rect2)

			pdf.SetColorFill(color.RGBA{10, 10, 10, 255})
			pdf.SetFillAlpha(0.4)
			drawRectange(&pdf, rect1)
			drawRectange(&pdf, rect2)
			pdf.SetColorFill(color.RGBA{10, 100, 10, 255})
			pdf.SetFillAlpha(0.2)
			drawRectange(&pdf, res)
		}
	}

	if err := saveApperanceToFile(&pdf, "testdata_out/aggregate_bbox.pdf"); err != nil {
		t.Error(err)
	}
}
