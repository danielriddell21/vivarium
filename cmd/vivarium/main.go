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

	Each agent is steered by a tiny hand-rolled recurrent neural network (19
	sensory inputs -> 12 hidden -> 4 outputs, tanh activations, no ML library).
	The hidden layer is fed its own previous activations (an Elman-style memory),
	so behaviour can depend on the recent past rather than being a pure reflex.
	Inputs come from directional vision: a ring of sectors around the agent, each
	reporting the proximity of the nearest target (food/prey) and nearest threat
	(predator), plus the signal broadcast by the nearest same-kind neighbour, and
	the agent's own energy. Outputs are turn, speed, an eat decision, and a signal.

Evolution

	When an agent's energy crosses a threshold it reproduces, splitting its energy
	with an offspring that inherits a CLONE of the parent's genome (brain weights)
	with small Gaussian mutations. Morphological traits (size, max speed, sense
	radius) and a learning rate co-evolve via the same mutate-on-inherit rule. Over
	generations the population drifts toward viable strategies (foraging, seeking,
	fleeing).

In-lifetime learning

	On top of evolution, each brain also LEARNS during its own life. Every tick the
	change in energy (eating = reward, costs/harm = penalty) drives a reward-
	modulated Hebbian update of a working copy of the weights, so behaviour that
	preceded reward is reinforced. The learning rate is the evolved Plasticity
	trait (0 = a fixed brain). Learning is Baldwinian: offspring inherit the GENOME,
	not what a parent learned, so each individual must learn anew — letting learning
	and evolution be studied separately.

Curiosity

	Each agent also learns a small forward (world) model that predicts its next
	senses from its current senses and action, trained online by gradient descent.
	The prediction error is the "surprise" of a transition and is added, scaled by
	the evolved Curiosity trait, to the reward above — an intrinsic motivation to
	seek novel, not-yet-predictable situations. As the model learns a region the
	surprise there fades, so curiosity moves on from the familiar.

Communication

	Every agent broadcasts a scalar signal (its 4th output) that same-kind
	neighbours hear through a "voice" vision channel (the nearest conspecific's
	signal per sector). Signals are double-buffered so all agents hear the previous
	tick's broadcast regardless of update order. Nothing rewards signalling
	directly, but because the channel feeds the evolving, learning brains, uses
	like alarm calls, flocking, or coordinated movement can emerge. Active
	broadcasts show in-world as a faint halo (warm = positive, cool = negative).

Controls

	space    pause / resume
	+ or =   double simulation speed (steps per frame)
	-        halve simulation speed
	click    select the nearest agent and open its inspector panel
	g        toggle the species view: colour agents by k-means cluster in
	         brain-genome space and show a 2D PCA map of the population
	l        toggle the lineage view: colour agents by lineage and show a
	         stacked chart of lineage abundance over time
	p        toggle the phylogeny view: a coalescent genealogy tree of the
	         living population (time runs left to right)
	h        toggle the communication halos drawn around broadcasting agents

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
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"os"

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
	configPath := flag.String("config", "", "path to a JSON config file overriding defaults (see -print-config)")
	printConfig := flag.Bool("print-config", false, "print the default config as JSON and exit")
	flag.Parse()

	if *printConfig {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(sim.DefaultConfig()); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Precedence: built-in defaults < config file < explicitly-set CLI flags.
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			log.Fatalf("reading config: %v", err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Fatalf("parsing config %s: %v", *configPath, err)
		}
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "width":
			cfg.Width = *width
		case "height":
			cfg.Height = *height
		case "plants":
			cfg.Plants = *plants
		case "herbivores":
			cfg.Herbivores = *herbivores
		case "carnivores":
			cfg.Carnivores = *carnivores
		case "rescue":
			cfg.Rescue = *rescue
		}
	})
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
