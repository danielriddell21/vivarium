package sim

import (
	"math/rand"
	"path/filepath"
	"testing"
)

// TestSnapshotRoundTrip evolves a world, saves it to disk, loads it back, and
// checks the population and genomes survive the trip and the loaded world runs.
func TestSnapshotRoundTrip(t *testing.T) {
	w := NewWorld(rand.New(rand.NewSource(7)), testConfig())
	for i := 0; i < 200; i++ {
		w.Step()
	}
	snap := w.Snapshot()
	if len(snap.Agents) == 0 {
		t.Fatal("snapshot captured no agents")
	}

	path := filepath.Join(t.TempDir(), "snap.json")
	if err := SaveSnapshot(path, snap); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSnapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}

	w2 := NewWorldFromSnapshot(rand.New(rand.NewSource(1)), loaded)
	live := 0
	for _, a := range w2.Agents {
		if a.Alive {
			live++
		}
	}
	if live != len(snap.Agents) {
		t.Fatalf("loaded %d living agents, snapshot had %d", live, len(snap.Agents))
	}

	// A reconstructed brain must reproduce the saved genome exactly.
	if len(w2.Agents) > 0 {
		a0 := snap.Agents[0]
		g := w2.Agents[0].Brain.Genome()
		for i := range g {
			if g[i] != a0.Genome[i] {
				t.Fatalf("genome differs after round trip at %d: %v vs %v", i, g[i], a0.Genome[i])
			}
		}
		if w2.Agents[0].Traits != a0.Traits {
			t.Fatalf("traits differ after round trip: %+v vs %+v", w2.Agents[0].Traits, a0.Traits)
		}
	}

	// The loaded world must continue running.
	for i := 0; i < 50; i++ {
		w2.Step()
	}
}
