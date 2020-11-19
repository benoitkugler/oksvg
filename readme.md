# SVG parser and renderer, written in Go

This package is a fork of [github.com/srwiley/oksvg](https://github.com/srwiley/oksvg): most of the code is copied from it and the core logic is the same.

However, it adds the possiblity of using differents rendering target, by splitting
the parsing and processing of the SVG file from its actual drawing.

Of course, you can still raster an icon into a PNG image (using `svgraster.RasterSVGIconToImage`, built on [github.com/srwiley/rasterx](https://github.com/srwiley/rasterx)), and can also use a PDF backend (using `svgpdf.RenderSVGIconToPDF`, built on [github.com/phpdave11/gofpdf](https://github.com/phpdave11/gofpdf)). Be aware that the PDF backend is still experimental and is missing features like miter limit control and gradient support.

Other backends should be easy to add, by implementing the `oksvg.Driver` interface.

See [Godoc](https://godoc.org/github.com/benoitkugler/oksvg) for more details.
