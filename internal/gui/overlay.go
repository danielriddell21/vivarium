//go:build ebiten

package gui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// drawHUD shows the run state, live counts, and control hints in the top-left.
func (g *Game) drawHUD(screen *ebiten.Image) {
	c := g.World.CountKinds()
	state := "RUNNING"
	if g.Paused {
		state = "PAUSED"
	}
	drawPanel(screen, 6, 6, 320, 120)
	drawText(screen, fmt.Sprintf("%s   speed x%d   tick %d", state, g.Speed, g.World.Tick), 12, 10, colText)
	drawText(screen, fmt.Sprintf("fps %.0f   tps %.0f   zoom x%.1f", ebiten.ActualFPS(), ebiten.ActualTPS(), g.cam.zoom), 12, 26, colText)
	drawText(screen, fmt.Sprintf("plants %d", c.Plants), 12, 42, colFood)
	drawText(screen, fmt.Sprintf("herbivores %d", c.Herbivores), 100, 42, colHerbivore)
	drawText(screen, fmt.Sprintf("carnivores %d", c.Carnivores), 12, 58, colCarnivore)
	drawText(screen, fmt.Sprintf("season x%.2f   light x%.2f", g.World.SeasonFactor(), g.World.LightFactor()), 12, 74, colText)
	drawText(screen, "space pause  +/- speed  click select  g/l/p views  h halos", 12, 90, colText)
	drawText(screen, "wheel zoom  arrows pan  0 reset  s save", 12, 106, colText)
	if g.saveMsgTTL > 0 {
		drawText(screen, g.saveMsg, 12, 128, colSelected)
	}
}

// graph geometry (bottom-left).
const (
	graphW      = 280.0
	graphH      = 90.0
	graphMargin = 8.0
)

// drawGraph plots plant/herbivore/carnivore counts over time as three polylines,
// each auto-scaled to the panel height by the running maximum.
func (g *Game) drawGraph(screen *ebiten.Image) {
	hist := g.World.History()
	gx := graphMargin
	gy := g.World.H - graphH - graphMargin
	drawPanel(screen, gx, gy, graphW, graphH)
	drawText(screen, "population over time", gx+6, gy+2, colText)

	if len(hist) < 2 {
		return
	}

	// Find the maximum across all series for a shared vertical scale.
	maxV := 1
	for _, c := range hist {
		maxV = maxInt(maxV, maxInt(c.Plants, maxInt(c.Herbivores, c.Carnivores)))
	}

	plotY := gy + 16
	plotH := graphH - 20
	line := func(get func(sim.Counts) int, clr color.Color) {
		n := len(hist)
		stepX := graphW / float64(n-1)
		var px, py float32
		for i, c := range hist {
			x := gx + float64(i)*stepX
			y := plotY + plotH*(1-float64(get(c))/float64(maxV))
			if i > 0 {
				vector.StrokeLine(screen, px, py, float32(x), float32(y), 1, clr, false)
			}
			px, py = float32(x), float32(y)
		}
	}
	line(func(c sim.Counts) int { return c.Plants }, colFood)
	line(func(c sim.Counts) int { return c.Herbivores }, colHerbivore)
	line(func(c sim.Counts) int { return c.Carnivores }, colCarnivore)

	drawText(screen, fmt.Sprintf("max %d", maxV), gx+graphW-60, gy+2, colText)
}

// drawInspector renders the selected agent's stats and live brain I/O on the right.
func (g *Game) drawInspector(screen *ebiten.Image) {
	a := g.Selected
	if a == nil {
		return
	}

	px := g.World.W - 230
	py := 6.0
	drawPanel(screen, px, py, 224, 396)

	header := colHerbivore
	if a.Kind == sim.Carnivore {
		header = colCarnivore
	}
	y := py + 4
	drawText(screen, fmt.Sprintf("#%d  %s", a.ID, a.Kind), px+8, y, header)
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
		drawText(screen, r, px+8, y, colText)
		y += 15
	}

	// Vision: directional bar strips (sector 0 = straight ahead). Guard against a
	// just-spawned agent that has not sensed yet.
	if len(a.LastInputs) >= 3*sim.VisionSectors {
		v := sim.VisionSectors
		y += 6
		drawText(screen, "vision: targets", px+8, y, colSelected)
		y += 15
		drawLevelBars(screen, a.LastInputs[0:v], px+8, y, 208, 16, colFood)
		y += 21
		drawText(screen, "vision: threats", px+8, y, colSelected)
		y += 15
		drawLevelBars(screen, a.LastInputs[v:2*v], px+8, y, 208, 16, colCarnivore)
		y += 21
		drawText(screen, "vision: voices", px+8, y, colSelected)
		y += 15
		drawBars(screen, a.LastInputs[2*v:3*v], px+8, y, 208, 16) // signed: signal in [-1,1]
		y += 21
	}

	drawText(screen, "brain outputs", px+8, y, colSelected)
	y += 15
	outLabels := []string{"turn", "speed", "eat", "signal"}
	y = drawVector(screen, a.LastOutputs, outLabels, px+8, y)

	y += 8
	drawText(screen, "memory (recurrent)", px+8, y, colSelected)
	y += 15
	drawBars(screen, a.LastMemory, px+8, y, 208, 18)
}

// drawLevelBars renders a row of upward bars for values in [0, 1] in clr, from a
// baseline at the bottom of the strip.
func drawLevelBars(screen *ebiten.Image, vals []float64, x, y, w, h float64, clr color.Color) {
	if len(vals) == 0 {
		return
	}
	base := y + h
	vector.StrokeLine(screen, float32(x), float32(base), float32(x+w), float32(base), 1, colText, false)
	slot := w / float64(len(vals))
	bw := slot * 0.7
	for i, v := range vals {
		if v > 1 {
			v = 1
		} else if v < 0 {
			v = 0
		}
		cx := x + float64(i)*slot + (slot-bw)/2
		bh := v * h
		vector.FillRect(screen, float32(cx), float32(base-bh), float32(bw), float32(bh), clr, false)
	}
}

// drawBars renders a row of signed bars for values in [-1, 1]: each bar grows up
// (green) for positive values and down (red) for negative, from a centre line.
func drawBars(screen *ebiten.Image, vals []float64, x, y, w, h float64) {
	if len(vals) == 0 {
		return
	}
	mid := y + h/2
	vector.StrokeLine(screen, float32(x), float32(mid), float32(x+w), float32(mid), 1, colText, false)
	slot := w / float64(len(vals))
	bw := slot * 0.7
	for i, v := range vals {
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		cx := x + float64(i)*slot + (slot-bw)/2
		bh := v * (h / 2)
		clr := colFood
		top := mid - bh
		if bh < 0 {
			clr = colCarnivore
			top = mid
			bh = -bh
		}
		vector.FillRect(screen, float32(cx), float32(top), float32(bw), float32(bh), clr, false)
	}
}

// drawVector prints a labelled list of values and returns the next y position.
func drawVector(screen *ebiten.Image, vals []float64, labels []string, x, y float64) float64 {
	for i, v := range vals {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}
		drawText(screen, fmt.Sprintf("%-8s % .2f", label, v), x, y, colText)
		y += 13
	}
	return y
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
