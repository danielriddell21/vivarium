package analytics

import (
	"math"
	"math/rand"
	"testing"
)

// TestKMeansSeparatesClusters builds three well-separated blobs and checks that
// k-means assigns each blob to a single cluster.
func TestKMeansSeparatesClusters(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	centers := [][]float64{{0, 0}, {10, 10}, {0, 10}}
	var pts [][]float64
	var truth []int
	for c, ctr := range centers {
		for i := 0; i < 40; i++ {
			pts = append(pts, []float64{
				ctr[0] + rng.NormFloat64()*0.3,
				ctr[1] + rng.NormFloat64()*0.3,
			})
			truth = append(truth, c)
		}
	}

	assign, _ := KMeans(pts, 3, 50, rand.New(rand.NewSource(7)), nil)

	// Every point in a true blob should share the same predicted cluster label.
	for c := 0; c < 3; c++ {
		label := -1
		for i := range pts {
			if truth[i] != c {
				continue
			}
			if label == -1 {
				label = assign[i]
			} else if assign[i] != label {
				t.Fatalf("blob %d split across clusters (%d vs %d)", c, label, assign[i])
			}
		}
	}
}

func TestKMeansFewerPointsThanK(t *testing.T) {
	pts := [][]float64{{0}, {5}}
	assign, centroids := KMeans(pts, 5, 10, rand.New(rand.NewSource(1)), nil)
	if len(assign) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assign))
	}
	if len(centroids) != 2 {
		t.Fatalf("k should be capped at point count, got %d centroids", len(centroids))
	}
}

// TestKMeansWarmStartKeepsLabels verifies that warm-starting from a previous run's
// centroids keeps cluster labels attached to the same regions when the data shifts
// slightly — the property that stops the species view's colours from flashing.
func TestKMeansWarmStartKeepsLabels(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	centers := [][]float64{{0, 0}, {10, 0}, {0, 10}, {10, 10}}
	makeData := func(jitter float64) [][]float64 {
		var pts [][]float64
		for _, ctr := range centers {
			for i := 0; i < 30; i++ {
				pts = append(pts, []float64{
					ctr[0] + rng.NormFloat64()*0.2 + jitter,
					ctr[1] + rng.NormFloat64()*0.2,
				})
			}
		}
		return pts
	}

	a := makeData(0)
	assignA, cents := KMeans(a, 4, 50, rand.New(rand.NewSource(2)), nil)

	// Same points, slightly shifted, warm-started from the previous centroids.
	b := make([][]float64, len(a))
	for i := range a {
		b[i] = []float64{a[i][0] + 0.05, a[i][1] - 0.05}
	}
	assignB, _ := KMeans(b, 4, 50, rand.New(rand.NewSource(99)), cents)

	// Without warm-start the labels would permute arbitrarily; with it, the same
	// point should keep (almost always) the same cluster label.
	same := 0
	for i := range assignA {
		if assignA[i] == assignB[i] {
			same++
		}
	}
	if frac := float64(same) / float64(len(assignA)); frac < 0.95 {
		t.Fatalf("warm start should keep labels stable, only %.0f%% matched", frac*100)
	}
}

// TestPCAFindsPrincipalAxis projects points stretched along a known diagonal and
// checks the first component captures most of the variance (range along axis 0 is
// much larger than along axis 1).
func TestPCAFindsPrincipalAxis(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	var pts [][]float64
	for i := 0; i < 200; i++ {
		t := rng.NormFloat64() * 5 // large spread along the (1,1) diagonal
		n := rng.NormFloat64() * 0.1
		pts = append(pts, []float64{t + n, t - n})
	}

	coords, ok := Project2D(pts, rand.New(rand.NewSource(3)))
	if !ok {
		t.Fatal("expected projection to succeed")
	}
	min0, max0, min1, max1 := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, c := range coords {
		min0, max0 = math.Min(min0, c[0]), math.Max(max0, c[0])
		min1, max1 = math.Min(min1, c[1]), math.Max(max1, c[1])
	}
	spread0, spread1 := max0-min0, max1-min1
	if spread0 < spread1*5 {
		t.Fatalf("PC1 should capture the dominant axis: spread0=%.2f spread1=%.2f", spread0, spread1)
	}
}

func TestPCATooFewPoints(t *testing.T) {
	if _, ok := Project2D([][]float64{{1, 2}}, rand.New(rand.NewSource(1))); ok {
		t.Fatal("expected projection to fail with a single point")
	}
}
