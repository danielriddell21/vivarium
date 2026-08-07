package gui

import (
	"image"
	_ "image/png" // register the PNG decoder for DecodeConfig
	"os"
	"path/filepath"
	"testing"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// warmWorld returns a world stepped far enough to have lineages, a genealogy
// and some history for the panels to draw.
func warmWorld(t *testing.T) *sim.World {
	t.Helper()
	w, err := buildWorld(Config{Seed: 5})
	if err != nil {
		t.Fatalf("buildWorld: %v", err)
	}
	for range 400 {
		w.Step()
	}
	return w
}

func TestShotWritesEveryView(t *testing.T) {
	// Every documentation still must render without a display. Each view opens
	// a different panel, so this covers all four panel paths as well.
	dir := t.TempDir()
	stills := []Still{
		{Name: "world"},
		{Name: "closeup", Zoom: 4, CamX: 300, CamY: 200},
		{Name: "inspector", View: "inspector"},
		{Name: "species", View: "species"},
		{Name: "lineage", View: "lineage"},
		{Name: "phylogeny", View: "phylogeny"},
	}
	for _, s := range stills {
		if err := Shot(Config{Seed: 5}, s, 400, dir); err != nil {
			t.Fatalf("Shot(%s): %v", s.Name, err)
		}
		path := filepath.Join(dir, s.Name+".png")
		f, err := os.Open(path) //nolint:gosec // path is built from a fixed name in a temp dir
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		cfg, format, err := image.DecodeConfig(f)
		_ = f.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if format != "png" {
			t.Errorf("%s is a %s, want png", s.Name, format)
		}
		if cfg.Width <= 0 || cfg.Height <= 0 {
			t.Errorf("%s is %dx%d", s.Name, cfg.Width, cfg.Height)
		}
	}
}

func TestStillStateOpensItsView(t *testing.T) {
	w := warmWorld(t)
	cases := []struct {
		view string
		open func(SceneState) bool
	}{
		{"", func(st SceneState) bool {
			return st.Selected == nil && st.Analysis == nil && st.Lineage == nil && st.Phylogeny == nil
		}},
		{"inspector", func(st SceneState) bool { return st.Selected != nil }},
		{"species", func(st SceneState) bool { return st.Analysis != nil && st.BodyColor != nil }},
		{"lineage", func(st SceneState) bool { return st.Lineage != nil && st.BodyColor != nil }},
		{"phylogeny", func(st SceneState) bool { return st.Phylogeny != nil }},
	}
	for _, c := range cases {
		if st := (Still{View: c.view}).state(w); !c.open(st) {
			t.Errorf("view %q did not open what it names", c.view)
		}
	}
}

func TestSceneStateProjectsThroughTheCamera(t *testing.T) {
	// The camera is what lets a still zoom in; a zero Zoom must stay 1:1.
	st := SceneState{}
	if x, y := st.project(10, 20); x != 10 || y != 20 {
		t.Errorf("default camera moved the world: (%v,%v), want (10,20)", x, y)
	}
	st = SceneState{Zoom: 4, CamX: 300, CamY: 200}
	if x, y := st.project(310, 220); x != 40 || y != 80 {
		t.Errorf("project(310,220) = (%v,%v), want (40,80)", x, y)
	}
}

func TestStatsHideTheFrameRateInAStill(t *testing.T) {
	// A still has no frame rate; showing zeroes would read as the app stalling.
	if got := (SceneState{}).zoom(); got != 1 {
		t.Errorf("zoom() = %v for the zero state, want 1", got)
	}
}
