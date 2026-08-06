package gui

import (
	"fmt"
	"image/color"

	"github.com/danielriddell21/crucible/canvas"
	"github.com/danielriddell21/crucible/geom"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// textAscent lifts a top-left text origin onto the canvas face's baseline.
const textAscent = 11

// DrawScene composes the world and the always-on overlays — the stats panel
// and the population graph — onto c. Drawing through the software canvas means
// the same code paints a live window and a headless recording.
//
// The inspector and the analysis, lineage and phylogeny panels stay with the
// window: they are interactive views, opened by hand, and never appear in a
// recording.
func DrawScene(c *canvas.Canvas, w *sim.World, st SceneState) {
	c.Fill(nightTint(w.LightFactor()))
	drawWorld(c, w, st)
	drawStats(c, w, st)
	drawPopulationGraph(c, w)
}

// SceneState is the view state the scene draws but the world does not hold.
type SceneState struct {
	Paused      bool
	Speed       int
	HideSignals bool
	FPS, TPS    float64
	Zoom        float64
}

func drawWorld(c *canvas.Canvas, w *sim.World, st SceneState) {
	for _, o := range w.Obstacles() {
		c.Circle(o.Pos.X, o.Pos.Y, o.Radius, colObstacle)
	}
	for _, f := range w.Foods {
		if !w.FoodRipe(f) {
			continue
		}
		col := colFood
		if f.Type == 1 {
			col = colFood2
		}
		c.Circle(f.Pos.X, f.Pos.Y, 2, col)
	}
	for _, a := range w.Agents {
		if a.Alive {
			drawAgentTo(c, a, !st.HideSignals)
		}
	}
}

// drawAgentTo paints one agent: its signal halo, its body, and the nose line
// showing where it is headed.
func drawAgentTo(c *canvas.Canvas, a *sim.Agent, showSignal bool) {
	body := colHerbivore
	if a.Kind == sim.Carnivore {
		body = colCarnivore
	}
	// Communication halo: a faint ring whose opacity grows with how loudly the
	// agent is broadcasting, warm for a positive signal, cool for a negative one.
	if s := a.Signal; showSignal && (s > 0.15 || s < -0.15) {
		mag := s
		if mag < 0 {
			mag = -mag
		}
		alpha := uint8(60 + 160*mag)
		hue := color.RGBA{R: 0xff, G: 0xcc, B: 0x55, A: alpha} // warm = positive
		if s < 0 {
			hue = color.RGBA{R: 0x66, G: 0xcc, B: 0xff, A: alpha} // cool = negative
		}
		c.Ring(a.Pos.X, a.Pos.Y, a.Traits.Size+4+3*mag, 1, hue)
	}

	if a.Kind == sim.Carnivore {
		tip := a.Pos.Add(geom.FromAngle(a.Heading).Scale(a.Traits.Size * 1.6))
		left := a.Pos.Add(geom.FromAngle(a.Heading + 2.4).Scale(a.Traits.Size))
		right := a.Pos.Add(geom.FromAngle(a.Heading - 2.4).Scale(a.Traits.Size))
		c.Polygon([][2]float64{{tip.X, tip.Y}, {left.X, left.Y}, {right.X, right.Y}}, body)
	} else {
		c.Circle(a.Pos.X, a.Pos.Y, a.Traits.Size, body)
	}

	nose := a.Pos.Add(geom.FromAngle(a.Heading).Scale(a.Traits.Size + 3))
	c.Line(a.Pos.X, a.Pos.Y, nose.X, nose.Y, 1, body)
}

func drawStats(c *canvas.Canvas, w *sim.World, st SceneState) {
	counts := w.CountKinds()
	state := "RUNNING"
	if st.Paused {
		state = "PAUSED"
	}
	panel(c, 6, 6, 320, 120)
	line := func(x, y int, s string, col color.RGBA) { c.Text(x, y+textAscent, s, col) }
	line(12, 10, fmt.Sprintf("%s   speed x%d   tick %d", state, st.Speed, w.Tick), colText)
	line(12, 26, fmt.Sprintf("fps %.0f   tps %.0f   zoom x%.1f", st.FPS, st.TPS, st.Zoom), colText)
	line(12, 42, fmt.Sprintf("plants %d", counts.Plants), colFood)
	line(100, 42, fmt.Sprintf("herbivores %d", counts.Herbivores), colHerbivore)
	line(12, 58, fmt.Sprintf("carnivores %d", counts.Carnivores), colCarnivore)
	line(12, 74, fmt.Sprintf("season x%.2f   light x%.2f", w.SeasonFactor(), w.LightFactor()), colText)
	line(12, 90, "space pause  +/- speed  click select  g/l/p views  h halos", colText)
	line(12, 106, "wheel zoom  arrows pan  0 reset  s save", colText)
}

func drawPopulationGraph(c *canvas.Canvas, w *sim.World) {
	hist := w.History()
	gx, gy := graphMargin, w.H-graphH-graphMargin
	panel(c, gx, gy, graphW, graphH)
	c.Text(int(gx)+6, int(gy)+2+textAscent, "population over time", colText)
	if len(hist) < 2 {
		return
	}

	maxV := 1
	for _, h := range hist {
		maxV = maxInt(maxV, maxInt(h.Plants, maxInt(h.Herbivores, h.Carnivores)))
	}
	plotY, plotH := gy+16, graphH-20
	series := func(get func(sim.Counts) int, col color.RGBA) {
		stepX := graphW / float64(len(hist)-1)
		var px, py float64
		for i, h := range hist {
			x := gx + float64(i)*stepX
			y := plotY + plotH*(1-float64(get(h))/float64(maxV))
			if i > 0 {
				c.Line(px, py, x, y, 1, col)
			}
			px, py = x, y
		}
	}
	series(func(h sim.Counts) int { return h.Plants }, colFood)
	series(func(h sim.Counts) int { return h.Herbivores }, colHerbivore)
	series(func(h sim.Counts) int { return h.Carnivores }, colCarnivore)
	c.Text(int(gx+graphW)-60, int(gy)+2+textAscent, fmt.Sprintf("max %d", maxV), colText)
}

// panel lays a translucent backing behind an overlay so its text stays legible
// over the world.
func panel(c *canvas.Canvas, x, y, w, h float64) {
	c.Blend(int(x), int(y), int(w), int(h), colPanel)
}
