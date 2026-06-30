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
	before := append([]float64(nil), b.genome.WIH...)

	b.Mutate(rng, 1.0, 0.5) // rate 1.0 => every weight perturbed
	changed := 0
	for i := range b.genome.WIH {
		if math.Abs(b.genome.WIH[i]-before[i]) > 0 {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("expected genome weights to change after full-rate mutation")
	}
}

func TestMutateZeroRateIsNoOp(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	b := New(rng, 3, 3, 2)
	before := append([]float64(nil), b.genome.WHO...)
	b.Mutate(rng, 0.0, 1.0)
	for i := range b.genome.WHO {
		if b.genome.WHO[i] != before[i] {
			t.Fatalf("zero-rate mutation changed weight %d", i)
		}
	}
}

func TestLearnModifiesLiveNotGenome(t *testing.T) {
	b := New(rand.New(rand.NewSource(5)), 4, 5, 2)
	genomeBefore := append([]float64(nil), b.genome.WHO...)

	b.Forward([]float64{0.6, -0.4, 0.9, 0.1})
	b.Learn(1.0, 0.1) // positive reward, non-zero rate

	if b.LearnedDrift() <= 0 {
		t.Fatal("expected live weights to drift after learning")
	}
	for i := range b.genome.WHO {
		if b.genome.WHO[i] != genomeBefore[i] {
			t.Fatalf("genome weight %d changed during learning (should be immutable)", i)
		}
	}
}

func TestLearnReinforcesOutput(t *testing.T) {
	b := New(rand.New(rand.NewSource(8)), 4, 6, 2)
	in := []float64{0.8, -0.2, 0.5, 0.3}

	b.Reset()
	first := b.Forward(in)
	idx := 0
	if math.Abs(first[1]) > math.Abs(first[0]) {
		idx = 1
	}
	mag0 := math.Abs(first[idx])

	for i := 0; i < 50; i++ {
		out := b.Forward(in)
		// Reward in the sign of the current dominant output to reinforce it.
		b.Learn(math.Copysign(1, out[idx]), 0.05)
	}
	b.Reset()
	magN := math.Abs(b.Forward(in)[idx])

	if magN <= mag0 {
		t.Fatalf("expected reinforced output to grow: %.4f -> %.4f", mag0, magN)
	}
}

func TestFromGenomeRoundTrip(t *testing.T) {
	b := New(rand.New(rand.NewSource(4)), 7, 8, 3)
	in := []float64{0.2, -0.1, 0.5, 0.9, -0.3, 0.0, 0.7}
	want := b.Forward(in) // from blank state

	rebuilt := FromGenome(7, 8, 3, b.Genome())
	if rebuilt == nil {
		t.Fatal("FromGenome returned nil for a matching-length genome")
	}
	got := rebuilt.Forward(in)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rebuilt output %d = %v, want %v", i, got[i], want[i])
		}
	}
	if FromGenome(7, 8, 3, []float64{1, 2, 3}) != nil {
		t.Fatal("FromGenome should reject a wrong-length genome")
	}
}

func TestCrossoverMixesParents(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	a := New(rng, 4, 4, 2)
	b := New(rng, 4, 4, 2)
	child := a.CrossoverWith(rng, b)

	if child.LearnedDrift() != 0 {
		t.Fatal("a fresh child should have live == genome (no learning)")
	}
	for i := range child.genome.WIH {
		if child.genome.WIH[i] != a.genome.WIH[i] && child.genome.WIH[i] != b.genome.WIH[i] {
			t.Fatalf("child gene %d (%v) came from neither parent (%v / %v)", i, child.genome.WIH[i], a.genome.WIH[i], b.genome.WIH[i])
		}
	}
	// With random parents, at least some genes should differ from parent a.
	diff := 0
	for i := range child.genome.WIH {
		if child.genome.WIH[i] != a.genome.WIH[i] {
			diff++
		}
	}
	if diff == 0 {
		t.Fatal("child should inherit some genes from the other parent")
	}
}

func TestCloneInheritsGenomeNotLearning(t *testing.T) {
	b := New(rand.New(rand.NewSource(17)), 4, 4, 2)
	for i := 0; i < 30; i++ {
		b.Forward([]float64{1, -1, 0.5, 0.2})
		b.Learn(1.0, 0.1)
	}
	if b.LearnedDrift() == 0 {
		t.Fatal("setup: parent should have learned something")
	}
	child := b.Clone()
	if child.LearnedDrift() != 0 {
		t.Fatal("child should start with live == genome (no inherited learning)")
	}
	for i := range child.genome.WHO {
		if child.genome.WHO[i] != b.genome.WHO[i] {
			t.Fatalf("child genome should match parent genome at %d", i)
		}
	}
}
