//go:build ebiten

package gui

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/danielriddell21/crucible/record"
	"github.com/danielriddell21/crucible/window"

	"github.com/danielriddell21/vivarium/internal/sim"
)

func Available() bool { return true }

func Run(o Config) error {
	if o.PrintConfig {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(sim.DefaultConfig()); err != nil {
			return fmt.Errorf("print config: %w", err)
		}
		return nil
	}

	cfg := sim.DefaultConfig()
	// Precedence: built-in defaults < config file < explicitly-set CLI flags.
	if o.ConfigPath != "" {
		c, err := sim.LoadConfig(o.ConfigPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		cfg = c
	}
	if o.Changed("width") {
		cfg.Width = o.Width
	}
	if o.Changed("height") {
		cfg.Height = o.Height
	}
	if o.Changed("plants") {
		cfg.Plants = o.Plants
	}
	if o.Changed("herbivores") {
		cfg.Herbivores = o.Herbivores
	}
	if o.Changed("carnivores") {
		cfg.Carnivores = o.Carnivores
	}
	if o.Changed("rescue") {
		cfg.Rescue = o.Rescue
	}
	if cfg.TargetPlants < cfg.Plants {
		cfg.TargetPlants = cfg.Plants
	}

	rng := rand.New(rand.NewSource(o.Seed))
	var world *sim.World
	if o.LoadPath != "" {
		snap, err := sim.LoadSnapshotFile(o.LoadPath)
		if err != nil {
			return fmt.Errorf("load snapshot: %w", err)
		}
		world = sim.NewWorldFromSnapshot(rng, snap)
		cfg.Width, cfg.Height = world.W, world.H
	} else {
		world = sim.NewWorld(rng, cfg)
	}
	game := NewGame(world)
	game.SnapshotPath = o.SnapPath
	if o.Rec.Recording() {
		game.rec = record.New(o.Rec)
		game.recPath = o.Rec.Path
		game.Speed = 2 // a steady pace for a lively recording
	}

	window.Configure(window.Options{
		Title: "Vivarium — evolving ecosystem", Width: int(cfg.Width), Height: int(cfg.Height),
		MinWidth: int(cfg.Width) / 2, MinHeight: int(cfg.Height) / 2,
	})

	if err := ebiten.RunGame(game); err != nil {
		return fmt.Errorf("run game: %w", err)
	}
	return nil
}
