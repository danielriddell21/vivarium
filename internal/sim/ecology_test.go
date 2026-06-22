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
	if a.Pos != start {
		// It may take a couple of ticks to reach; step until it would enter.
	}
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
