package gui

import (
	"fmt"
	"path/filepath"

	"github.com/danielriddell21/crucible/canvas"
	"github.com/danielriddell21/crucible/record"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// Still names one documentation screenshot: which view is open, and how far in
// the camera is.
type Still struct {
	// Name is the file's base name, without an extension.
	Name string
	// View selects the panel to open: "", "inspector", "species", "lineage" or
	// "phylogeny". An empty view shows the world alone.
	View string
	// Zoom magnifies the world; zero means 1:1. CamX and CamY place the
	// frame's top-left corner in world coordinates.
	Zoom, CamX, CamY float64
}

// Shot renders one still from a world warmed up for the given number of ticks
// and writes it to dir as a PNG. The frame comes from the same [DrawScene] the
// window draws, so a screenshot in the docs is a screenshot of the real thing.
func Shot(o Config, s Still, warmup int, dir string) error {
	w, err := buildWorld(o)
	if err != nil {
		return err
	}
	if w == nil {
		return fmt.Errorf("still %s: nothing to render", s.Name)
	}
	for range warmup {
		w.Step()
	}

	c := canvas.New(int(w.W), int(w.H))
	DrawScene(c, w, s.state(w))

	path := filepath.Join(dir, s.Name+".png")
	if err := record.SavePNG(path, record.FromRGBA(c.Pixels(), int(w.W), int(w.H))); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	fmt.Println(path)
	return nil
}

// state opens the still's view and points its camera. The analysis, lineage
// and phylogeny views are computed on the spot, exactly as the window computes
// them when the player opens one.
func (s Still) state(w *sim.World) SceneState {
	st := SceneState{Speed: 1, Zoom: s.Zoom, CamX: s.CamX, CamY: s.CamY}
	switch s.View {
	case "inspector":
		st.Selected = pickAgent(w)
	case "species":
		st.Analysis = computeAnalysis(w, nil)
		st.BodyColor = clusterBodyColor(st.Analysis)
	case "lineage":
		st.Lineage = computeLineageView(w)
		st.BodyColor = lineageBodyColor(st.Lineage)
	case "phylogeny":
		st.Phylogeny = computePhylogeny(w)
	}
	return st
}

// pickAgent chooses the agent the inspector opens on: the oldest carnivore if
// there is one, since it has the fullest set of readings, and otherwise the
// oldest agent alive.
func pickAgent(w *sim.World) *sim.Agent {
	var best *sim.Agent
	for _, a := range w.Agents {
		if !a.Alive {
			continue
		}
		switch {
		case best == nil,
			a.Kind == sim.Carnivore && best.Kind != sim.Carnivore,
			a.Kind == best.Kind && a.Age > best.Age:
			best = a
		}
	}
	return best
}
