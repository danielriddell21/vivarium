package sim

import (
	"math/rand"
	"testing"
)

// TestSeasonsModulateRegrowth checks the seasonal cycle scales plant regrowth:
// a plant regrows faster at the summer peak than at the winter trough.
func TestSeasonsModulateRegrowth(t *testing.T) {
	cfg := testConfig()
	cfg.Params.SeasonLength = 1000
	cfg.Params.SeasonAmplitude = 0.6
	w := NewWorld(rand.New(rand.NewSource(1)), cfg)

	grow := func(atTick int) float64 {
		w.Tick = atTick
		f := &Food{Energy: 0}
		w.regrowFood(f)
		return f.Energy
	}
	summer := grow(250) // quarter cycle: sin = +1, peak growth
	winter := grow(750) // three-quarter cycle: sin = -1, trough
	if summer <= winter {
		t.Fatalf("summer regrowth (%v) should exceed winter (%v)", summer, winter)
	}

	// Disabling seasons gives a flat factor of 1.
	w.params.SeasonLength = 0
	if f := w.SeasonFactor(); f != 1 {
		t.Fatalf("season factor with SeasonLength 0 should be 1, got %v", f)
	}
}

// TestObstacleBlocksMovement checks an agent cannot walk into a terrain rock but
// can still move when its path is clear.
func TestObstacleBlocksMovement(t *testing.T) {
	w := &World{W: 400, H: 300, rng: rand.New(rand.NewSource(1)), params: DefaultParams()}
	w.obstacles = []Obstacle{{Pos: geom2(130, 100), Radius: 20}}
	a := w.newAgent(Herbivore, geom2(100, 100), nil, Traits{}, 0)
	a.Traits.Size = 4
	a.Traits.MaxSpeed = 5
	a.Heading = 0 // east, straight at the rock
	w.Agents = []*Agent{a}
	w.reindex()

	start := a.Pos
	a.act(w, []float64{0, 1, 0, 0}) // full speed ahead
	// It may take a couple of ticks to reach the rock; step until it would enter.
	for i := 0; i < 20; i++ {
		a.act(w, []float64{0, 1, 0, 0})
	}
	if w.penetration(a.Pos, a.Traits.Size) > 0 {
		t.Fatalf("agent penetrated the obstacle: pos %+v", a.Pos)
	}
	if a.Pos.X <= start.X {
		t.Fatal("agent should have advanced toward (and stopped against) the rock")
	}
}

// TestDietAffectsEnergyGain checks a specialist herbivore gains more from its
// preferred food type than from the opposite type.
func TestDietAffectsEnergyGain(t *testing.T) {
	gain := func(diet float64, foodType int) float64 {
		w := &World{W: 200, H: 200, rng: rand.New(rand.NewSource(1)), params: DefaultParams()}
		h := w.newAgent(Herbivore, geom2(50, 50), nil, Traits{}, 0)
		h.Traits.Diet = diet
		h.Traits.Size = 4
		w.Agents = []*Agent{h}
		w.Foods = []*Food{{Pos: geom2(51, 50), Energy: DefaultParams().FoodMaxEnergy, Type: foodType}}
		w.reindex()
		e0 := h.Energy
		w.resolveEat(h)
		return h.Energy - e0
	}
	matched := gain(0, 0)    // type-0 specialist eating type 0
	mismatched := gain(0, 1) // same specialist eating type 1
	if matched <= mismatched {
		t.Fatalf("matched diet should yield more energy: matched %v vs mismatched %v", matched, mismatched)
	}
	if mismatched != 0 {
		t.Fatalf("opposite food should yield ~0 energy for a pure specialist, got %v", mismatched)
	}
}

// TestVisionOcclusion checks terrain between an agent and a target hides it, while
// the same target is visible once the obstacle is removed.
func TestVisionOcclusion(t *testing.T) {
	build := func(withRock bool) []float64 {
		w := &World{W: 1000, H: 1000, rng: rand.New(rand.NewSource(1)), params: DefaultParams()}
		w.params.DayLength = 0 // full daylight, so only occlusion matters
		if withRock {
			w.obstacles = []Obstacle{{Pos: geom2(150, 100), Radius: 20}} // dead ahead, between
		}
		a := w.newAgent(Carnivore, geom2(100, 100), nil, Traits{}, 0)
		a.Heading = 0
		a.Traits.SenseRadius = 200
		prey := w.newAgent(Herbivore, geom2(200, 100), nil, Traits{}, 0) // due east, behind the rock
		w.Agents = []*Agent{a, prey}
		w.reindex()
		return a.sense(w)
	}
	if blocked := build(true); blocked[0] != 0 {
		t.Fatalf("prey behind a rock should be hidden, got sector 0 = %v", blocked[0])
	}
	if clear := build(false); clear[0] <= 0 {
		t.Fatal("prey in the open should be visible in sector 0")
	}
}

// TestDayNightShrinksVision checks an agent perceives a distant neighbour by day
// but not at night, when effective vision contracts.
func TestDayNightShrinksVision(t *testing.T) {
	build := func(tick int) []float64 {
		w := &World{W: 1000, H: 1000, rng: rand.New(rand.NewSource(1)), params: DefaultParams()}
		w.Tick = tick
		a := w.newAgent(Carnivore, geom2(100, 100), nil, Traits{}, 0)
		a.Heading = 0
		a.Traits.SenseRadius = 200
		prey := w.newAgent(Herbivore, geom2(290, 100), nil, Traits{}, 0) // 190 away, east
		w.Agents = []*Agent{a, prey}
		w.reindex()
		return a.sense(w)
	}
	noon := build(300)     // quarter cycle: full daylight
	midnight := build(900) // three-quarter cycle: darkest
	if noon[0] <= 0 {
		t.Fatal("prey within day vision should light target sector 0")
	}
	if midnight[0] != 0 {
		t.Fatalf("prey beyond night vision should be invisible, got %v", midnight[0])
	}
	// Light is brighter at noon than midnight.
	wn := &World{params: DefaultParams(), Tick: 300}
	wm := &World{params: DefaultParams(), Tick: 900}
	if wn.LightFactor() <= wm.LightFactor() {
		t.Fatal("noon should be brighter than midnight")
	}
}

// TestFindMate checks mate selection: a nearby mature same-kind agent qualifies,
// while a different species, an immature agent, or one out of range does not.
func TestFindMate(t *testing.T) {
	w := &World{W: 500, H: 500, rng: rand.New(rand.NewSource(1)), params: DefaultParams()}
	w.params.MateRadius = 50
	seeker := w.newAgent(Herbivore, geom2(100, 100), nil, Traits{}, 0)
	mate := w.newAgent(Herbivore, geom2(120, 100), nil, Traits{}, 0)  // valid: same kind, in range
	carn := w.newAgent(Carnivore, geom2(110, 100), nil, Traits{}, 0)  // wrong kind
	young := w.newAgent(Herbivore, geom2(105, 100), nil, Traits{}, 0) // immature
	young.ReproCooldown = 30
	far := w.newAgent(Herbivore, geom2(300, 100), nil, Traits{}, 0) // out of range
	w.Agents = []*Agent{seeker, mate, carn, young, far}
	w.reindex()

	if got := w.findMate(seeker); got != mate {
		t.Fatalf("expected the valid same-kind mature in-range mate, got %+v", got)
	}

	// With no valid partner, findMate returns nil (reproduction falls back to clone).
	w.Agents = []*Agent{seeker, carn, young, far}
	w.reindex()
	if got := w.findMate(seeker); got != nil {
		t.Fatalf("expected no mate, got #%d", got.ID)
	}
}
