# SVG parser and renderer, written in Go

This package is a fork of github.com/srwiley/oksvg, the core logic is similar.
However, it adds the possiblity of using differents rendered target, by splitting
the parsing and processing of the SVG file from its actual drawing.
Of course, you can still raster an icon into a PNG image (using svgraster, which itself uses github.com/srwiley/rasterx), and it will be possible to use a PDF backend (using svgpdf, still WIP).
