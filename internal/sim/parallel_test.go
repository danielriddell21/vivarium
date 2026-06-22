package sim

import (
	"math/rand"
	"testing"
)

// bigConfig has enough agents to exercise the parallel think path (active count
// above parallelThinkThreshold).
func bigConfig() Config {
	c := DefaultConfig()
	c.Width, c.Height = 1200, 900
	c.Plants, c.Herbivores, c.Carnivores = 300, 320, 30
	c.TargetPlants = 400
	return c
}

// TestParallelDeterminism runs a large world (parallel think phase active) twice
// from the same seed and requires identical trajectories — concurrency must not
// affect results. Run with -race to also catch data races in the think phase.
func TestParallelDeterminism(t *testing.T) {
	run := func() []Counts {
		w := NewWorld(rand.New(rand.NewSource(123)), bigConfig())
		var trace []Counts
		for i := 0; i < 120; i++ {
			w.Step()
			trace = append(trace, w.CountKinds())
		}
		return trace
	}
	a, b := run(), run()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("parallel run diverged at step %d: %+v vs %+v", i, a[i], b[i])
		}
	}
	// Sanity: the world should actually have been large enough to go parallel.
	if a[0].Herbivores+a[0].Carnivores < parallelThinkThreshold {
		t.Skip("population stayed below the parallel threshold; test inconclusive")
	}
}
