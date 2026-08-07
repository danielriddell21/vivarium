package gui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/danielriddell21/crucible/canvas"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// The interactive views — the inspector and the analysis, lineage and
// phylogeny panels — draw here rather than against the display, so the window
// and tools/demogen paint them with the same code. The stills under docs/demos
// are exactly these panels.

// drawInspector shows everything known about one agent: its traits, what its
// senses currently report, and what its brain did with that.
func drawInspector(c *canvas.Canvas, a *sim.Agent, worldW float64) {
	px := worldW - 230
	py := 6.0
	panel(c, px, py, 224, 396)

	header := colHerbivore
	if a.Kind == sim.Carnivore {
		header = colCarnivore
	}
	text := func(s string, x, y float64, col color.RGBA) { c.Text(int(x), int(y)+textAscent, s, col) }

	y := py + 4
	text(fmt.Sprintf("#%d  %s", a.ID, a.Kind), px+8, y, header)
	y += 18
	rows := []string{
		fmt.Sprintf("energy %.1f", a.Energy),
		fmt.Sprintf("age %d   gen %d", a.Age, a.Generation),
		fmt.Sprintf("heading %.0f deg", normDeg(radToDeg(a.Heading))),
		fmt.Sprintf("size %.1f  speed %.2f  diet %.2f", a.Traits.Size, a.Traits.MaxSpeed, a.Traits.Diet),
		fmt.Sprintf("sense %.0f  plast %.3f", a.Traits.SenseRadius, a.Traits.Plasticity),
		fmt.Sprintf("curio %.2f  surprise %.2f", a.Traits.Curiosity, a.LastSurprise),
		fmt.Sprintf("learned %.3f  reward %+.2f", a.Brain.LearnedDrift(), a.LastReward),
	}
	for _, r := range rows {
		text(r, px+8, y, colText)
		y += 15
	}

	// Vision: directional bar strips (sector 0 = straight ahead). Guard against a
	// just-spawned agent that has not sensed yet.
	if len(a.LastInputs) >= 3*sim.VisionSectors {
		v := sim.VisionSectors
		y += 6
		text("vision: targets", px+8, y, colSelected)
		y += 15
		drawLevelBars(c, a.LastInputs[0:v], px+8, y, 208, 16, colFood)
		y += 21
		text("vision: threats", px+8, y, colSelected)
		y += 15
		drawLevelBars(c, a.LastInputs[v:2*v], px+8, y, 208, 16, colCarnivore)
		y += 21
		text("vision: voices", px+8, y, colSelected)
		y += 15
		drawBars(c, a.LastInputs[2*v:3*v], px+8, y, 208, 16) // signed: signal in [-1,1]
		y += 21
	}

	text("brain outputs", px+8, y, colSelected)
	y += 15
	for i, v := range a.LastOutputs {
		label := ""
		if outLabels := [...]string{"turn", "speed", "eat", "signal"}; i < len(outLabels) {
			label = outLabels[i]
		}
		text(fmt.Sprintf("%-8s % .2f", label, v), px+8, y, colText)
		y += 13
	}

	y += 8
	text("memory (recurrent)", px+8, y, colSelected)
	y += 15
	drawBars(c, a.LastMemory, px+8, y, 208, 18)
}

// drawLevelBars plots values in [0,1] as bars rising from a baseline.
func drawLevelBars(c *canvas.Canvas, vals []float64, x, y, w, h float64, col color.RGBA) {
	if len(vals) == 0 {
		return
	}
	base := y + h
	c.Line(x, base, x+w, base, 1, colText)
	slot := w / float64(len(vals))
	bw := slot * 0.7
	for i, v := range vals {
		v = clamp01(v)
		cx := x + float64(i)*slot + (slot-bw)/2
		bh := v * h
		c.Rect(int(cx), int(base-bh), int(bw), int(bh), col)
	}
}

// drawBars plots signed values in [-1,1] above and below a midline, warm for
// positive and cool for negative.
func drawBars(c *canvas.Canvas, vals []float64, x, y, w, h float64) {
	if len(vals) == 0 {
		return
	}
	mid := y + h/2
	c.Line(x, mid, x+w, mid, 1, colText)
	slot := w / float64(len(vals))
	bw := slot * 0.7
	for i, v := range vals {
		v = clampSigned(v)
		cx := x + float64(i)*slot + (slot-bw)/2
		bh := v * (h / 2)
		col := colFood
		top := mid - bh
		if bh < 0 {
			col = colCarnivore
			top = mid
			bh = -bh
		}
		c.Rect(int(cx), int(top), int(bw), int(bh), col)
	}
}

// drawAnalysisPanel shows the population clustered by genome, with a PCA
// scatter of the same sample beside the legend.
func drawAnalysisPanel(c *canvas.Canvas, a *analysis, worldW, worldH float64) {
	const pw, ph = 232.0, 196.0
	px, py := worldW-pw-6, worldH-ph-6
	panel(c, px, py, pw, ph)
	text := func(s string, x, y float64, col color.RGBA) { c.Text(int(x), int(y)+textAscent, s, col) }
	text("species (genome k-means + PCA)", px+6, py+4, colText)

	if a == nil || a.total < 2 {
		text("population too small", px+6, py+22, colText)
		return
	}
	text(fmt.Sprintf("K=%d   n=%d/%d", a.k, a.sampled, a.total), px+6, py+20, colText)

	// Legend: a coloured swatch and size per cluster.
	ly := py + 36
	for cl := range a.k {
		c.Rect(int(px+8), int(ly), 8, 8, clusterColor(cl))
		text(fmt.Sprintf("sp%d  %d", cl, a.sizes[cl]), px+22, ly-3, colText)
		ly += 14
	}

	// PCA scatter to the right of the legend.
	plotX, plotY := px+110, py+34
	plotW, plotH := pw-116, ph-42
	strokeRect(c, plotX, plotY, plotW, plotH, colPanel)
	if !a.pcaOK {
		return
	}
	spanX := nonZeroSpan(a.maxX - a.minX)
	spanY := nonZeroSpan(a.maxY - a.minY)
	for agent, xy := range a.coords {
		fx := plotX + (xy[0]-a.minX)/spanX*plotW
		fy := plotY + plotH - (xy[1]-a.minY)/spanY*plotH
		c.Circle(fx, fy, 1.5, clusterColor(a.clusterOf[agent]))
	}
}

// drawLineagePanel stacks each lineage's share of the population over time, so
// a founder line taking over shows as a widening band.
func drawLineagePanel(c *canvas.Canvas, lv *lineageView, hist []map[int]int, worldW, worldH float64) {
	const pw, ph = 232.0, 196.0
	px, py := worldW-pw-6, worldH-ph-6
	panel(c, px, py, pw, ph)
	text := func(s string, x, y float64, col color.RGBA) { c.Text(int(x), int(y)+textAscent, s, col) }
	text("lineages over time", px+6, py+4, colText)

	if lv == nil || len(hist) < 2 {
		text("gathering history...", px+6, py+22, colText)
		return
	}

	plotX, plotY := px+6, py+34
	plotW, plotH := pw-12, ph-58
	n := len(hist)
	for col := range int(plotW) {
		idx := col * (n - 1) / (int(plotW) - 1)
		s := hist[idx]
		x := plotX + float64(col)
		bottom := plotY + plotH

		var accH float64
		total := 0
		for _, cnt := range s {
			total += cnt
		}
		sumTop := 0
		for _, id := range lv.topIDs {
			cnt := s[id]
			sumTop += cnt
			segH := float64(cnt) / float64(lv.maxTotal) * plotH
			if segH > 0 {
				c.Rect(int(x), int(bottom-accH-segH), 1, int(segH), lineagePalette[lv.colorOf[id]])
				accH += segH
			}
		}
		if otherH := float64(total-sumTop) / float64(lv.maxTotal) * plotH; otherH > 0 {
			c.Rect(int(x), int(bottom-accH-otherH), 1, int(otherH), colOther)
		}
	}

	// Caption: how many lineages are alive now vs how many get their own band.
	live := 0
	for _, cnt := range hist[n-1] {
		if cnt > 0 {
			live++
		}
	}
	text(fmt.Sprintf("%d live lineages; top %d shown, rest grey", live, len(lv.topIDs)), px+6, py+ph-16, colText)
}

// drawPhylogenyPanel draws the coalescent tree of the living population, older
// generations to the left.
func drawPhylogenyPanel(c *canvas.Canvas, pv *phyloView, worldW, worldH float64) {
	px, py := worldW*0.10, 64.0
	pw, ph := worldW*0.80, worldH-130

	panel(c, px, py, pw, ph)
	text := func(s string, x, y float64, col color.RGBA) { c.Text(int(x), int(y)+textAscent, s, col) }
	text("phylogeny - coalescent tree of living population (older left, now right)", px+8, py+4, colText)

	if pv == nil || pv.leaves < 2 {
		text("building genealogy...", px+8, py+22, colText)
		return
	}

	plotX, plotY := px+12, py+26
	plotW, plotH := pw-24, ph-40
	for _, e := range pv.edges {
		p := pv.pos[e[0]]
		ch := pv.pos[e[1]]
		x1, x2 := plotX+p.x*plotW, plotX+ch.x*plotW
		y1, y2 := plotY+p.y*plotH, plotY+ch.y*plotH
		col := lineagePalette[ch.lineage%len(lineagePalette)]
		// Rectangular elbow: vertical at the parent's time, then horizontal to child.
		c.Line(x1, y1, x1, y2, 1, col)
		c.Line(x1, y2, x2, y2, 1, col)
	}
	// Mark the living leaves.
	for _, id := range pv.leafIDs {
		p := pv.pos[id]
		c.Circle(plotX+p.x*plotW, plotY+p.y*plotH, 2, lineagePalette[p.lineage%len(lineagePalette)])
	}

	text(fmt.Sprintf("%d leaves shown of %d living; %d nodes", pv.leaves, pv.living, len(pv.pos)), px+8, py+ph-16, colText)
}

// strokeRect outlines a rectangle a pixel wide.
func strokeRect(c *canvas.Canvas, x, y, w, h float64, col color.RGBA) {
	c.Line(x, y, x+w, y, 1, col)
	c.Line(x+w, y, x+w, y+h, 1, col)
	c.Line(x+w, y+h, x, y+h, 1, col)
	c.Line(x, y+h, x, y, 1, col)
}

func clamp01(v float64) float64     { return math.Max(0, math.Min(1, v)) }
func clampSigned(v float64) float64 { return math.Max(-1, math.Min(1, v)) }

// nonZeroSpan keeps a degenerate axis from dividing by zero.
func nonZeroSpan(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}

func radToDeg(r float64) float64 { return r * 180 / math.Pi }

func normDeg(d float64) float64 {
	d = math.Mod(d, 360)
	if d < 0 {
		d += 360
	}
	return d
}
