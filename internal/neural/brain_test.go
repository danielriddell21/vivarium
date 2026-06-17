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
	b.Reset() // clear memory so the second pass starts from the same state
	c := b.Forward(in)
	for i := range a {
		if a[i] != c[i] {
			t.Fatalf("forward not deterministic at %d: %v vs %v", i, a[i], c[i])
		}
	}
}

// TestRecurrenceCarriesState verifies the network is stateful: feeding the same
// input twice gives different outputs (the memory advanced), and Reset restores
// the original response.
func TestRecurrenceCarriesState(t *testing.T) {
	b := New(rand.New(rand.NewSource(13)), 4, 6, 2)
	in := []float64{0.5, -0.3, 0.2, 0.8}

	first := b.Forward(in)
	second := b.Forward(in)

	differ := false
	for i := range first {
		if first[i] != second[i] {
			differ = true
		}
	}
	if !differ {
		t.Fatal("expected recurrent state to change the output on the second pass")
	}

	b.Reset()
	afterReset := b.Forward(in)
	for i := range first {
		if first[i] != afterReset[i] {
			t.Fatalf("Reset should restore the initial response at %d: %v vs %v", i, first[i], afterReset[i])
		}
	}
}

// TestCloneResetsMemory checks that offspring do not inherit the parent's memory.
func TestCloneResetsMemory(t *testing.T) {
	b := New(rand.New(rand.NewSource(21)), 3, 4, 2)
	b.Forward([]float64{1, 1, 1}) // advance the parent's state
	cp := b.Clone()
	for i, v := range cp.State() {
		if v != 0 {
			t.Fatalf("clone memory[%d] = %v, want 0 (blank)", i, v)
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
	b.Reset() // compare from the same memory state, not the advanced one
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
