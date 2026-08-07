//go:build ebiten

package gui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/danielriddell21/crucible/record"
	"github.com/danielriddell21/crucible/window"
)

func Available() bool { return true }

func Run(o Config) error {
	world, err := buildWorld(o)
	if err != nil || world == nil {
		return err
	}
	game := NewGame(world)
	game.SnapshotPath = o.SnapPath
	if o.Rec.Recording() {
		game.rec = record.New(o.Rec)
		game.recPath = o.Rec.Path
		game.Speed = 2 // a steady pace for a lively recording
	}

	window.Configure(window.Options{
		Title: "Vivarium — evolving ecosystem", Width: int(world.W), Height: int(world.H),
		MinWidth: int(world.W) / 2, MinHeight: int(world.H) / 2,
	})

	if err := ebiten.RunGame(game); err != nil {
		return fmt.Errorf("run game: %w", err)
	}
	return nil
}
