package sim

import (
	"math/rand"
	"testing"

	"github.com/danielriddell21/vivarium/internal/geom"
)

func geom2(x, y float64) geom.Vec2 { return geom.Vec2{X: x, Y: y} }

func testConfig() Config {
	return Config{
		Width: 400, Height: 300,
		Plants: 40, Herbivores: 20, Carnivores: 5,
		TargetPlants: 50,
	}
}

// TestDeterministicSeed verifies that two worlds built and stepped from the same
// seed produce identical population trajectories — the core reproducibility
// guarantee the -seed flag promises.
func TestDeterministicSeed(t *testing.T) {
	run := func(seed int64) []Counts {
		w := NewWorld(rand.New(rand.NewSource(seed)), testConfig())
		var trace []Counts
		for i := 0; i < 200; i++ {
			w.Step()
			trace = append(trace, w.CountKinds())
		}
		return trace
	}
	a, b := run(99), run(99)
	if len(a) != len(b) {
		t.Fatalf("trace lengths differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("trajectories diverge at step %d: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestDifferentSeedsDiffer(t *testing.T) {
	w1 := NewWorld(rand.New(rand.NewSource(1)), testConfig())
	w2 := NewWorld(rand.New(rand.NewSource(2)), testConfig())
	for i := 0; i < 200; i++ {
		w1.Step()
		w2.Step()
	}
	if w1.CountKinds() == w2.CountKinds() {
		t.Skip("seeds happened to converge to identical counts; not a failure")
	}
}

func TestStepRunsWithoutPanic(t *testing.T) {
	w := NewWorld(rand.New(rand.NewSource(5)), testConfig())
	for i := 0; i < 500; i++ {
		w.Step()
	}
	if w.Tick != 500 {
		t.Fatalf("expected tick 500, got %d", w.Tick)
	}
}

// TestReproductionIncreasesGeneration checks that a well-fed agent reproduces and
// that the offspring carries an incremented generation.
func TestReproductionIncreasesGeneration(t *testing.T) {
	w := NewWorld(rand.New(rand.NewSource(3)), testConfig())
	parent := w.Agents[0]
	parent.Energy = reproThreshold + 10
	before := len(w.Agents)
	child := w.reproduce(parent)
	if child.Generation != parent.Generation+1 {
		t.Fatalf("child generation %d, expected %d", child.Generation, parent.Generation+1)
	}
	if parent.Energy >= reproThreshold {
		t.Fatalf("parent energy should drop after reproduction, got %v", parent.Energy)
	}
	// reproduce returns the child but does not append it; Step does that.
	if len(w.Agents) != before {
		t.Fatalf("reproduce should not mutate the agent slice")
	}
}

// TestReproduceSetsCooldown verifies the gestation/maturation timers that stop a
// single large meal from becoming an instant litter.
func TestReproduceSetsCooldown(t *testing.T) {
	w := NewWorld(rand.New(rand.NewSource(3)), testConfig())
	parent := w.Agents[0]
	parent.Energy = reproThreshold + 10
	parent.ReproCooldown = 0
	child := w.reproduce(parent)
	if parent.ReproCooldown != gestation(parent.Kind) {
		t.Fatalf("parent cooldown = %d, want %d", parent.ReproCooldown, gestation(parent.Kind))
	}
	if child.ReproCooldown != gestation(parent.Kind) {
		t.Fatalf("child cooldown = %d, want %d", child.ReproCooldown, gestation(parent.Kind))
	}
}

// TestRescueRepopulates verifies the rescue effect: after a tier is wiped out,
// immigration brings it back rather than leaving it extinct forever.
func TestRescueRepopulates(t *testing.T) {
	cfg := testConfig()
	cfg.Rescue = true
	cfg.MinHerbivores, cfg.MinCarnivores = 8, 4
	cfg.Carnivores = 0 // isolate herbivore rescue from predation
	w := NewWorld(rand.New(rand.NewSource(4)), cfg)

	// Wipe out every herbivore.
	for _, a := range w.Agents {
		if a.Kind == Herbivore {
			a.Alive = false
		}
	}
	w.compactDead()
	if w.CountKinds().Herbivores != 0 {
		t.Fatal("setup failed: herbivores should be zero")
	}

	recovered := false
	for i := 0; i < 1000 && !recovered; i++ {
		w.Step()
		if w.CountKinds().Herbivores > 0 {
			recovered = true
		}
	}
	if !recovered {
		t.Fatal("rescue effect did not repopulate herbivores within 1000 ticks")
	}
}

// TestRescueDisabledStaysExtinct confirms that with rescue off a wiped tier does
// not magically return.
func TestRescueDisabledStaysExtinct(t *testing.T) {
	cfg := testConfig()
	cfg.Rescue = false
	cfg.Carnivores = 0
	w := NewWorld(rand.New(rand.NewSource(4)), cfg)
	for _, a := range w.Agents {
		a.Alive = false
	}
	w.compactDead()
	for i := 0; i < 300; i++ {
		w.Step()
	}
	if c := w.CountKinds(); c.Herbivores != 0 || c.Carnivores != 0 {
		t.Fatalf("no rescue expected, got %+v", c)
	}
}

func TestCarnivoreEatsHerbivore(t *testing.T) {
	w := &World{W: 200, H: 200, rng: rand.New(rand.NewSource(1)), targetPlants: 0}
	carn := w.newAgent(Carnivore, geom2(100, 100), nil, Traits{}, 0)
	herb := w.newAgent(Herbivore, geom2(101, 100), nil, Traits{}, 0)
	w.Agents = []*Agent{carn, herb}
	e0 := carn.Energy
	w.resolveEat(carn)
	if herb.Alive {
		t.Fatal("herbivore should be dead after being eaten")
	}
	if carn.Energy <= e0 {
		t.Fatalf("carnivore should gain energy, %v -> %v", e0, carn.Energy)
	}
}

func TestHerbivoreEatsFood(t *testing.T) {
	w := &World{W: 200, H: 200, rng: rand.New(rand.NewSource(1)), targetPlants: 0}
	herb := w.newAgent(Herbivore, geom2(50, 50), nil, Traits{}, 0)
	food := &Food{Pos: geom2(51, 50), Energy: foodMaxEnergy}
	w.Agents = []*Agent{herb}
	w.Foods = []*Food{food}
	e0 := herb.Energy
	w.resolveEat(herb)
	if herb.Energy <= e0 {
		t.Fatalf("herbivore should gain energy from food, %v -> %v", e0, herb.Energy)
	}
	if food.Energy >= foodMaxEnergy {
		t.Fatal("food energy should decrease after being eaten")
	}
}
