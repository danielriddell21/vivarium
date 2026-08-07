//go:build ebiten

package gui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"

	"github.com/danielriddell21/crucible/geom"

	"github.com/danielriddell21/vivarium/internal/sim"
)

var face = text.NewGoXFace(basicfont.Face7x13)

func (g *Game) Draw(screen *ebiten.Image) {
	world := g.ensureWorldImg()
	world.Fill(nightTint(g.World.LightFactor()))

	for _, o := range g.World.Obstacles() {
		vector.FillCircle(world, float32(o.Pos.X), float32(o.Pos.Y), float32(o.Radius), colObstacle, true)
	}

	for _, f := range g.World.Foods {
		if !g.World.FoodRipe(f) {
			continue
		}
		clr := colFood
		if f.Type == 1 {
			clr = colFood2
		}
		vector.FillCircle(world, float32(f.Pos.X), float32(f.Pos.Y), 2, clr, false)
	}

	for _, a := range g.World.Agents {
		if !a.Alive {
			continue
		}
		drawAgent(world, a, a == g.Selected, g.agentBodyColor(a), !g.hideSignals)
	}

	// Blit the world through the camera, then draw overlays in screen space.
	screen.Fill(colBackground)
	op := &ebiten.DrawImageOptions{GeoM: g.cam.GeoM()}
	screen.DrawImage(world, op)

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

	if g.rec != nil && !g.rec.Done() {
		b := screen.Bounds()
		if g.pix == nil {
			g.pix = make([]byte, 4*b.Dx()*b.Dy())
		}
		screen.ReadPixels(g.pix)
		g.rec.Add(&image.RGBA{Pix: g.pix, Stride: 4 * b.Dx(), Rect: image.Rect(0, 0, b.Dx(), b.Dy())})
	}
}

func (g *Game) ensureWorldImg() *ebiten.Image {
	if g.worldImg == nil {
		g.worldImg = ebiten.NewImage(int(g.World.W), int(g.World.H))
	}
	return g.worldImg
}

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
		vector.FillCircle(dst, float32(a.Pos.X), float32(a.Pos.Y), float32(a.Traits.Size), body, true)
	}

	// Heading indicator.
	nose := a.Pos.Add(geom.FromAngle(a.Heading).Scale(a.Traits.Size + 3))
	vector.StrokeLine(dst, float32(a.Pos.X), float32(a.Pos.Y), float32(nose.X), float32(nose.Y), 1, body, true)
}

func drawTriangle(dst *ebiten.Image, pos geom.Vec2, heading, size float64, clr color.Color) {
	tip := pos.Add(geom.FromAngle(heading).Scale(size * 1.6))
	left := pos.Add(geom.FromAngle(heading + 2.4).Scale(size))
	right := pos.Add(geom.FromAngle(heading - 2.4).Scale(size))

	var p vector.Path
	p.MoveTo(float32(tip.X), float32(tip.Y))
	p.LineTo(float32(left.X), float32(left.Y))
	p.LineTo(float32(right.X), float32(right.Y))
	p.Close()

	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &p, nil, op)
}

func drawText(dst *ebiten.Image, s string, x, y float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

func drawPanel(dst *ebiten.Image, x, y, w, h float64) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), colPanel, false)
}

func radToDeg(r float64) float64 { return r * 180 / math.Pi }

func normDeg(d float64) float64 {
	d = math.Mod(d, 360)
	if d < 0 {
		d += 360
	}
	return d
}
