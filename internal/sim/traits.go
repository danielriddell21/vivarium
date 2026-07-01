package sim

import (
	"cmp"
	"math"
	"math/rand"
)

type Traits struct {
	Size        float64
	MaxSpeed    float64
	SenseRadius float64
	Plasticity  float64
	Curiosity   float64
	Diet        float64
}

func (t Traits) crossover(rng *rand.Rand, o Traits) Traits {
	pick := func(a, b float64) float64 {
		if rng.Float64() < 0.5 {
			return a
		}
		return b
	}
	return Traits{
		Size:        pick(t.Size, o.Size),
		MaxSpeed:    pick(t.MaxSpeed, o.MaxSpeed),
		SenseRadius: pick(t.SenseRadius, o.SenseRadius),
		Plasticity:  pick(t.Plasticity, o.Plasticity),
		Curiosity:   pick(t.Curiosity, o.Curiosity),
		Diet:        pick(t.Diet, o.Diet),
	}
}

func edibility(diet float64, foodType int) float64 {
	tv := 0.0
	if NumFoodTypes > 1 {
		tv = float64(foodType) / float64(NumFoodTypes-1)
	}
	e := 1 - math.Abs(diet-tv)
	if e < 0 {
		e = 0
	}
	return e
}

const (
	minSize, maxSize           = 2.0, 9.0
	minSpeed, maxSpeed         = 0.4, 3.5
	minSense, maxSense         = 30.0, 220.0
	minPlast, maxPlast         = 0.0, 0.04
	minCurio, maxCurio         = 0.0, 1.5
	traitMutationStdFractional = 0.12
)

func clamp[T cmp.Ordered](v, lo, hi T) T {
	return max(lo, min(hi, v))
}

func defaultTraits(rng *rand.Rand, k Kind) Traits {
	jitter := func(base float64) float64 { return base * (0.85 + 0.3*rng.Float64()) }
	switch k {
	case Carnivore:
		return Traits{
			Size:        clamp(jitter(5.5), minSize, maxSize),
			MaxSpeed:    clamp(jitter(2.2), minSpeed, maxSpeed),
			SenseRadius: clamp(jitter(140), minSense, maxSense),
			Plasticity:  clamp(jitter(0.01), minPlast, maxPlast),
			Curiosity:   clamp(jitter(0.3), minCurio, maxCurio),
			Diet:        rng.Float64(), // diverse initial diets so niches can emerge
		}
	default: // Herbivore
		return Traits{
			Size:        clamp(jitter(4.0), minSize, maxSize),
			MaxSpeed:    clamp(jitter(1.6), minSpeed, maxSpeed),
			SenseRadius: clamp(jitter(110), minSense, maxSense),
			Plasticity:  clamp(jitter(0.01), minPlast, maxPlast),
			Curiosity:   clamp(jitter(0.3), minCurio, maxCurio),
			Diet:        rng.Float64(), // diverse initial diets so niches can emerge
		}
	}
}

func (t Traits) mutated(rng *rand.Rand) Traits {
	nudge := func(v, lo, hi float64) float64 {
		v += rng.NormFloat64() * traitMutationStdFractional * v
		return clamp(v, lo, hi)
	}
	return Traits{
		Size:        nudge(t.Size, minSize, maxSize),
		MaxSpeed:    nudge(t.MaxSpeed, minSpeed, maxSpeed),
		SenseRadius: nudge(t.SenseRadius, minSense, maxSense),
		// Plasticity and Curiosity use absolute perturbations so a lineage can
		// evolve them on from zero (or back off) rather than being stuck once at 0.
		Plasticity: clamp(t.Plasticity+rng.NormFloat64()*0.004, minPlast, maxPlast),
		Curiosity:  clamp(t.Curiosity+rng.NormFloat64()*0.1, minCurio, maxCurio),
		Diet:       clamp(t.Diet+rng.NormFloat64()*0.05, 0, 1),
	}
}
