package gui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/danielriddell21/crucible/canvas"
	"github.com/danielriddell21/crucible/demo"
	"github.com/danielriddell21/crucible/record"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// recordSpeed advances the world this many ticks per captured frame, the same
// steady pace the windowed recorder uses.
const recordSpeed = 2

// Render records a run to o.Rec.Path without opening a window, drawing each
// frame with [DrawScene]. It needs no display, so documentation media builds
// anywhere.
//
// The file extension picks the format: .gif or .mp4.
func Render(o Config, w *sim.World) error {
	c := canvas.New(int(w.W), int(w.H))
	// The scene is broad flat colour over a dark background, so a palette built
	// from its own colours plus delta frames keeps the file small.
	rec := record.New(o.Rec,
		record.WithPalette(demo.Ramp(scenePalette(), sceneShades)),
		record.WithFrameDiff(),
	)
	st := SceneState{Speed: recordSpeed, Zoom: 1}

	clip := demo.Clip{
		Frames: o.Rec.Frames,
		Step: func(int) error {
			for range recordSpeed {
				w.Step()
			}
			return nil
		},
		Frame: func(int) image.Image {
			DrawScene(c, w, st)
			return record.FromRGBA(c.Pixels(), int(w.W), int(w.H))
		},
	}
	if _, err := clip.Record(rec); err != nil {
		return fmt.Errorf("capture run: %w", err)
	}
	if err := rec.Save(o.Rec.Path); err != nil {
		return fmt.Errorf("save recording: %w", err)
	}
	fmt.Printf("%s: %d frames\n", o.Rec.Path, rec.Len())
	return nil
}

// sceneShades is how many brightness levels each scene colour is banded into
// for a recording's palette.
const sceneShades = 20

// scenePalette is every colour the scene draws with, for GIF quantisation.
func scenePalette() []color.RGBA {
	return []color.RGBA{
		colBackground, colFood, colFood2, colHerbivore, colCarnivore,
		colSelected, colText, colObstacle,
		{R: 0xff, G: 0xcc, B: 0x55, A: 0xff}, // warm signal halo
		{R: 0x66, G: 0xcc, B: 0xff, A: 0xff}, // cool signal halo
	}
}
