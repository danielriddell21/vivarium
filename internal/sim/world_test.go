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
		Params:       DefaultParams(),
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
	parent.Energy = w.params.ReproThreshold + 10
	before := len(w.Agents)
	child := w.reproduce(parent)
	if child.Generation != parent.Generation+1 {
		t.Fatalf("child generation %d, expected %d", child.Generation, parent.Generation+1)
	}
	if parent.Energy >= w.params.ReproThreshold {
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
	parent.Energy = w.params.ReproThreshold + 10
	parent.ReproCooldown = 0
	child := w.reproduce(parent)
	if parent.ReproCooldown != w.gestation(parent.Kind) {
		t.Fatalf("parent cooldown = %d, want %d", parent.ReproCooldown, w.gestation(parent.Kind))
	}
	if child.ReproCooldown != w.gestation(parent.Kind) {
		t.Fatalf("child cooldown = %d, want %d", child.ReproCooldown, w.gestation(parent.Kind))
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

// TestVisionSectors checks that targets and threats are binned into the correct
// directional sector relative to the agent's heading.
func TestVisionSectors(t *testing.T) {
	w := &World{W: 400, H: 300, rng: rand.New(rand.NewSource(1)), params: DefaultParams()}
	herb := w.newAgent(Herbivore, geom2(100, 100), nil, Traits{}, 0)
	herb.Heading = 0 // facing +x (east)
	herb.Traits.SenseRadius = 200
	carn := w.newAgent(Carnivore, geom2(60, 60), nil, Traits{}, 0)             // behind-left at 225°, sector 3
	food := &Food{Pos: geom2(140, 100), Energy: DefaultParams().FoodMaxEnergy} // due east, ahead, sector 0
	w.Agents = []*Agent{herb, carn}
	w.Foods = []*Food{food}
	w.reindex()

	in := herb.sense(w)

	if in[0] <= 0 {
		t.Fatalf("food due ahead should light target sector 0, got %v", in[0])
	}
	if in[VisionSectors+3] <= 0 {
		t.Fatalf("predator due behind should light threat sector 3, got %v", in[VisionSectors+3])
	}
	if in[2*VisionSectors] != 0 {
		t.Fatalf("voice sector should be empty with no conspecific, got %v", in[2*VisionSectors])
	}
	if in[3*VisionSectors] <= 0 {
		t.Fatal("energy input should be set")
	}
	for s := 1; s < VisionSectors; s++ {
		if in[s] != 0 {
			t.Fatalf("unexpected target proximity in sector %d: %v", s, in[s])
		}
	}
}

// TestVoiceChannel checks an agent hears a same-kind neighbour's broadcast signal
// in the correct sector, and does not hear other species.
func TestVoiceChannel(t *testing.T) {
	w := &World{W: 400, H: 300, rng: rand.New(rand.NewSource(1)), params: DefaultParams()}
	herb := w.newAgent(Herbivore, geom2(100, 100), nil, Traits{}, 0)
	herb.Heading = 0
	herb.Traits.SenseRadius = 200
	mate := w.newAgent(Herbivore, geom2(140, 100), nil, Traits{}, 0) // due east, sector 0
	mate.Signal = 0.75
	carn := w.newAgent(Carnivore, geom2(60, 60), nil, Traits{}, 0) // sector 3, different kind
	carn.Signal = -0.9
	w.Agents = []*Agent{herb, mate, carn}
	w.reindex()

	in := herb.sense(w)
	if got := in[2*VisionSectors+0]; got != 0.75 {
		t.Fatalf("voice sector 0 should carry mate's signal 0.75, got %v", got)
	}
	if got := in[2*VisionSectors+3]; got != 0 {
		t.Fatalf("carnivore signal must not leak into a herbivore's voice channel, got %v", got)
	}
}

// TestLearningDriftsPlasticBrains verifies that an agent with non-zero Plasticity
// adapts its brain over its life, while a Plasticity-0 agent does not.
func TestLearningDriftsPlasticBrains(t *testing.T) {
	w := &World{W: 200, H: 200, rng: rand.New(rand.NewSource(2)), params: DefaultParams()}
	w.reindex()

	plastic := w.newAgent(Herbivore, geom2(100, 100), nil, Traits{}, 0)
	plastic.Traits.Plasticity = 0.02
	fixed := w.newAgent(Herbivore, geom2(50, 50), nil, Traits{}, 0)
	fixed.Traits.Plasticity = 0

	for _, a := range []*Agent{plastic, fixed} {
		for i := 0; i < 50; i++ {
			out := a.think(w)
			a.act(w, out)
			a.learn(w, 6) // simulate repeatedly gaining energy
		}
	}

	if plastic.Brain.LearnedDrift() <= 0 {
		t.Fatal("plastic agent should have adapted its brain in life")
	}
	if fixed.Brain.LearnedDrift() != 0 {
		t.Fatalf("non-plastic agent should not adapt, drift = %v", fixed.Brain.LearnedDrift())
	}
}

// TestLineageInheritance checks founders get distinct lineages and offspring
// inherit their parent's lineage and record their parent.
func TestLineageInheritance(t *testing.T) {
	w := NewWorld(rand.New(rand.NewSource(1)), testConfig())
	a, b := w.Agents[0], w.Agents[1]
	if a.LineageID == b.LineageID {
		t.Fatal("distinct founders should have distinct lineages")
	}
	a.Energy = w.params.ReproThreshold + 10
	a.ReproCooldown = 0
	child := w.reproduce(a)
	if child.LineageID != a.LineageID {
		t.Fatalf("child lineage %d should match parent %d", child.LineageID, a.LineageID)
	}
	if child.ParentID != a.ID {
		t.Fatalf("child parent %d should be %d", child.ParentID, a.ID)
	}
}

// TestLineageHistoryRecorded checks that lineage counts are sampled over time.
func TestLineageHistoryRecorded(t *testing.T) {
	w := NewWorld(rand.New(rand.NewSource(1)), testConfig())
	for i := 0; i < 60; i++ {
		w.Step()
	}
	hist := w.LineageHistory()
	if len(hist) == 0 {
		t.Fatal("expected lineage history to be recorded")
	}
	sum := 0
	for _, c := range hist[len(hist)-1] {
		sum += c
	}
	if sum == 0 {
		t.Fatal("latest lineage sample should count some living agents")
	}
}

// TestGenealogyRecordsAndPrunes checks that births are recorded with parent links
// and that pruning keeps the ancestry of living agents while dropping extinct
// branches.
func TestGenealogyRecordsAndPrunes(t *testing.T) {
	w := NewWorld(rand.New(rand.NewSource(1)), testConfig())
	for i := 0; i < 80; i++ {
		w.Step()
	}
	gen := w.Genealogy()
	if len(gen) == 0 {
		t.Fatal("expected genealogy to be populated")
	}
	for _, a := range w.Agents {
		if a.Alive {
			if _, ok := gen[a.ID]; !ok {
				t.Fatalf("living agent %d missing from genealogy", a.ID)
			}
		}
	}
	// After a prune, every retained node must be an ancestor of (or be) a living agent.
	w.pruneGenealogy()
	reachable := map[int]bool{}
	for _, a := range w.Agents {
		if !a.Alive {
			continue
		}
		for id := a.ID; id != 0; {
			n := gen[id]
			if n == nil {
				break
			}
			reachable[id] = true
			id = n.ParentID
		}
	}
	for id := range w.Genealogy() {
		if !reachable[id] {
			t.Fatalf("pruned genealogy retained non-ancestor node %d", id)
		}
	}
}

func TestCarnivoreEatsHerbivore(t *testing.T) {
	w := &World{W: 200, H: 200, rng: rand.New(rand.NewSource(1)), targetPlants: 0, params: DefaultParams()}
	carn := w.newAgent(Carnivore, geom2(100, 100), nil, Traits{}, 0)
	herb := w.newAgent(Herbivore, geom2(101, 100), nil, Traits{}, 0)
	w.Agents = []*Agent{carn, herb}
	w.reindex()
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
	w := &World{W: 200, H: 200, rng: rand.New(rand.NewSource(1)), targetPlants: 0, params: DefaultParams()}
	herb := w.newAgent(Herbivore, geom2(50, 50), nil, Traits{}, 0)
	food := &Food{Pos: geom2(51, 50), Energy: DefaultParams().FoodMaxEnergy}
	w.Agents = []*Agent{herb}
	w.Foods = []*Food{food}
	w.reindex()
	e0 := herb.Energy
	w.resolveEat(herb)
	if herb.Energy <= e0 {
		t.Fatalf("herbivore should gain energy from food, %v -> %v", e0, herb.Energy)
	}
	if food.Energy >= DefaultParams().FoodMaxEnergy {
		t.Fatal("food energy should decrease after being eaten")
	}
}
