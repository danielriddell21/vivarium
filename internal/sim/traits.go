package sim

import "math/rand"

// Traits are the heritable scalar (morphological) properties of an agent that
// co-evolve alongside its neural brain. Like brain weights, they are passed to
// offspring with small mutations, so body plan and behaviour adapt together.
type Traits struct {
	Size        float64 // radius in pixels; affects eating reach and draw size
	MaxSpeed    float64 // maximum movement speed (pixels/tick)
	SenseRadius float64 // how far the agent can perceive food/prey/predators
	Plasticity  float64 // in-lifetime learning rate (0 = a fixed, non-learning brain)
}

// Trait bounds keep mutated morphologies physically sensible.
const (
	minSize, maxSize           = 2.0, 9.0
	minSpeed, maxSpeed         = 0.4, 3.5
	minSense, maxSense         = 30.0, 220.0
	minPlast, maxPlast         = 0.0, 0.04
	traitMutationStdFractional = 0.12 // mutation std as a fraction of the value
)

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// defaultTraits returns the starting morphology for a freshly spawned agent of
// the given kind, with a little per-individual randomness from rng.
func defaultTraits(rng *rand.Rand, k Kind) Traits {
	jitter := func(base float64) float64 { return base * (0.85 + 0.3*rng.Float64()) }
	switch k {
	case Carnivore:
		return Traits{
			Size:        clamp(jitter(5.5), minSize, maxSize),
			MaxSpeed:    clamp(jitter(2.2), minSpeed, maxSpeed),
			SenseRadius: clamp(jitter(140), minSense, maxSense),
			Plasticity:  clamp(jitter(0.01), minPlast, maxPlast),
		}
	default: // Herbivore
		return Traits{
			Size:        clamp(jitter(4.0), minSize, maxSize),
			MaxSpeed:    clamp(jitter(1.6), minSpeed, maxSpeed),
			SenseRadius: clamp(jitter(110), minSense, maxSense),
			Plasticity:  clamp(jitter(0.01), minPlast, maxPlast),
		}
	}
}

// mutated returns a copy of t with each trait perturbed by Gaussian noise and
// clamped to its allowed range.
func (t Traits) mutated(rng *rand.Rand) Traits {
	nudge := func(v, lo, hi float64) float64 {
		v += rng.NormFloat64() * traitMutationStdFractional * v
		return clamp(v, lo, hi)
	}
	return Traits{
		Size:        nudge(t.Size, minSize, maxSize),
		MaxSpeed:    nudge(t.MaxSpeed, minSpeed, maxSpeed),
		SenseRadius: nudge(t.SenseRadius, minSense, maxSense),
		// Plasticity uses an absolute perturbation so a lineage can evolve learning
		// on from zero (or back off it) rather than being stuck once it hits 0.
		Plasticity: clamp(t.Plasticity+rng.NormFloat64()*0.004, minPlast, maxPlast),
	}
}
