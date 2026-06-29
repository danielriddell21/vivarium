//go:build ebiten

package cli

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/danielriddell21/vivarium/internal/render"
	"github.com/danielriddell21/vivarium/internal/sim"
)

// runGame resolves the configuration and opens the Ebiten window.
func runGame(o guiOpts) error {
	if o.printConfig {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(sim.DefaultConfig()); err != nil {
			return fmt.Errorf("print config: %w", err)
		}
		return nil
	}

	cfg := sim.DefaultConfig()
	// Precedence: built-in defaults < config file < explicitly-set CLI flags.
	if o.configPath != "" {
		c, err := sim.LoadConfig(o.configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		cfg = c
	}
	if o.changed("width") {
		cfg.Width = o.width
	}
	if o.changed("height") {
		cfg.Height = o.height
	}
	if o.changed("plants") {
		cfg.Plants = o.plants
	}
	if o.changed("herbivores") {
		cfg.Herbivores = o.herbivores
	}
	if o.changed("carnivores") {
		cfg.Carnivores = o.carnivores
	}
	if o.changed("rescue") {
		cfg.Rescue = o.rescue
	}
	if cfg.TargetPlants < cfg.Plants {
		cfg.TargetPlants = cfg.Plants
	}

	rng := rand.New(rand.NewSource(o.seed))
	var world *sim.World
	if o.loadPath != "" {
		snap, err := sim.LoadSnapshotFile(o.loadPath)
		if err != nil {
			return fmt.Errorf("load snapshot: %w", err)
		}
		world = sim.NewWorldFromSnapshot(rng, snap)
		cfg.Width, cfg.Height = world.W, world.H
	} else {
		world = sim.NewWorld(rng, cfg)
	}
	game := render.NewGame(world)
	game.SnapshotPath = o.snapPath

	ebiten.SetWindowSize(int(cfg.Width), int(cfg.Height))
	ebiten.SetWindowTitle("Vivarium — evolving ecosystem")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		return fmt.Errorf("run game: %w", err)
	}
	return nil
}
