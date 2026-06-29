//go:build ebiten

package gui

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// Available reports whether the Ebiten window is compiled in.
func Available() bool { return true }

// Run resolves the configuration and opens the Ebiten window.
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

	ebiten.SetWindowSize(int(cfg.Width), int(cfg.Height))
	ebiten.SetWindowTitle("Vivarium — evolving ecosystem")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		return fmt.Errorf("run game: %w", err)
	}
	return nil
}
