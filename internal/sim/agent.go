package sim

import (
	"math"

	"github.com/danielriddell21/vivarium/internal/geom"
	"github.com/danielriddell21/vivarium/internal/neural"
)

// Kind distinguishes the two trophic tiers of mobile agents. Plants are modelled
// separately as Food, not as agents.
type Kind int

const (
	Herbivore Kind = iota
	Carnivore
)

func (k Kind) String() string {
	if k == Carnivore {
		return "Carnivore"
	}
	return "Herbivore"
}

// Sensing & neural network dimensions shared by every brain.
//
// Vision is a ring of VisionSectors equal-angle sectors covering the full 360°
// around the agent, oriented relative to its heading (sector 0 points straight
// ahead, sectors proceed counter-clockwise). Each sector reports the proximity
// (0 = nothing/at the sense-radius edge, 1 = adjacent) of the nearest thing it
// sees, on two channels:
//
//	target channel — what the agent eats (plants for herbivores, herbivores for
//	                 carnivores)
//	threat channel — what eats it (carnivores for herbivores; always empty for
//	                 carnivores, which are apex predators)
//
// Inputs are laid out as:
//
//	[0 .. VisionSectors)               target-channel proximity per sector
//	[VisionSectors .. 2*VisionSectors) threat-channel proximity per sector
//	[2*VisionSectors]                  own energy (normalised, ~0..1)
//
// Outputs:
//
//	0: turn   (-1..1, scaled to a max turn per tick)
//	1: speed  (-1..1, mapped to 0..MaxSpeed)
//	2: eat    (>0 means attempt to feed this tick)
const (
	VisionSectors = 6
	BrainInputs   = VisionSectors*2 + 1
	BrainHidden   = 10
	BrainOutputs  = 3
)

// Metabolism. Carnivores have ~3x the upkeep of herbivores, so when prey is
// scarce they starve quickly. This predator self-limitation is what damps the
// boom-bust collapse into coexistence.
const (
	maxTurnPerTick = 0.45  // radians
	herbBasalCost  = 0.06  // herbivore upkeep per tick
	carnBasalCost  = 0.20  // carnivore upkeep per tick
	moveCost       = 0.04  // additional energy lost per unit of speed
	maxEnergy      = 200.0 // hard cap so a big meal can't be hoarded into many births
)

// basalCost returns the per-tick upkeep for the agent's kind.
func (a *Agent) basalCost() float64 {
	if a.Kind == Carnivore {
		return carnBasalCost
	}
	return herbBasalCost
}

// Agent is a mobile organism driven by an evolved neural brain.
type Agent struct {
	ID         int
	Kind       Kind
	Pos        geom.Vec2
	Heading    float64 // radians
	Energy     float64
	Age        int
	Generation int

	Brain  *neural.Brain
	Traits Traits
	Alive  bool

	// ReproCooldown is a gestation/maturation timer: an agent can only
	// reproduce when it reaches zero. It prevents a predator from converting one
	// large meal into an instant litter, which is the other half of the fix for
	// runaway carnivore booms.
	ReproCooldown int

	// Cached I/O from the most recent tick, surfaced by the inspector panel.
	// LastMemory is the brain's recurrent state (previous hidden activations).
	LastInputs  []float64
	LastOutputs []float64
	LastMemory  []float64
}

// sense builds the brain input vector by casting the agent's vision over the
// world: every visible target and threat within the sense radius is binned into
// the directional sector it falls in, keeping the nearest (highest proximity) per
// sector. Candidates come from the spatial grid, so the scan cost depends on local
// density rather than total population. See the input-layout comment above.
func (a *Agent) sense(w *World) []float64 {
	in := make([]float64, BrainInputs)
	in[2*VisionSectors] = clamp(a.Energy/reproThreshold, 0, 1.5)

	// These sub-slices share backing storage with in, so writing to them fills
	// the corresponding input ranges directly.
	target := in[0:VisionSectors]
	threat := in[VisionSectors : 2*VisionSectors]
	r := a.Traits.SenseRadius

	switch a.Kind {
	case Herbivore:
		w.grid.forEachFoodNear(a.Pos, r, func(f *Food) {
			if f.Ripe() {
				a.see(w, f.Pos, target)
			}
		})
		w.grid.forEachAgentNear(a.Pos, r, func(o *Agent) {
			if o.Alive && o.Kind == Carnivore && o.ID != a.ID {
				a.see(w, o.Pos, threat)
			}
		})
	case Carnivore:
		w.grid.forEachAgentNear(a.Pos, r, func(o *Agent) {
			if o.Alive && o.Kind == Herbivore && o.ID != a.ID {
				a.see(w, o.Pos, target)
			}
		})
		// Carnivores are apex predators here: no threat channel.
	}
	return in
}

// see records the proximity of point p into whichever vision sector it lies in,
// relative to the agent's heading, keeping the maximum (nearest) per sector.
// Points beyond the sense radius are ignored.
func (a *Agent) see(w *World, p geom.Vec2, sectors []float64) {
	d := a.Pos.ShortestDelta(p, w.W, w.H)
	dist := d.Len()
	if dist == 0 || dist > a.Traits.SenseRadius {
		return
	}
	rel := math.Mod(d.Angle()-a.Heading, 2*math.Pi)
	if rel < 0 {
		rel += 2 * math.Pi
	}
	sec := int(rel / (2 * math.Pi / VisionSectors))
	if sec >= VisionSectors { // guard the boundary case rel ≈ 2π
		sec = VisionSectors - 1
	}
	if prox := 1 - dist/a.Traits.SenseRadius; prox > sectors[sec] {
		sectors[sec] = prox
	}
}

// think runs the brain on the current senses and caches the I/O for inspection.
// The brain is recurrent, so each call also advances its internal memory.
func (a *Agent) think(w *World) []float64 {
	in := a.sense(w)
	out := a.Brain.Forward(in)
	a.LastInputs, a.LastOutputs, a.LastMemory = in, out, a.Brain.State()
	return out
}

// act applies the brain outputs: it turns, moves (paying energy), and reports
// whether the agent wants to eat this tick.
func (a *Agent) act(w *World, out []float64) (wantsEat bool) {
	a.Heading += out[0] * maxTurnPerTick
	a.Heading = math.Mod(a.Heading, 2*math.Pi)

	speed := (out[1] + 1) / 2 * a.Traits.MaxSpeed
	if speed < 0 {
		speed = 0
	}
	a.Pos = a.Pos.Add(geom.FromAngle(a.Heading).Scale(speed)).WrapTo(w.W, w.H)

	a.Energy -= a.basalCost() + moveCost*speed
	if a.Energy > maxEnergy {
		a.Energy = maxEnergy
	}
	if a.ReproCooldown > 0 {
		a.ReproCooldown--
	}
	a.Age++
	return out[2] > 0
}
