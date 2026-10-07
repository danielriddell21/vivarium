package sim

import (
	"math"
	"math/rand"
	"testing"
)

func TestEdibility(t *testing.T) {
	last := NumFoodTypes - 1
	cases := []struct {
		name     string
		diet     float64
		foodType int
		want     float64
	}{
		{"specialist on its own food", 0, 0, 1},
		{"specialist on the other end", 0, last, 0},
		{"the other specialist", 1, last, 1},
		{"a generalist halfway", 0.5, 0, 0.5},
	}
	for _, c := range cases {
		if got := edibility(c.diet, c.foodType); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: edibility(%v, %d) = %v, want %v", c.name, c.diet, c.foodType, got, c.want)
		}
	}
	for diet := 0.0; diet <= 1; diet += 0.1 {
		for ft := 0; ft < NumFoodTypes; ft++ {
			if e := edibility(diet, ft); e < 0 || e > 1 {
				t.Fatalf("edibility(%v, %d) = %v, want within [0, 1]", diet, ft, e)
			}
		}
	}
}

func TestClamp(t *testing.T) {
	if clamp(-1, 0, 10) != 0 || clamp(11, 0, 10) != 10 || clamp(4, 0, 10) != 4 {
		t.Error("clamp does not hold values to [lo, hi]")
	}
}

func inBounds(t *testing.T, tr Traits) {
	t.Helper()
	check := func(name string, v, lo, hi float64) {
		if v < lo || v > hi {
			t.Errorf("%s = %v, want within [%v, %v]", name, v, lo, hi)
		}
	}
	check("Size", tr.Size, minSize, maxSize)
	check("MaxSpeed", tr.MaxSpeed, minSpeed, maxSpeed)
	check("SenseRadius", tr.SenseRadius, minSense, maxSense)
	check("Plasticity", tr.Plasticity, minPlast, maxPlast)
	check("Curiosity", tr.Curiosity, minCurio, maxCurio)
	check("Diet", tr.Diet, 0, 1)
}

func TestDefaultTraitsSuitTheKind(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for range 200 {
		c := defaultTraits(rng, Carnivore)
		h := defaultTraits(rng, Herbivore)
		inBounds(t, c)
		inBounds(t, h)
		// The jitter is ±15%, which never closes these gaps.
		if c.Size <= h.Size {
			t.Fatalf("carnivore size %v not above herbivore %v", c.Size, h.Size)
		}
		if c.MaxSpeed <= h.MaxSpeed {
			t.Fatalf("carnivore speed %v not above herbivore %v", c.MaxSpeed, h.MaxSpeed)
		}
		if c.Size < 5.5*0.85 || c.Size > 5.5*1.15 {
			t.Fatalf("carnivore size %v is outside 5.5 ± 15%%", c.Size)
		}
		if h.SenseRadius < 110*0.85 || h.SenseRadius > 110*1.15 {
			t.Fatalf("herbivore sense %v is outside 110 ± 15%%", h.SenseRadius)
		}
	}
}

func TestCrossoverTakesEachTraitFromAParent(t *testing.T) {
	a := Traits{Size: 2, MaxSpeed: 0.5, SenseRadius: 40, Plasticity: 0.001, Curiosity: 0.1, Diet: 0}
	b := Traits{Size: 8, MaxSpeed: 3, SenseRadius: 200, Plasticity: 0.03, Curiosity: 1.2, Diet: 1}
	rng := rand.New(rand.NewSource(7))
	fromA, fromB := 0, 0
	for range 200 {
		c := a.crossover(rng, b)
		for i, v := range []float64{c.Size, c.MaxSpeed, c.SenseRadius, c.Plasticity, c.Curiosity, c.Diet} {
			pa := []float64{a.Size, a.MaxSpeed, a.SenseRadius, a.Plasticity, a.Curiosity, a.Diet}[i]
			pb := []float64{b.Size, b.MaxSpeed, b.SenseRadius, b.Plasticity, b.Curiosity, b.Diet}[i]
			switch v {
			case pa:
				fromA++
			case pb:
				fromB++
			default:
				t.Fatalf("child trait %d = %v, from neither parent (%v, %v)", i, v, pa, pb)
			}
		}
	}
	// Each pick is a fair coin, so both parents contribute about half.
	if fromA < 500 || fromB < 500 {
		t.Errorf("parents contributed %d and %d of 1200 traits, want roughly even", fromA, fromB)
	}
}

func TestMutatedStaysInBoundsAndMoves(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	start := Traits{Size: 5, MaxSpeed: 2, SenseRadius: 120, Plasticity: 0, Curiosity: 0, Diet: 0.5}
	tr := start
	for range 500 {
		tr = tr.mutated(rng)
		inBounds(t, tr)
	}
	if tr.Size == start.Size || tr.MaxSpeed == start.MaxSpeed || tr.SenseRadius == start.SenseRadius || tr.Diet == start.Diet {
		t.Errorf("500 mutations left a trait unchanged: %+v", tr)
	}
	// Absolute perturbations let a lineage leave zero.
	if tr.Plasticity == 0 && tr.Curiosity == 0 {
		t.Errorf("plasticity and curiosity never left zero: %+v", tr)
	}
}

func TestMutationIsSmall(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	start := Traits{Size: 5, MaxSpeed: 2, SenseRadius: 120, Plasticity: 0.02, Curiosity: 0.7, Diet: 0.5}
	for range 200 {
		m := start.mutated(rng)
		// A fractional step of 12% std rarely exceeds 60% (5 sigma).
		if math.Abs(m.Size-start.Size) > 0.6*start.Size {
			t.Fatalf("size jumped from %v to %v", start.Size, m.Size)
		}
		if math.Abs(m.Diet-start.Diet) > 0.25 {
			t.Fatalf("diet jumped from %v to %v", start.Diet, m.Diet)
		}
	}
}
