// Package render wires the simulation to Ebiten: it draws the world, overlays
// (population graph, HUD, inspector), and handles keyboard/mouse input. It is the
// only package besides cmd that depends on Ebiten, keeping the simulation core
// free of any graphics concerns.
package render

import (
	"github.com/danielriddell21/vivarium/internal/geom"
	"github.com/danielriddell21/vivarium/internal/sim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	maxSpeed = 32 // maximum simulation steps per displayed frame
	minSpeed = 1
)

// Game adapts a sim.World to the ebiten.Game interface.
type Game struct {
	World    *sim.World
	Paused   bool
	Speed    int        // simulation steps per frame when running
	Selected *sim.Agent // agent shown in the inspector, or nil
}

// NewGame returns a Game ready to be passed to ebiten.RunGame.
func NewGame(w *sim.World) *Game {
	return &Game{World: w, Speed: 1}
}

// Update handles input and advances the simulation.
func (g *Game) Update() error {
	g.handleInput()
	if !g.Paused {
		for i := 0; i < g.Speed; i++ {
			g.World.Step()
		}
	}
	// Drop the inspector selection if its agent has died.
	if g.Selected != nil && !g.Selected.Alive {
		g.Selected = nil
	}
	return nil
}

func (g *Game) handleInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.Paused = !g.Paused
	}
	// '+' / '=' speed up, '-' slow down. Both keypad and main row are accepted.
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		g.Speed = clampInt(g.Speed*2, minSpeed, maxSpeed)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		g.Speed = clampInt(g.Speed/2, minSpeed, maxSpeed)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		g.Selected = g.World.NearestAgent(geom.Vec2{X: float64(mx), Y: float64(my)})
	}
}

// Layout fixes the logical screen to the world size; Ebiten scales to the window.
func (g *Game) Layout(_, _ int) (int, int) {
	return int(g.World.W), int(g.World.H)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
