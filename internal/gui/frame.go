package gui

import (
	"fmt"
	"image/color"

	"github.com/danielriddell21/crucible/canvas"
	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/keymap"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// textAscent lifts a top-left text origin onto the canvas face's baseline.
const textAscent = 11

// DrawScene composes a whole frame onto c: the world through the camera, the
// always-on stats panel and population graph, and whichever interactive views
// are open. Drawing through the software canvas means the same code paints the
// live window and every still and clip under docs/demos.
func DrawScene(c *canvas.Canvas, w *sim.World, st SceneState) {
	c.Fill(nightTint(w.LightFactor()))
	drawWorld(c, w, st)
	drawStats(c, w, st)
	drawPopulationGraph(c, w)
	if st.Selected != nil {
		drawInspector(c, st.Selected, w.W)
	}
	if st.Analysis != nil {
		drawAnalysisPanel(c, st.Analysis, w.W, w.H)
	}
	if st.Lineage != nil {
		drawLineagePanel(c, st.Lineage, w.LineageHistory(), w.W, w.H)
	}
	if st.Phylogeny != nil {
		drawPhylogenyPanel(c, st.Phylogeny, w.W, w.H)
	}
}

// SceneState is the view state the scene draws but the world does not hold:
// where the camera is, what the player has opened, and the readouts only the
// display knows.
type SceneState struct {
	Paused      bool
	Speed       int
	HideSignals bool
	FPS, TPS    float64

	// CamX, CamY is the world point at the frame's top-left, and Zoom the
	// magnification. A zero Zoom means 1.
	CamX, CamY, Zoom float64

	// Selected opens the inspector on one agent; the three view pointers open
	// their panels. All are nil when closed.
	Selected  *sim.Agent
	Analysis  *analysis
	Lineage   *lineageView
	Phylogeny *phyloView

	// BodyColor overrides an agent's body colour — the analysis and lineage
	// views recolour the world by cluster or founder. Nil leaves the default.
	BodyColor func(*sim.Agent) (color.RGBA, bool)
}

// zoom is the magnification to draw at, treating the zero value as 1:1.
func (st SceneState) zoom() float64 {
	if st.Zoom <= 0 {
		return 1
	}
	return st.Zoom
}

// project maps a world point onto the frame.
func (st SceneState) project(x, y float64) (float64, float64) {
	z := st.zoom()
	return (x - st.CamX) * z, (y - st.CamY) * z
}

func drawWorld(c *canvas.Canvas, w *sim.World, st SceneState) {
	z := st.zoom()
	for _, o := range w.Obstacles() {
		x, y := st.project(o.Pos.X, o.Pos.Y)
		c.Circle(x, y, o.Radius*z, colObstacle)
	}
	for _, f := range w.Foods {
		if !w.FoodRipe(f) {
			continue
		}
		col := colFood
		if f.Type == 1 {
			col = colFood2
		}
		x, y := st.project(f.Pos.X, f.Pos.Y)
		c.Circle(x, y, 2*z, col)
	}
	for _, a := range w.Agents {
		if a.Alive {
			drawAgentTo(c, a, st)
		}
	}
}

// drawAgentTo paints one agent: its signal halo, its selection ring, its body,
// and the nose line showing where it is headed.
func drawAgentTo(c *canvas.Canvas, a *sim.Agent, st SceneState) {
	body := colHerbivore
	if a.Kind == sim.Carnivore {
		body = colCarnivore
	}
	if st.BodyColor != nil {
		if col, ok := st.BodyColor(a); ok {
			body = col
		}
	}
	z := st.zoom()
	ax, ay := st.project(a.Pos.X, a.Pos.Y)
	size := a.Traits.Size * z

	// Communication halo: a faint ring whose opacity grows with how loudly the
	// agent is broadcasting, warm for a positive signal, cool for a negative one.
	if s := a.Signal; !st.HideSignals && (s > 0.15 || s < -0.15) {
		mag := s
		if mag < 0 {
			mag = -mag
		}
		alpha := uint8(60 + 160*mag)
		hue := color.RGBA{R: 0xff, G: 0xcc, B: 0x55, A: alpha} // warm = positive
		if s < 0 {
			hue = color.RGBA{R: 0x66, G: 0xcc, B: 0xff, A: alpha} // cool = negative
		}
		c.Ring(ax, ay, size+(4+3*mag)*z, 1, hue)
	}
	if a == st.Selected {
		c.Ring(ax, ay, size+3*z, 2, colSelected)
	}

	if a.Kind == sim.Carnivore {
		pt := func(offset, scale float64) [2]float64 {
			p := a.Pos.Add(geom.FromAngle(a.Heading + offset).Scale(a.Traits.Size * scale))
			x, y := st.project(p.X, p.Y)
			return [2]float64{x, y}
		}
		c.Polygon([][2]float64{pt(0, 1.6), pt(2.4, 1), pt(-2.4, 1)}, body)
	} else {
		c.Circle(ax, ay, size, body)
	}

	nose := a.Pos.Add(geom.FromAngle(a.Heading).Scale(a.Traits.Size + 3))
	nx, ny := st.project(nose.X, nose.Y)
	c.Line(ax, ay, nx, ny, 1, body)
}

// controls is vivarium's control scheme. crucible/keymap formats and wraps it
// to the stats panel, so it reads "key: action" like every other app's and
// stops running past the panel edge as it did when it was two fixed lines.
var controls = []keymap.Binding{
	{Key: "space", Action: "pause"},
	{Key: "+/-", Action: "speed"},
	{Key: "click", Action: "select"},
	{Key: "g/l/p", Action: "views"},
	{Key: "h", Action: "halos"},
	{Key: "wheel", Action: "zoom"},
	{Key: "arrows", Action: "pan"},
	{Key: "0", Action: "reset"},
	{Key: "s", Action: "save"},
}

// The stats panel's geometry. It sits at statsX,statsY and grows downward to
// fit however many rows the controls wrap onto.
const (
	statsX, statsY = 6.0, 6.0
	statsW         = 320.0
	statsInset     = 6.0  // text inset from the panel edge
	statsLineH     = 16   // one row of text
	statsHintsY    = 90.0 // where the control rows start
	statsFooter    = 4.0  // gap below the last row
)

func drawStats(c *canvas.Canvas, w *sim.World, st SceneState) {
	counts := w.CountKinds()
	state := "RUNNING"
	if st.Paused {
		state = "PAUSED"
	}
	rows := keymap.Rows(controls, statsW-2*statsInset, func(s string) int { return len(s) * canvas.GlyphWidth })
	panel(c, statsX, statsY, statsW, statsHintsY-statsY+float64(len(rows))*statsLineH+statsFooter)
	line := func(x, y int, s string, col color.RGBA) { c.Text(x, y+textAscent, s, col) }
	line(12, 10, fmt.Sprintf("%s   speed x%d   tick %d", state, st.Speed, w.Tick), colText)
	// A still has no frame rate to report, so it shows the zoom alone rather
	// than a pair of zeroes that would read as the app stalling.
	rate := fmt.Sprintf("zoom x%.1f", st.zoom())
	if st.FPS > 0 || st.TPS > 0 {
		rate = fmt.Sprintf("fps %.0f   tps %.0f   %s", st.FPS, st.TPS, rate)
	}
	line(12, 26, rate, colText)
	line(12, 42, fmt.Sprintf("plants %d", counts.Plants), colFood)
	line(100, 42, fmt.Sprintf("herbivores %d", counts.Herbivores), colHerbivore)
	line(12, 58, fmt.Sprintf("carnivores %d", counts.Carnivores), colCarnivore)
	line(12, 74, fmt.Sprintf("season x%.2f   light x%.2f", w.SeasonFactor(), w.LightFactor()), colText)
	for i, r := range rows {
		line(int(statsX+statsInset), int(statsHintsY)+i*statsLineH, r, colText)
	}
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
