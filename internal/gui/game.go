//go:build ebiten

package gui

import (
	"cmp"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/danielriddell21/crucible/geom"

	"github.com/danielriddell21/vivarium/internal/sim"
)

const (
	maxSpeed = 32
	minSpeed = 1
)

type Game struct {
	World    *sim.World
	Paused   bool
	Speed    int
	Selected *sim.Agent

	analysisOn   bool
	analysis     *analysis
	framesToScan int

	lineageOn       bool
	lineageView     *lineageView
	framesToLineage int

	phyloOn       bool
	phylo         *phyloView
	framesToPhylo int

	hideSignals bool

	cam      camera
	worldImg *ebiten.Image

	SnapshotPath string
	saveMsg      string
	saveMsgTTL   int
}

func NewGame(w *sim.World) *Game {
	return &Game{World: w, Speed: 1, cam: newCamera()}
}

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
	g.handleSpeedKeys()
	g.handleViewKeys()
	g.handleCamera()
	g.refreshOverlays()
}

func (g *Game) handleSpeedKeys() {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.Paused = !g.Paused
	}
	// '+' / '=' speed up, '-' slow down. Both keypad and main row are accepted.
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		g.Speed = clamp(g.Speed*2, minSpeed, maxSpeed)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		g.Speed = clamp(g.Speed/2, minSpeed, maxSpeed)
	}
}

func (g *Game) handleViewKeys() {
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.analysisOn = !g.analysisOn
		g.framesToScan = 0 // recompute immediately on enable
		g.lineageOn, g.phyloOn = false, false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		g.lineageOn = !g.lineageOn
		g.framesToLineage = 0
		g.analysisOn, g.phyloOn = false, false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.phyloOn = !g.phyloOn
		g.framesToPhylo = 0
		g.analysisOn, g.lineageOn = false, false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.hideSignals = !g.hideSignals
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && g.SnapshotPath != "" {
		if err := sim.SaveSnapshot(g.SnapshotPath, g.World.Snapshot()); err != nil {
			g.saveMsg = "save failed: " + err.Error()
		} else {
			g.saveMsg = "saved population to " + g.SnapshotPath
		}
		g.saveMsgTTL = 180 // ~3s at 60fps
	}
	if g.saveMsgTTL > 0 {
		g.saveMsgTTL--
	}
}

func (g *Game) handleCamera() {
	mx, my := ebiten.CursorPosition()
	if _, wy := ebiten.Wheel(); wy != 0 {
		factor := 1.1
		if wy < 0 {
			factor = 1 / 1.1
		}
		g.cam.zoomAt(factor, float64(mx), float64(my), g.World.W, g.World.H)
	}
	const panStep = 12
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.cam.pan(-panStep, 0, g.World.W, g.World.H)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.cam.pan(panStep, 0, g.World.W, g.World.H)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.cam.pan(0, -panStep, g.World.W, g.World.H)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.cam.pan(0, panStep, g.World.W, g.World.H)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key0) {
		g.cam = newCamera()
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		wx, wy := g.cam.screenToWorld(float64(mx), float64(my))
		g.Selected = g.World.NearestAgent(geom.Vec2{X: wx, Y: wy})
	}
}

func (g *Game) refreshOverlays() {
	if g.analysisOn {
		if g.framesToScan <= 0 {
			g.analysis = computeAnalysis(g.World, g.analysis)
			g.framesToScan = analysisRefreshFrames
		}
		g.framesToScan--
	}
	if g.lineageOn {
		if g.framesToLineage <= 0 {
			g.lineageView = computeLineageView(g.World)
			g.framesToLineage = analysisRefreshFrames
		}
		g.framesToLineage--
	}
	if g.phyloOn {
		if g.framesToPhylo <= 0 {
			g.phylo = computePhylogeny(g.World)
			g.framesToPhylo = analysisRefreshFrames
		}
		g.framesToPhylo--
	}
}

func (g *Game) Layout(_, _ int) (int, int) {
	return int(g.World.W), int(g.World.H)
}

func clamp[T cmp.Ordered](v, lo, hi T) T {
	return max(lo, min(hi, v))
}
