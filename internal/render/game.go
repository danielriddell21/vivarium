//go:build ebiten

package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/danielriddell21/vivarium/internal/geom"
	"github.com/danielriddell21/vivarium/internal/sim"
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

	// Analysis ("species") view: clusters the population in brain-genome space and
	// projects it to 2D. Recomputed on a throttle so it stays cheap.
	analysisOn   bool
	analysis     *analysis
	framesToScan int

	// Lineage view: a stacked chart of lineage abundance over time, with agents
	// coloured by lineage. Mutually exclusive with the species view.
	lineageOn       bool
	lineageView     *lineageView
	framesToLineage int

	// Phylogeny view: a coalescent genealogy tree of the living population.
	phyloOn       bool
	phylo         *phyloView
	framesToPhylo int

	// hideSignals suppresses the communication halos drawn around agents.
	hideSignals bool

	// cam pans/zooms the world view; worldImg is the offscreen buffer the world is
	// drawn into before being blitted through the camera transform.
	cam      camera
	worldImg *ebiten.Image

	// SnapshotPath is where the 's' key writes the population; saveMsg/saveMsgTTL
	// drive a brief on-screen confirmation.
	SnapshotPath string
	saveMsg      string
	saveMsgTTL   int
}

// NewGame returns a Game ready to be passed to ebiten.RunGame.
func NewGame(w *sim.World) *Game {
	return &Game{World: w, Speed: 1, cam: newCamera()}
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
	g.handleSpeedKeys()
	g.handleViewKeys()
	g.handleCamera()
	g.refreshOverlays()
}

// handleSpeedKeys toggles pause and adjusts simulation speed.
func (g *Game) handleSpeedKeys() {
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
}

// handleViewKeys toggles the analysis overlays and handles the save key.
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

// handleCamera applies wheel zoom, arrow-key panning, reset, and click-select.
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

// refreshOverlays recomputes whichever analysis overlay is active, on a throttle.
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
