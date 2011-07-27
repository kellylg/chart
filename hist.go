package chart

import (
	"fmt"
	"math"
	//	"os"
	//	"strings"
)


type HistChartData struct {
	Name    string
	Style   DataStyle
	Samples []float64
}


type HistChart struct {
	XRange, YRange Range // Lower limit of YRange is fixed and not available for input
	Title          string
	Xlabel, Ylabel string
	Key            Key
	Horizontal     bool // Display is horizontal bars
	Stacked        bool // Display different data sets ontop of each other
	ShowVal        bool
	Data           []HistChartData
	FirstBin       float64 // center of the first (lowest bin)
	BinWidth       float64
	Gap            float64   // gap between bins in (bin-width units): 0<=Gap<1,
	Sep            float64   // separation of bars in one bin (in bar width units) -1<Sep<1
	TBinWidth      TimeDelta // for time XRange
}

func (c *HistChart) AddData(name string, data []float64, style DataStyle) {
	c.Data = append(c.Data, HistChartData{name, style, data})
	c.Key.Entries = append(c.Key.Entries, KeyEntry{Text: name, Style: style})

	if len(c.Data) == 1 { // first data set 
		c.XRange.DataMin = data[0]
		c.XRange.DataMax = data[0]
	}
	for _, d := range data {
		if d < c.XRange.DataMin {
			c.XRange.DataMin = d
		} else if d > c.XRange.DataMax {
			c.XRange.DataMax = d
		}
	}
	c.XRange.Min = c.XRange.DataMin
	c.XRange.Max = c.XRange.DataMax
}

func (c *HistChart) AddDataInt(name string, data []int, style DataStyle) {
	fdata := make([]float64, len(data))
	for i, d := range data {
		fdata[i] = float64(d)
	}
	c.AddData(name, fdata, style)
}

func (c *HistChart) AddDataGeneric(name string, data []Value, style DataStyle) {
	fdata := make([]float64, len(data))
	for i, d := range data {
		fdata[i] = d.XVal()
	}
	c.AddData(name, fdata, style)
}

func (hc *HistChart) PlotTxt(w, h int) string {
	width, leftm, height, topm, kb, numxtics, numytics := LayoutTxt(w, h, hc.Title, hc.Xlabel, hc.Ylabel, hc.XRange.TicSetting.Hide, hc.YRange.TicSetting.Hide, &hc.Key, 1, 1)

	// Outside bound ranges for histograms are nicer
	leftm, width = leftm+2, width-2
	topm, height = topm, height-1

	hc.XRange.Setup(numxtics, numxtics+1, width, leftm, false)

	// TODO(vodo) BinWidth might be input....
	hc.BinWidth = hc.XRange.TicSetting.Delta
	binCnt := int((hc.XRange.Max-hc.XRange.Min)/hc.BinWidth + 0.5)
	hc.FirstBin = hc.XRange.Min + hc.BinWidth/2

	counts := make([][]int, len(hc.Data))
	hc.YRange.DataMin = 0
	max := 0
	for i, data := range hc.Data {
		count := make([]int, binCnt)
		for _, x := range data.Samples {
			bin := int((x - hc.XRange.Min) / hc.BinWidth)
			count[bin] = count[bin] + 1
			if count[bin] > max {
				max = count[bin]
			}
		}
		counts[i] = count
		// fmt.Printf("Count: %v\n", count)
	}
	if hc.Stacked { // recalculate max
		max = 0
		for bin := 0; bin < binCnt; bin++ {
			sum := 0
			for i := range counts {
				sum += counts[i][bin]
			}
			// fmt.Printf("sum of bin %d = %d\n", bin, sum)
			if sum > max {
				max = sum
			}
		}
	}
	hc.YRange.DataMax = float64(max)
	hc.YRange.Setup(numytics, numytics+2, height, topm, true)

	tb := NewTextBuf(w, h)

	if hc.Title != "" {
		tb.Text(width/2+leftm, 0, hc.Title, 0)
	}

	TxtXRange(hc.XRange, tb, topm+height+1, 0, hc.Xlabel, 0)
	TxtYRange(hc.YRange, tb, leftm-2, 0, hc.Ylabel, 0)

	xf := hc.XRange.Data2Screen
	yf := hc.YRange.Data2Screen

	numSets := len(hc.Data)
	for i, tic := range hc.XRange.Tics {
		xs := xf(tic.Pos)
		// tb.Put(xs, topm+height+1, '+')
		// tb.Text(lx, topm+height+2, tic.Label, 0)

		if i == 0 {
			continue
		}

		last := hc.XRange.Tics[i-1]
		lasts := xf(last.Pos)

		var blockW int
		if hc.Stacked {
			blockW = xs - lasts - 1
		} else {
			blockW = int(float64(xs-lasts-numSets) / float64(numSets))
		}
		// fmt.Printf("blockW= %d\n", blockW)

		center := (tic.Pos + last.Pos) / 2
		bin := int((center - hc.XRange.Min) / hc.BinWidth)
		xs = lasts
		lastCnt := 0
		y0 := yf(0)

		minCnt := int(math.Fabs(hc.YRange.Screen2Data(0)-hc.YRange.Screen2Data(1)) / 2)

		for d, _ := range hc.Data {
			cnt := counts[d][bin]
			y := yf(float64(lastCnt + cnt))
			if cnt > minCnt {
				fill := Symbol[d%len(Symbol)]

				tb.Block(xs+1, y, blockW, y0-y, fill)

				if hc.ShowVal {
					lab := fmt.Sprintf("%d", cnt)
					if blockW-len(lab) >= 4 {
						lab = " " + lab + " "
					}
					xlab := xs + blockW/2 + 1 // hc.XRange.Data2Screen(center)
					if blockW%2 == 1 {
						xlab++
					}
					ylab := y - 1
					if numSets > 1 {
						ylab = yf(float64(lastCnt) + float64(cnt)/2)
					}
					tb.Text(xlab, ylab, lab, 0)
					// fmt.Printf("Set %d: %s at %d\n", d, lab, ylab)
				}
			}
			if !hc.Stacked {
				xs += blockW + 1
			} else {
				lastCnt += cnt
				y0 = y
			}
		}
	}

	if kb != nil {
		tb.Paste(hc.Key.X, hc.Key.Y, kb)
	}

	return tb.String()
}


// G = B * Gf;  S = W *Sf
// W = (B(1-Gf))/(N-(N-1)Sf)
// S = (B(1-Gf))/(N/Sf - (N-1))
// N   Gf    Sf
// 2   1/4  1/3
// 3   1/5  1/2
// 4   1/6  2/3
// 5   1/6  3/4
func (c *HistChart) widthFactor() (gf, sf float64) {
	if c.Stacked {
		gf = c.Gap
		sf = -1
		return
	}

	switch len(c.Data) {
	case 1:
		gf = c.Gap
		sf = -1
		return
	case 2:
		gf = 1.0 / 4.0
		sf = -1.0 / 3.0
	case 3:
		gf = 1.0 / 5.0
		sf = -1.0 / 2.0
	case 4:
		gf = 1.0 / 6.0
		sf = -2.0 / 3.0
	default:
		gf = 1.0 / 6.0
		sf = -2.0 / 4.0
	}

	if c.Gap != 0 {
		gf = c.Gap
	}
	if c.Sep != 0 {
		sf = c.Sep
	}
	return
}


func (c *HistChart) binify(binStart, binWidth float64, binCnt int) (counts [][]int, max int) {
	x2bin := func(x float64) int { return int((x - binStart) / binWidth) }

	counts = make([][]int, len(c.Data)) // counts[d][b] is count of bin b in dataset d
	max = 0
	for i, data := range c.Data {
		count := make([]int, binCnt)
		for _, x := range data.Samples {
			bin := x2bin(x)
			if bin < 0 || bin >= binCnt {
				continue
			}
			count[bin] = count[bin] + 1
			if count[bin] > max {
				max = count[bin]
			}
		}
		counts[i] = count
		// fmt.Printf("Count: %v\n", count)
	}
	fmt.Printf("Maximum count: %d\n", max)
	if c.Stacked { // recalculate max
		max = 0
		for bin := 0; bin < binCnt; bin++ {
			sum := 0
			for i := range counts {
				sum += counts[i][bin]
			}
			// fmt.Printf("sum of bin %d = %d\n", bin, sum)
			if sum > max {
				max = sum
			}
		}
		fmt.Printf("Re-Maxed to count: %d\n", max)
	}
	return
}


func (c *HistChart) Plot(g Graphics) {
	layout := Layout(g, c.Title, c.XRange.Label, c.YRange.Label,
		c.XRange.TicSetting.Hide, c.YRange.TicSetting.Hide, &c.Key)
	fw, fh, _ := g.FontMetrics(DataStyle{})
	fw += 0

	width, height := layout.Width, layout.Height
	topm, leftm := layout.Top, layout.Left
	numxtics, numytics := layout.NumXtics, layout.NumYtics

	// Outside bound ranges for histograms are nicer
	leftm, width = leftm+int(2*fw), width-int(2*fw)
	topm, height = topm, height-int(1*fh)

	c.XRange.Setup(2*numxtics, 2*numxtics+5, width, leftm, false)

	// TODO(vodo) a) BinWidth might be input, alignment to tics should be nice, binCnt, ...
	if c.BinWidth == 0 {
		c.BinWidth = c.XRange.TicSetting.Delta
	}
	if c.BinWidth == 0 {
		c.BinWidth = 1
	}
	binCnt := int((c.XRange.Max-c.XRange.Min)/c.BinWidth + 0.5)
	c.FirstBin = c.XRange.Min + c.BinWidth/2
	binStart := c.XRange.Min // BUG: if min not on tic: ugly
	fmt.Printf("%d bins from %.2f width %.2f\n", binCnt, binStart, c.BinWidth)
	counts, max := c.binify(binStart, c.BinWidth, binCnt)

	// Fix lower end of y axis
	c.YRange.DataMax = float64(max)
	c.YRange.DataMin = 0
	c.YRange.MinMode.Fixed = true
	c.YRange.MinMode.Value = 0
	c.YRange.Setup(numytics, numytics+2, height, topm, true)

	g.Begin()

	if c.Title != "" {
		g.Title(c.Title)
	}

	g.XAxis(c.XRange, topm+height+fh, topm)
	g.YAxis(c.YRange, leftm-int(2*fw), leftm+width)

	xf := c.XRange.Data2Screen
	yf := c.YRange.Data2Screen

	numSets := len(c.Data)
	n := float64(numSets)
	gf, sf := c.widthFactor()

	ww := c.BinWidth * (1 - gf) // w'
	var w, s float64
	if !c.Stacked {
		w = ww / (n + (n-1)*sf)
		s = w * sf
	} else {
		w = ww
		s = -ww
	}

	fmt.Printf("gf=%.3f, sf=%.3f, bw=%.3f   ===>  ww=%.2f,   w=%.2f,  s=%.2f\n", gf, sf, c.BinWidth, ww, w, s)
	for d := numSets - 1; d >= 0; d-- {
		bars := make([]Barinfo, binCnt)
		ws := 0
		for b := 0; b < binCnt; b++ {
			xb := binStart + (float64(b)+0.5)*c.BinWidth
			x := xb - ww/2 + float64(d)*(s+w)
			xs := xf(x)
			xss := xf(x + w)
			ws = xss - xs
			bars[b].x, bars[b].w = xs, xss-xs
			off := 0
			if c.Stacked {
				for dd := d - 1; dd >= 0; dd-- {
					off += counts[dd][b]
				}
			}
			a, aa := yf(float64(off+counts[d][b])), yf(float64(off))
			bars[b].y, bars[b].h = a, abs(a-aa)
		}
		g.Bars(bars, c.Data[d].Style)
		if !c.Stacked && sf < 0 && gf != 0 && fh > 1 {
			// Whitelining
			lw := 1
			if ws > 25 {
				lw = 2
			}
			white := DataStyle{LineColor: "#ffffff", LineWidth: lw, LineStyle: SolidLine}
			for _, b := range bars {
				g.Line(b.x, b.y-1, b.x+b.w+1, b.y-1, white)
				g.Line(b.x+b.w+1, b.y-1, b.x+b.w+1, b.y+b.h, white)
			}
		}
	}

	if !c.Key.Hide {
		g.Key(layout.KeyX, layout.KeyY, c.Key)
	}
	g.End()
}
