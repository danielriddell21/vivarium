package render

import (
	"image/color"
	"math"

	"github.com/danielriddell21/vivarium/internal/geom"
	"github.com/danielriddell21/vivarium/internal/sim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// Palette for the world and overlays.
var (
	colBackground = color.RGBA{0x12, 0x16, 0x14, 0xff}
	colFood       = color.RGBA{0x3c, 0xb0, 0x43, 0xff}
	colHerbivore  = color.RGBA{0x4f, 0x9d, 0xff, 0xff}
	colCarnivore  = color.RGBA{0xe0, 0x4f, 0x4f, 0xff}
	colSelected   = color.RGBA{0xff, 0xe0, 0x4f, 0xff}
	colText       = color.RGBA{0xe6, 0xe6, 0xe6, 0xff}
	colPanel      = color.RGBA{0x00, 0x00, 0x00, 0xc0}
)

// face is a shared bitmap font for all overlay text (no external font files).
var face = text.NewGoXFace(basicfont.Face7x13)

// whiteImage is a 3x3 white source used to fill arbitrary vector paths.
var whiteImage = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img
}()

// Draw renders one frame: world first, then overlays.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colBackground)

	for _, f := range g.World.Foods {
		if !g.World.FoodRipe(f) {
			continue
		}
		vector.DrawFilledCircle(screen, float32(f.Pos.X), float32(f.Pos.Y), 2, colFood, false)
	}

	for _, a := range g.World.Agents {
		if !a.Alive {
			continue
		}
		drawAgent(screen, a, a == g.Selected, g.agentBodyColor(a), !g.hideSignals)
	}

	g.drawGraph(screen)
	g.drawHUD(screen)
	g.drawInspector(screen)
	if g.analysisOn {
		g.drawAnalysisPanel(screen)
	}
	if g.lineageOn {
		g.drawLineagePanel(screen)
	}
	if g.phyloOn {
		g.drawPhylogenyPanel(screen)
	}
}

// drawAgent renders one agent. override, when non-nil, replaces the default
// kind-based body colour (used by the species/analysis view). showSignal toggles
// the communication halo.
func drawAgent(dst *ebiten.Image, a *sim.Agent, selected bool, override color.Color, showSignal bool) {
	var body color.Color = colHerbivore
	if a.Kind == sim.Carnivore {
		body = colCarnivore
	}
	if override != nil {
		body = override
	}
	// Communication halo: a faint ring whose opacity grows with how loudly the
	// agent is broadcasting, warm for a positive signal, cool for a negative one.
	if s := a.Signal; showSignal && (s > 0.15 || s < -0.15) {
		mag := s
		if mag < 0 {
			mag = -mag
		}
		alpha := uint8(60 + 160*mag)
		hue := color.RGBA{0xff, 0xcc, 0x55, alpha} // warm = positive
		if s < 0 {
			hue = color.RGBA{0x66, 0xcc, 0xff, alpha} // cool = negative
		}
		vector.StrokeCircle(dst, float32(a.Pos.X), float32(a.Pos.Y), float32(a.Traits.Size+4+3*mag), 1, hue, true)
	}
	if selected {
		// Highlight ring behind the body.
		vector.StrokeCircle(dst, float32(a.Pos.X), float32(a.Pos.Y), float32(a.Traits.Size+3), 2, colSelected, true)
	}

	switch a.Kind {
	case sim.Carnivore:
		drawTriangle(dst, a.Pos, a.Heading, a.Traits.Size, body)
	default:
		vector.DrawFilledCircle(dst, float32(a.Pos.X), float32(a.Pos.Y), float32(a.Traits.Size), body, true)
	}

	// Heading indicator.
	nose := a.Pos.Add(geom.FromAngle(a.Heading).Scale(a.Traits.Size + 3))
	vector.StrokeLine(dst, float32(a.Pos.X), float32(a.Pos.Y), float32(nose.X), float32(nose.Y), 1, body, true)
}

// drawTriangle fills an isoceles triangle centred at pos, pointing along heading.
func drawTriangle(dst *ebiten.Image, pos geom.Vec2, heading, size float64, clr color.Color) {
	tip := pos.Add(geom.FromAngle(heading).Scale(size * 1.6))
	left := pos.Add(geom.FromAngle(heading + 2.4).Scale(size))
	right := pos.Add(geom.FromAngle(heading - 2.4).Scale(size))

	var p vector.Path
	p.MoveTo(float32(tip.X), float32(tip.Y))
	p.LineTo(float32(left.X), float32(left.Y))
	p.LineTo(float32(right.X), float32(right.Y))
	p.Close()

	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	r, gg, b, alpha := clr.RGBA()
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(gg) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(alpha) / 0xffff
	}
	dst.DrawTriangles(vs, is, whiteImage, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

// drawText renders s at (x, y) measured from the top-left.
func drawText(dst *ebiten.Image, s string, x, y float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

// drawPanel draws a translucent rounded-ish background box for overlay text.
func drawPanel(dst *ebiten.Image, x, y, w, h float64) {
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), colPanel, false)
}

func radToDeg(r float64) float64 { return r * 180 / math.Pi }

// normDeg normalises an angle in degrees to the range [0, 360).
func normDeg(d float64) float64 {
	d = math.Mod(d, 360)
	if d < 0 {
		d += 360
	}
	return d
}
