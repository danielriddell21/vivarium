package sim

import (
	"math/rand"
	"testing"

	"github.com/danielriddell21/vivarium/internal/geom"
)

// bruteNearestAgent finds the nearest living agent of kind k within radius by
// scanning every agent — the reference the grid query must match.
func bruteNearestAgent(w *World, pos geom.Vec2, k Kind, radius float64, exclude int) *Agent {
	var best *Agent
	bestD := radius
	for _, a := range w.Agents {
		if !a.Alive || a.Kind != k || a.ID == exclude {
			continue
		}
		if d := pos.ToroidalDist(a.Pos, w.W, w.H); d <= bestD {
			bestD, best = d, a
		}
	}
	return best
}

// TestGridMatchesBruteForce checks that grid-accelerated nearest queries return
// the same nearest agent (by distance) as a full linear scan, including near the
// toroidal edges where the cell window wraps.
func TestGridMatchesBruteForce(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	w := &World{W: 640, H: 480, rng: rng, params: DefaultParams()}
	for i := 0; i < 400; i++ {
		w.Agents = append(w.Agents, w.newAgent(Herbivore, w.randPos(), nil, Traits{}, 0))
	}
	w.reindex()

	for i := 0; i < 200; i++ {
		pos := w.randPos()
		radius := 20 + rng.Float64()*200
		got := w.nearestAgentOfKind(pos, Herbivore, radius, -1)
		want := bruteNearestAgent(w, pos, Herbivore, radius, -1)

		// Distances must match (the exact agent can differ only on an exact tie).
		switch {
		case got == nil && want == nil:
			// both found nothing — fine
		case got == nil || want == nil:
			t.Fatalf("query %d: grid=%v brute=%v (one found nothing)", i, got, want)
		default:
			dg := pos.ToroidalDist(got.Pos, w.W, w.H)
			db := pos.ToroidalDist(want.Pos, w.W, w.H)
			if dg != db {
				t.Fatalf("query %d: grid nearest dist %.4f != brute %.4f", i, dg, db)
			}
		}
	}
}
