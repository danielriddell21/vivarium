/*
Vivarium is a small 2D ecosystem simulation in which behaviour evolves rather
than being programmed.

Model

	The world is a toroidal (wrap-around) plane with three trophic tiers:
	  - Plants (food): stationary patches that regrow energy over time.
	  - Herbivores: agents that eat plants.
	  - Carnivores: agents that eat herbivores.
	Every agent has an energy budget. Energy drains each tick (basal cost) and
	with movement; eating restores it. An agent dies when energy reaches zero.

Brain

	Each agent is steered by a tiny hand-rolled feedforward neural network (7
	sensory inputs -> 8 hidden -> 3 outputs, tanh activations, no ML library).
	Inputs encode normalised energy and the relative bearing/proximity of the
	nearest target (food/prey) and nearest threat (predator). Outputs are turn,
	speed, and an eat decision.

Evolution

	There is no backpropagation. When an agent's energy crosses a threshold it
	reproduces, splitting its energy with an offspring that inherits a CLONE of
	the parent's brain with small Gaussian weight mutations. Morphological traits
	(size, max speed, sense radius) co-evolve via the same mutate-on-inherit rule.
	Over generations the population drifts toward viable strategies (foraging,
	seeking, fleeing).

Controls

	space    pause / resume
	+ or =   double simulation speed (steps per frame)
	-        halve simulation speed
	click    select the nearest agent and open its inspector panel

Overlays

	A live line chart shows plant/herbivore/carnivore counts over time, a HUD
	shows run state and counts, and the inspector shows a selected agent's energy,
	age, generation, traits, and current neural inputs/outputs.

Reproducibility

	All randomness is drawn from a single seeded generator; pass -seed for
	deterministic runs.
*/
package main

import (
	"flag"
	"log"
	"math/rand"

	"github.com/danielriddell21/vivarium/internal/render"
	"github.com/danielriddell21/vivarium/internal/sim"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	cfg := sim.DefaultConfig()
	seed := flag.Int64("seed", 1, "random seed for reproducible runs")
	width := flag.Float64("width", cfg.Width, "world width in pixels")
	height := flag.Float64("height", cfg.Height, "world height in pixels")
	plants := flag.Int("plants", cfg.Plants, "initial plant count")
	herbivores := flag.Int("herbivores", cfg.Herbivores, "initial herbivore count")
	carnivores := flag.Int("carnivores", cfg.Carnivores, "initial carnivore count")
	rescue := flag.Bool("rescue", cfg.Rescue, "rescue effect: immigrants arrive when a tier nears extinction")
	flag.Parse()

	cfg.Width, cfg.Height = *width, *height
	cfg.Plants, cfg.Herbivores, cfg.Carnivores = *plants, *herbivores, *carnivores
	cfg.Rescue = *rescue
	if cfg.TargetPlants < cfg.Plants {
		cfg.TargetPlants = cfg.Plants
	}

	rng := rand.New(rand.NewSource(*seed))
	world := sim.NewWorld(rng, cfg)
	game := render.NewGame(world)

	ebiten.SetWindowSize(int(cfg.Width), int(cfg.Height))
	ebiten.SetWindowTitle("Vivarium — evolving ecosystem")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
