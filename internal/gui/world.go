package gui

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// buildWorld resolves the configuration — defaults, then a config file, then
// any explicitly-set flags — and returns the world to run. A printed config
// returns a nil world and no error, meaning there is nothing to run.
func buildWorld(o Config) (*sim.World, error) {
	// Changed reports which flags the user set explicitly. A programmatic
	// caller — the demo generator — sets none, so treat a missing hook as
	// "nothing overridden" rather than dereferencing it.
	changed := o.Changed
	if changed == nil {
		changed = func(string) bool { return false }
	}
	if o.PrintConfig {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(sim.DefaultConfig()); err != nil {
			return nil, fmt.Errorf("print config: %w", err)
		}
		return nil, nil
	}

	cfg := sim.DefaultConfig()
	// Precedence: built-in defaults < config file < explicitly-set CLI flags.
	if o.ConfigPath != "" {
		c, err := sim.LoadConfig(o.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
		cfg = c
	}
	if changed("width") {
		cfg.Width = o.Width
	}
	if changed("height") {
		cfg.Height = o.Height
	}
	if changed("plants") {
		cfg.Plants = o.Plants
	}
	if changed("herbivores") {
		cfg.Herbivores = o.Herbivores
	}
	if changed("carnivores") {
		cfg.Carnivores = o.Carnivores
	}
	if changed("rescue") {
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
			return nil, fmt.Errorf("load snapshot: %w", err)
		}
		world = sim.NewWorldFromSnapshot(rng, snap)
		cfg.Width, cfg.Height = world.W, world.H
	} else {
		world = sim.NewWorld(rng, cfg)
	}
	return world, nil
}
