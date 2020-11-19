# SVG parser and renderer, written in Go

This package is a fork of [github.com/srwiley/oksvg](https://github.com/srwiley/oksvg): most of the code is copied from it and the core logic is the same.
However, it adds the possiblity of using differents rendered target, by splitting
the parsing and processing of the SVG file from its actual drawing.
Of course, you can still raster an icon into a PNG image (using `svgraster.RasterSVGIconToImage`, which itself uses [github.com/srwiley/rasterx](https://github.com/srwiley/rasterx)), and it will be possible to use a PDF backend (using svgpdf and gofpdf, still WIP).

Other backends should be easy to add, by implementing the `oksvg.Driver` interface.

See [Godoc](https://godoc.org/github.com/benoitkugler/oksvg) for more details.
