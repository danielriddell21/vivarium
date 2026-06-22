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
