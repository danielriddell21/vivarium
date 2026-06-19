// Package analytics provides small, dependency-free machine-learning routines for
// analysing the evolving population: k-means clustering (to discover emergent
// "species" in genome space) and a 2D PCA projection (to visualise that space).
// Everything is hand-rolled on plain [][]float64 — no ML libraries — and uses a
// caller-supplied *rand.Rand so it never perturbs the simulation's own RNG.
package analytics

import (
	"math"
	"math/rand"
)

// KMeans partitions pts into k clusters with Lloyd's algorithm and k-means++
// seeding. It returns, for each point, the index of its assigned cluster, plus the
// final centroids. All points must share the same dimensionality. If there are
// fewer points than k, each point becomes its own cluster.
func KMeans(pts [][]float64, k, iters int, rng *rand.Rand) (assign []int, centroids [][]float64) {
	n := len(pts)
	assign = make([]int, n)
	if n == 0 || k <= 0 {
		return assign, nil
	}
	if k > n {
		k = n
	}

	centroids = seedPlusPlus(pts, k, rng)
	for it := 0; it < iters; it++ {
		changed := false
		// Assignment step: nearest centroid.
		for i, p := range pts {
			best, bestD := 0, math.Inf(1)
			for c, cen := range centroids {
				if d := sqDist(p, cen); d < bestD {
					best, bestD = c, d
				}
			}
			if assign[i] != best {
				assign[i] = best
				changed = true
			}
		}
		// Update step: mean of each cluster.
		sums := make([][]float64, k)
		counts := make([]int, k)
		dim := len(pts[0])
		for c := range sums {
			sums[c] = make([]float64, dim)
		}
		for i, p := range pts {
			c := assign[i]
			counts[c]++
			for d := range p {
				sums[c][d] += p[d]
			}
		}
		for c := range centroids {
			if counts[c] == 0 {
				// Re-seed an empty cluster onto a random point to stay useful.
				centroids[c] = append([]float64(nil), pts[rng.Intn(n)]...)
				continue
			}
			for d := range sums[c] {
				centroids[c][d] = sums[c][d] / float64(counts[c])
			}
		}
		if !changed && it > 0 {
			break
		}
	}
	return assign, centroids
}

// seedPlusPlus chooses k initial centroids using the k-means++ scheme: spread
// seeds out by sampling each next one with probability proportional to its squared
// distance from the nearest chosen seed.
func seedPlusPlus(pts [][]float64, k int, rng *rand.Rand) [][]float64 {
	n := len(pts)
	centroids := make([][]float64, 0, k)
	centroids = append(centroids, append([]float64(nil), pts[rng.Intn(n)]...))

	d2 := make([]float64, n)
	for len(centroids) < k {
		var total float64
		for i, p := range pts {
			best := math.Inf(1)
			for _, c := range centroids {
				if d := sqDist(p, c); d < best {
					best = d
				}
			}
			d2[i] = best
			total += best
		}
		if total == 0 { // all remaining points coincide with a centroid
			centroids = append(centroids, append([]float64(nil), pts[rng.Intn(n)]...))
			continue
		}
		target := rng.Float64() * total
		idx := 0
		for cum := 0.0; idx < n; idx++ {
			cum += d2[idx]
			if cum >= target {
				break
			}
		}
		if idx >= n {
			idx = n - 1
		}
		centroids = append(centroids, append([]float64(nil), pts[idx]...))
	}
	return centroids
}

func sqDist(a, b []float64) float64 {
	var s float64
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return s
}
