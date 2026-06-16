package neural

import (
	"math"
	"math/rand"
	"testing"
)

func TestForwardShapeAndBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	b := New(rng, 8, 8, 3)
	out := b.Forward([]float64{0.5, -0.5, 1, -1, 0, 0.2, -0.2, 0.9})
	if len(out) != 3 {
		t.Fatalf("expected 3 outputs, got %d", len(out))
	}
	for i, o := range out {
		if o < -1 || o > 1 {
			t.Errorf("output %d = %v out of tanh range", i, o)
		}
	}
}

func TestForwardDeterministic(t *testing.T) {
	b := New(rand.New(rand.NewSource(42)), 4, 5, 2)
	in := []float64{0.1, 0.2, 0.3, 0.4}
	a := b.Forward(in)
	c := b.Forward(in)
	for i := range a {
		if a[i] != c[i] {
			t.Fatalf("forward not deterministic at %d: %v vs %v", i, a[i], c[i])
		}
	}
}

func TestForwardHandlesShortInput(t *testing.T) {
	b := New(rand.New(rand.NewSource(7)), 6, 4, 2)
	// Fewer inputs than In should not panic (missing inputs treated as zero).
	out := b.Forward([]float64{1, 2})
	if len(out) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(out))
	}
}

func TestCloneIsDeepAndIndependent(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	b := New(rng, 4, 4, 2)
	cp := b.Clone()

	in := []float64{0.3, -0.7, 0.1, 0.9}
	before := b.Forward(in)

	// Mutating the clone must not change the original's behaviour.
	cp.Mutate(rng, 1.0, 1.0)
	after := b.Forward(in)
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("mutating clone changed original output at %d", i)
		}
	}
}

func TestMutateChangesWeights(t *testing.T) {
	rng := rand.New(rand.NewSource(9))
	b := New(rng, 4, 4, 2)
	before := append([]float64(nil), b.WIH...)

	b.Mutate(rng, 1.0, 0.5) // rate 1.0 => every weight perturbed
	changed := 0
	for i := range b.WIH {
		if math.Abs(b.WIH[i]-before[i]) > 0 {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("expected weights to change after full-rate mutation")
	}
}

func TestMutateZeroRateIsNoOp(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	b := New(rng, 3, 3, 2)
	before := append([]float64(nil), b.WHO...)
	b.Mutate(rng, 0.0, 1.0)
	for i := range b.WHO {
		if b.WHO[i] != before[i] {
			t.Fatalf("zero-rate mutation changed weight %d", i)
		}
	}
}
