//go:build ebiten

package gui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/danielriddell21/crucible/canvas"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// Draw composes the whole frame on the software canvas and blits it. Nothing
// is drawn against the display itself, so the window and the documentation
// media in tools/demogen are the same pixels.
func (g *Game) Draw(screen *ebiten.Image) {
	c := g.frame()
	DrawScene(c, g.World, g.sceneState())
	screen.WritePixels(c.Pixels())
}

// frame returns the canvas the frame is composed on, sized to the world.
func (g *Game) frame() *canvas.Canvas {
	w, h := int(g.World.W), int(g.World.H)
	if g.canvas == nil {
		g.canvas = canvas.New(w, h)
	}
	return g.canvas
}

// sceneState gathers everything the scene draws that the world does not hold:
// the camera, the open views, and the display's own readouts.
func (g *Game) sceneState() SceneState {
	st := SceneState{
		Paused:      g.Paused,
		Speed:       g.Speed,
		HideSignals: g.hideSignals,
		FPS:         ebiten.ActualFPS(),
		TPS:         ebiten.ActualTPS(),
		CamX:        g.cam.X,
		CamY:        g.cam.Y,
		Zoom:        g.cam.Zoom,
		Selected:    g.Selected,
		BodyColor:   g.agentBodyColor,
	}
	if g.analysisOn {
		st.Analysis = g.analysis
	}
	if g.lineageOn {
		st.Lineage = g.lineageView
	}
	if g.phyloOn {
		st.Phylogeny = g.phylo
	}
	return st
}

// agentBodyColor recolours the world by whichever view is open: by genome
// cluster for the analysis view, by founder for the lineage view.
func (g *Game) agentBodyColor(a *sim.Agent) (color.RGBA, bool) {
	if g.lineageOn {
		if f := lineageBodyColor(g.lineageView); f != nil {
			return f(a)
		}
	}
	if g.analysisOn {
		if f := clusterBodyColor(g.analysis); f != nil {
			return f(a)
		}
	}
	return color.RGBA{}, false
}
