package sim

import (
	"math"
	"math/rand"

	"github.com/danielriddell21/vivarium/internal/geom"
	"github.com/danielriddell21/vivarium/internal/neural"
)

// Energy / reproduction tunables.
const (
	reproThreshold  = 120.0 // energy at which an agent reproduces
	startEnergyHerb = 70.0
	startEnergyCarn = 90.0
	carnivoreGain   = 0.5 // fraction of prey energy a carnivore absorbs
	carnivoreBonus  = 8.0 // flat energy bonus per kill; small so one kill alone
	//                       does not instantly fund a birth
	// Gestation/maturation cooldowns (ticks). Prey reproduce faster than
	// predators, as in real food webs, so herbivores can recover from predation.
	gestationHerb = 45
	gestationCarn = 110
	mutationRate  = 0.18 // per-weight probability of mutation in offspring
	mutationStd   = 0.35 // std of Gaussian weight perturbation
	historyEvery  = 6    // sample population counts every N ticks
	maxHistory    = 1200 // cap on retained history samples
)

// gestation returns the post-reproduction cooldown for the given kind.
func gestation(k Kind) int {
	if k == Carnivore {
		return gestationCarn
	}
	return gestationHerb
}

// rescueChance is the per-tick probability of one immigrant arriving for a tier
// that has fallen below its rescue floor.
const rescueChance = 0.05

// Config holds the initial-population and world-size parameters.
type Config struct {
	Width, Height                  float64
	Plants, Herbivores, Carnivores int
	TargetPlants                   int // plant count the world tries to maintain

	// Rescue enables a metapopulation "rescue effect": when a mobile tier drops
	// below its floor, occasional immigrants arrive so the ecosystem recovers
	// instead of collapsing to permanent extinction. Disable for raw dynamics.
	Rescue                       bool
	MinHerbivores, MinCarnivores int
}

// DefaultConfig returns a balanced starting configuration.
func DefaultConfig() Config {
	return Config{
		Width: 960, Height: 720,
		Plants: 200, Herbivores: 80, Carnivores: 8,
		TargetPlants:  260,
		Rescue:        true,
		MinHerbivores: 8,
		MinCarnivores: 4,
	}
}

// Counts is a snapshot of population sizes for one history sample.
type Counts struct {
	Plants, Herbivores, Carnivores int
}

// World holds all simulation state. A single *rand.Rand drives every random
// decision so that runs are fully reproducible from a seed.
type World struct {
	W, H   float64
	Foods  []*Food
	Agents []*Agent

	Tick   int
	rng    *rand.Rand
	nextID int

	targetPlants int
	rescue       bool
	minHerb      int
	minCarn      int
	history      []Counts
}

// NewWorld builds a world from cfg, seeding plants and agents at random
// positions using rng.
func NewWorld(rng *rand.Rand, cfg Config) *World {
	w := &World{
		W: cfg.Width, H: cfg.Height,
		rng:          rng,
		targetPlants: cfg.TargetPlants,
		rescue:       cfg.Rescue,
		minHerb:      cfg.MinHerbivores,
		minCarn:      cfg.MinCarnivores,
	}
	for i := 0; i < cfg.Plants; i++ {
		w.Foods = append(w.Foods, &Food{Pos: w.randPos(), Energy: foodMaxEnergy * rng.Float64()})
	}
	for i := 0; i < cfg.Herbivores; i++ {
		w.Agents = append(w.Agents, w.newAgent(Herbivore, w.randPos(), nil, Traits{}, 0))
	}
	for i := 0; i < cfg.Carnivores; i++ {
		w.Agents = append(w.Agents, w.newAgent(Carnivore, w.randPos(), nil, Traits{}, 0))
	}
	w.sampleHistory()
	return w
}

func (w *World) randPos() geom.Vec2 {
	return geom.Vec2{X: w.rng.Float64() * w.W, Y: w.rng.Float64() * w.H}
}

// newAgent constructs an agent. If parentBrain is nil a fresh random brain and
// default traits are created (founder agents); otherwise the brain/traits are
// inherited copies that the caller has already mutated.
func (w *World) newAgent(k Kind, pos geom.Vec2, brain *neural.Brain, traits Traits, gen int) *Agent {
	w.nextID++
	if brain == nil {
		brain = neural.New(w.rng, BrainInputs, BrainHidden, BrainOutputs)
		traits = defaultTraits(w.rng, k)
	}
	start := startEnergyHerb
	if k == Carnivore {
		start = startEnergyCarn
	}
	return &Agent{
		ID:         w.nextID,
		Kind:       k,
		Pos:        pos,
		Heading:    w.rng.Float64() * 2 * math.Pi,
		Energy:     start,
		Generation: gen,
		Brain:      brain,
		Traits:     traits,
		Alive:      true,
	}
}

// Step advances the simulation by one tick.
func (w *World) Step() {
	for _, f := range w.Foods {
		f.regrow()
	}

	var newborns []*Agent
	for _, a := range w.Agents {
		if !a.Alive {
			continue
		}
		out := a.think(w)
		if a.act(w, out) {
			w.resolveEat(a)
		}
		if a.Energy <= 0 {
			a.Alive = false
			continue
		}
		if a.ReproCooldown <= 0 && a.Energy >= reproThreshold {
			newborns = append(newborns, w.reproduce(a))
		}
	}
	w.Agents = append(w.Agents, newborns...)
	w.compactDead()
	w.maintainFood()
	w.maintainPopulations()

	w.Tick++
	if w.Tick%historyEvery == 0 {
		w.sampleHistory()
	}
}

// resolveEat handles a feeding attempt for one agent.
func (w *World) resolveEat(a *Agent) {
	switch a.Kind {
	case Herbivore:
		if f := w.nearestRipeFood(a.Pos, a.Traits.Size+foodEatRadius); f != nil {
			bite := math.Min(foodBiteEnergy, f.Energy)
			f.Energy -= bite
			a.Energy += bite
		}
	case Carnivore:
		reach := a.Traits.Size + foodEatRadius
		if prey := w.nearestAgentOfKind(a.Pos, Herbivore, reach, a.ID); prey != nil && prey.Alive {
			prey.Alive = false
			a.Energy += prey.Energy*carnivoreGain + carnivoreBonus
			if a.Energy > maxEnergy {
				a.Energy = maxEnergy
			}
		}
	}
}

// reproduce spawns a mutated offspring next to the parent and splits the
// parent's energy with it.
func (w *World) reproduce(parent *Agent) *Agent {
	child := parent.Energy / 2
	parent.Energy -= child

	brain := parent.Brain.Clone()
	brain.Mutate(w.rng, mutationRate, mutationStd)
	traits := parent.Traits.mutated(w.rng)

	parent.ReproCooldown = gestation(parent.Kind)

	off := w.newAgent(parent.Kind, parent.Pos, brain, traits, parent.Generation+1)
	off.Energy = child
	off.ReproCooldown = gestation(parent.Kind) // maturation: newborns can't breed immediately
	// Place the newborn a short random offset away so it does not perfectly
	// overlap its parent.
	off.Pos = off.Pos.Add(geom.FromAngle(w.rng.Float64()*2*math.Pi).Scale(parent.Traits.Size)).WrapTo(w.W, w.H)
	return off
}

// compactDead removes dead agents in place, preserving order.
func (w *World) compactDead() {
	keep := w.Agents[:0]
	for _, a := range w.Agents {
		if a.Alive {
			keep = append(keep, a)
		}
	}
	w.Agents = keep
}

// maintainFood tops the world up toward its target plant count, adding at most
// one plant per tick to keep regrowth gradual.
func (w *World) maintainFood() {
	if len(w.Foods) < w.targetPlants && w.rng.Float64() < 0.5 {
		w.Foods = append(w.Foods, &Food{Pos: w.randPos(), Energy: foodBiteEnergy})
	}
}

// maintainPopulations applies the rescue effect: when a mobile tier sits below
// its floor, immigrants occasionally arrive so the ecosystem can recover from a
// near-collapse rather than going extinct for good.
func (w *World) maintainPopulations() {
	if !w.rescue {
		return
	}
	c := w.CountKinds()
	if c.Herbivores < w.minHerb && w.rng.Float64() < rescueChance {
		w.immigrate(Herbivore)
	}
	if c.Carnivores < w.minCarn && w.rng.Float64() < rescueChance {
		w.immigrate(Carnivore)
	}
}

// immigrate introduces a single new agent of kind k. If any member of that tier
// survives, the newcomer descends from a random survivor (cloned, mutated brain
// and traits) so evolution continues; otherwise it arrives with a fresh brain.
func (w *World) immigrate(k Kind) {
	var brain *neural.Brain
	var traits Traits
	gen := 0
	if donor := w.randomAgentOfKind(k); donor != nil {
		brain = donor.Brain.Clone()
		brain.Mutate(w.rng, mutationRate, mutationStd)
		traits = donor.Traits.mutated(w.rng)
		gen = donor.Generation
	}
	a := w.newAgent(k, w.randPos(), brain, traits, gen)
	a.ReproCooldown = gestation(k) // immigrants must establish before breeding
	w.Agents = append(w.Agents, a)
}

// randomAgentOfKind returns a uniformly random living agent of kind k, or nil.
func (w *World) randomAgentOfKind(k Kind) *Agent {
	var chosen *Agent
	seen := 0
	for _, a := range w.Agents {
		if !a.Alive || a.Kind != k {
			continue
		}
		seen++
		if w.rng.Intn(seen) == 0 { // reservoir sampling, size 1
			chosen = a
		}
	}
	return chosen
}

// nearestRipeFood returns the closest ripe plant within radius of pos, or nil.
func (w *World) nearestRipeFood(pos geom.Vec2, radius float64) *Food {
	var best *Food
	bestD := radius
	for _, f := range w.Foods {
		if !f.Ripe() {
			continue
		}
		d := pos.ToroidalDist(f.Pos, w.W, w.H)
		if d <= bestD {
			bestD, best = d, f
		}
	}
	return best
}

// nearestAgentOfKind returns the closest living agent of kind k within radius of
// pos, excluding the agent with id excludeID, or nil.
func (w *World) nearestAgentOfKind(pos geom.Vec2, k Kind, radius float64, excludeID int) *Agent {
	var best *Agent
	bestD := radius
	for _, a := range w.Agents {
		if !a.Alive || a.Kind != k || a.ID == excludeID {
			continue
		}
		d := pos.ToroidalDist(a.Pos, w.W, w.H)
		if d <= bestD {
			bestD, best = d, a
		}
	}
	return best
}

// CountKinds returns the current population sizes.
func (w *World) CountKinds() Counts {
	c := Counts{Plants: len(w.Foods)}
	for _, a := range w.Agents {
		if !a.Alive {
			continue
		}
		if a.Kind == Carnivore {
			c.Carnivores++
		} else {
			c.Herbivores++
		}
	}
	return c
}

func (w *World) sampleHistory() {
	w.history = append(w.history, w.CountKinds())
	if len(w.history) > maxHistory {
		w.history = w.history[len(w.history)-maxHistory:]
	}
}

// History returns the retained population-count samples (oldest first).
func (w *World) History() []Counts { return w.history }

// NearestAgent returns the living agent closest to pos (ignoring wrap, since this
// is used for screen-space clicks), or nil if there are no living agents.
func (w *World) NearestAgent(pos geom.Vec2) *Agent {
	var best *Agent
	bestD := math.MaxFloat64
	for _, a := range w.Agents {
		if !a.Alive {
			continue
		}
		d := a.Pos.Sub(pos).Len()
		if d < bestD {
			bestD, best = d, a
		}
	}
	return best
}
