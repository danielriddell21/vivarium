// Command demogen renders vivarium's documentation media headlessly: short,
// deterministic clips of the ecosystem running. It steps the simulation and
// composes frames on a software canvas — the same code the window draws with —
// so it needs no display.
//
// Regenerate every asset under docs/demos with:
//
//	just demos      // or: go run ./tools/demogen
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielriddell21/crucible/record"

	"github.com/danielriddell21/vivarium/internal/gui"
)

const outDir = "docs/demos"

// seed fixes the world the stills are taken from, so they stay reproducible.
const seed = 5

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "demogen:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	for _, c := range clips() {
		if err := c.record(); err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
	}
	return shots()
}

// shots is the documentation set of screenshots: the world on its own, a
// close-up of a few agents, and one per interactive view. Each is rendered
// from a warmed-up world so there is something evolved to look at.
func shots() error {
	const warmup = 1500
	for _, s := range []gui.Still{
		{Name: "world"},
		{Name: "closeup", Zoom: 4, CamX: 300, CamY: 200},
		{Name: "inspector", View: "inspector"},
		{Name: "species", View: "species"},
		{Name: "lineage", View: "lineage"},
		{Name: "phylogeny", View: "phylogeny"},
	} {
		if err := gui.Shot(gui.Config{Seed: seed}, s, warmup, outDir); err != nil {
			return fmt.Errorf("still %s: %w", s.Name, err)
		}
	}
	return nil
}

// clip is one recorded run: a seed, a length, and how the frames are encoded.
type clip struct {
	name string
	// ext is the output extension; an .mp4 records video instead of a GIF.
	ext    string
	seed   int64
	frames int
	// scale downsamples the recording; the world is large, so the clips halve
	// it to keep the files reasonable.
	scale int
}

// clips is the documentation set. The overview runs long enough for the
// population curves to move and for predation to show.
func clips() []clip {
	return []clip{
		{name: "overview", ext: ".gif", seed: seed, frames: 200, scale: 2},
	}
}

func (c clip) record() error {
	path := filepath.Join(outDir, c.name+c.ext)
	cfg := gui.Config{
		Seed: c.seed,
		Rec:  record.Options{Path: path, Frames: c.frames, FPS: 30, Scale: c.scale},
	}
	if err := gui.Render(cfg); err != nil {
		return fmt.Errorf("render: %w", err)
	}
	return nil
}
