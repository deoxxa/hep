// Copyright ©2016 The go-hep Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hplot

import (
	"errors"
	"fmt"
	"image/color"
	"math"

	"go-hep.org/x/hep/hbook"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// H1D implements the plotter.Plotter interface,
// drawing a histogram of the data.
type H1D struct {
	// Hist is the histogramming data
	Hist *hbook.H1D

	// FillColor is the color used to fill each
	// bar of the histogram.  If the color is nil
	// then the bars are not filled.
	FillColor color.Color

	// LineStyle is the style of the outline of each
	// bar of the histogram.
	draw.LineStyle

	// LogY allows rendering with a log-scaled Y axis.
	// When enabled, histogram bins with no entries will be discarded from
	// the histogram's DataRange.
	// The lowest Y value for the DataRange will be corrected to leave an
	// arbitrary amount of height for the smallest bin entry so it is visible
	// on the final plot.
	LogY bool

	// InfoStyle is the style of infos displayed for
	// the histogram (entries, mean, rms)
	Infos HInfos
}

type HInfoStyle uint32

const (
	HInfoNone    HInfoStyle = 0
	HInfoEntries HInfoStyle = 1 << iota
	HInfoMean
	HInfoRMS
	HInfoStdDev
	HInfoSummary HInfoStyle = HInfoEntries | HInfoMean | HInfoStdDev
)

type HInfos struct {
	Style HInfoStyle
}

// NewH1FromXYer returns a new histogram
// that represents the distribution of values
// using the given number of bins.
//
// Each y value is assumed to be the frequency
// count for the corresponding x.
//
// It panics if the number of bins is non-positive.
func NewH1FromXYer(xy plotter.XYer, n int) *H1D {
	if n <= 0 {
		panic(errors.New("hplot: histogram with non-positive number of bins"))
	}
	h := newHistFromXYer(xy, n)
	return NewH1D(h)
}

// NewH1FromValuer returns a new histogram, as in
// NewH1FromXYer, except that it accepts a plotter.Valuer
// instead of an XYer.
func NewH1FromValuer(vs plotter.Valuer, n int) *H1D {
	return NewH1FromXYer(unitYs{vs}, n)
}

type unitYs struct {
	plotter.Valuer
}

func (u unitYs) XY(i int) (float64, float64) {
	return u.Value(i), 1.0
}

// NewH1D returns a new histogram, as in
// NewH1DFromXYer, except that it accepts a hbook.H1D
// instead of a plotter.XYer
func NewH1D(h *hbook.H1D) *H1D {
	return &H1D{
		Hist:      h,
		LineStyle: plotter.DefaultLineStyle,
	}
}

// DataRange returns the minimum and maximum X and Y values
func (h *H1D) DataRange() (xmin, xmax, ymin, ymax float64) {
	if !h.LogY {
		return h.Hist.DataRange()
	}

	xmin = math.Inf(+1)
	xmax = math.Inf(-1)
	ymin = math.Inf(+1)
	ymax = math.Inf(-1)
	ylow := math.Inf(+1) // ylow will hold the smallest non-zero y value.
	for _, bin := range h.Hist.Binning.Bins {
		if bin.XMax() > xmax {
			xmax = bin.XMax()
		}
		if bin.XMin() < xmin {
			xmin = bin.XMin()
		}
		if bin.SumW() > ymax {
			ymax = bin.SumW()
		}
		if bin.SumW() < ymin {
			ymin = bin.SumW()
		}
		if bin.SumW() != 0 && bin.SumW() < ylow {
			ylow = bin.SumW()
		}
	}

	if ymin == 0 && !math.IsInf(ylow, +1) {
		// Reserve a bit of space for the smallest bin to be displayed still.
		ymin = ylow * 0.5
	}

	return
}

// Plot implements the Plotter interface, drawing a line
// that connects each point in the Line.
func (h *H1D) Plot(c draw.Canvas, p *plot.Plot) {
	trX, trY := p.Transforms(&c)
	var pts []vg.Point
	hist := h.Hist
	bins := h.Hist.Binning.Bins
	nbins := len(bins)

	yfct := func(sumw float64) (ymin, ymax vg.Length) {
		return trY(0), trY(sumw)
	}
	if h.LogY {
		yfct = func(sumw float64) (ymin, ymax vg.Length) {
			ymin = c.Min.Y
			ymax = c.Min.Y
			if 0 != sumw {
				ymax = trY(sumw)
			}
			return ymin, ymax
		}
	}

	for i, bin := range bins {
		xmin := trX(bin.XMin())
		xmax := trX(bin.XMax())
		sumw := bin.SumW()
		ymin, ymax := yfct(sumw)
		switch i {
		case 0:
			pts = append(pts, vg.Point{X: xmin, Y: ymin})
			pts = append(pts, vg.Point{X: xmin, Y: ymax})
			pts = append(pts, vg.Point{X: xmax, Y: ymax})

		case nbins - 1:
			lft := bins[i-1]
			xlft := trX(lft.XMax())
			_, ylft := yfct(lft.SumW())
			pts = append(pts, vg.Point{X: xlft, Y: ylft})
			pts = append(pts, vg.Point{X: xmin, Y: ymax})
			pts = append(pts, vg.Point{X: xmax, Y: ymax})
			pts = append(pts, vg.Point{X: xmax, Y: ymin})

		default:
			lft := bins[i-1]
			xlft := trX(lft.XMax())
			_, ylft := yfct(lft.SumW())
			pts = append(pts, vg.Point{X: xlft, Y: ylft})
			pts = append(pts, vg.Point{X: xmin, Y: ymax})
			pts = append(pts, vg.Point{X: xmax, Y: ymax})
		}
	}

	if h.FillColor != nil {
		c.FillPolygon(h.FillColor, c.ClipPolygonXY(pts))
	}
	c.StrokeLines(h.LineStyle, c.ClipLinesXY(pts)...)

	if h.Infos.Style != HInfoNone {
		fnt, err := vg.MakeFont(DefaultStyle.Fonts.Name, DefaultStyle.Fonts.Tick.Size)
		if err == nil {
			sty := draw.TextStyle{Font: fnt}
			legend := histLegend{
				ColWidth:  DefaultStyle.Fonts.Tick.Size,
				TextStyle: sty,
			}

			for i := uint32(0); i < 32; i++ {
				switch h.Infos.Style & (1 << i) {
				case HInfoEntries:
					legend.Add("Entries", hist.Entries())
				case HInfoMean:
					legend.Add("Mean", hist.XMean())
				case HInfoRMS:
					legend.Add("RMS", hist.XRMS())
				case HInfoStdDev:
					legend.Add("Std Dev", hist.XStdDev())
				default:
				}
			}
			legend.Top = true

			legend.draw(c)
		}
	}
}

// GlyphBoxes returns a slice of GlyphBoxes,
// one for each of the bins, implementing the
// plot.GlyphBoxer interface.
func (h *H1D) GlyphBoxes(p *plot.Plot) []plot.GlyphBox {
	bins := h.Hist.Binning.Bins
	bs := make([]plot.GlyphBox, 0, len(bins))
	for i := range bins {
		bin := bins[i]
		y := bin.SumW()
		if h.LogY && y == 0 {
			continue
		}
		var box plot.GlyphBox
		xmin := bin.XMin()
		w := p.X.Norm(bin.XWidth())
		box.X = p.X.Norm(xmin + 0.5*w)
		box.Y = p.Y.Norm(y)
		box.Rectangle.Min.X = vg.Length(xmin - 0.5*w)
		box.Rectangle.Min.Y = vg.Length(y - 0.5*w)
		box.Rectangle.Max.X = vg.Length(w)
		box.Rectangle.Max.Y = vg.Length(0)

		r := vg.Points(5)
		box.Rectangle.Min = vg.Point{X: 0, Y: 0}
		box.Rectangle.Max = vg.Point{X: 0, Y: r}
		bs = append(bs, box)
	}
	return bs
}

// Normalize normalizes the histogram so that the
// total area beneath it sums to a given value.
// func (h *Histogram) Normalize(sum float64) {
// 	mass := 0.0
// 	for _, b := range h.Bins {
// 		mass += b.Weight
// 	}
// 	for i := range h.Bins {
// 		h.Bins[i].Weight *= sum / (h.Width * mass)
// 	}
// }

// Thumbnail draws a rectangle in the given style of the histogram.
func (h *H1D) Thumbnail(c *draw.Canvas) {
	ymin := c.Min.Y
	ymax := c.Max.Y
	xmin := c.Min.X
	xmax := c.Max.X

	pts := []vg.Point{
		{X: xmin, Y: ymin},
		{X: xmax, Y: ymin},
		{X: xmax, Y: ymax},
		{X: xmin, Y: ymax},
	}
	if h.FillColor != nil {
		c.FillPolygon(h.FillColor, c.ClipPolygonXY(pts))
	}
	pts = append(pts, vg.Point{X: xmin, Y: ymin})
	c.StrokeLines(h.LineStyle, c.ClipLinesXY(pts)...)
}

func newHistFromXYer(xys plotter.XYer, n int) *hbook.H1D {
	xmin, xmax := plotter.Range(plotter.XValues{XYer: xys})
	h := hbook.NewH1D(n, xmin, xmax)

	for i := 0; i < xys.Len(); i++ {
		x, y := xys.XY(i)
		h.Fill(x, y)
	}

	return h
}

// A Legend gives a description of the meaning of different
// data elements of the plot.  Each legend entry has a name
// and a thumbnail, where the thumbnail shows a small
// sample of the display style of the corresponding data.
type histLegend struct {
	// TextStyle is the style given to the legend
	// entry texts.
	draw.TextStyle

	// Padding is the amount of padding to add
	// betweeneach entry of the legend.  If Padding
	// is zero then entries are spaced based on the
	// font size.
	Padding vg.Length

	// Top and Left specify the location of the legend.
	// If Top is true the legend is located along the top
	// edge of the plot, otherwise it is located along
	// the bottom edge.  If Left is true then the legend
	// is located along the left edge of the plot, and the
	// text is positioned after the icons, otherwise it is
	// located along the right edge and the text is
	// positioned before the icons.
	Top, Left bool

	// XOffs and YOffs are added to the legend's
	// final position.
	XOffs, YOffs vg.Length

	// ColWidth is the width of legend names
	ColWidth vg.Length

	// entries are all of the legendEntries described
	// by this legend.
	entries []legendEntry
}

// A legendEntry represents a single line of a legend, it
// has a name and an icon.
type legendEntry struct {
	// text is the text associated with this entry.
	text string

	// value is the value associated with this entry
	value string
}

// draw draws the legend to the given canvas.
func (l *histLegend) draw(c draw.Canvas) {
	textx := c.Min.X
	hdr := l.entryWidth() //+ l.TextStyle.Width(" ")
	l.ColWidth = hdr
	if !l.Left {
		textx = c.Max.X - l.ColWidth
	}
	textx += l.XOffs

	enth := l.entryHeight()
	y := c.Max.Y - enth
	if !l.Top {
		y = c.Min.Y + (enth+l.Padding)*(vg.Length(len(l.entries))-1)
	}
	y += l.YOffs

	colx := &draw.Canvas{
		Canvas: c.Canvas,
		Rectangle: vg.Rectangle{
			Min: vg.Point{X: c.Min.X, Y: y},
			Max: vg.Point{X: 2 * l.ColWidth, Y: enth},
		},
	}
	for _, e := range l.entries {
		yoffs := (enth - l.TextStyle.Height(e.text)) / 2
		txt := l.TextStyle
		txt.XAlign = draw.XLeft
		c.FillText(txt, vg.Point{X: textx - hdr, Y: colx.Min.Y + yoffs}, e.text)
		txt.XAlign = draw.XRight
		c.FillText(txt, vg.Point{X: textx + hdr, Y: colx.Min.Y + yoffs}, e.value)
		colx.Min.Y -= enth + l.Padding
	}

	bboxXmin := textx - hdr - l.TextStyle.Width(" ")
	bboxXmax := c.Max.X
	bboxYmin := colx.Min.Y + enth
	bboxYmax := c.Max.Y
	bbox := []vg.Point{
		{X: bboxXmin, Y: bboxYmax},
		{X: bboxXmin, Y: bboxYmin},
		{X: bboxXmax, Y: bboxYmin},
		{X: bboxXmax, Y: bboxYmax},
		{X: bboxXmin, Y: bboxYmax},
	}
	c.StrokeLines(plotter.DefaultLineStyle, bbox)
}

// entryHeight returns the height of the tallest legend
// entry text.
func (l *histLegend) entryHeight() (height vg.Length) {
	for _, e := range l.entries {
		if h := l.TextStyle.Height(e.text); h > height {
			height = h
		}
	}
	return
}

// entryWidth returns the width of the largest legend
// entry text.
func (l *histLegend) entryWidth() (width vg.Length) {
	for _, e := range l.entries {
		if w := l.TextStyle.Width(e.value); w > width {
			width = w
		}
	}
	return
}

// Add adds an entry to the legend with the given name.
// The entry's thumbnail is drawn as the composite of all of the
// thumbnails.
func (l *histLegend) Add(name string, value interface{}) {
	str := ""
	switch value.(type) {
	case float64, float32:
		str = fmt.Sprintf("%6.4g ", value)
	default:
		str = fmt.Sprintf("%v ", value)
	}
	l.entries = append(l.entries, legendEntry{text: name, value: str})
}
