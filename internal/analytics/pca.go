package analytics

import (
	"math"
	"math/rand"
)

// Project2D projects high-dimensional points onto their top two principal
// components, returning a 2D coordinate per point. It mean-centres the data and
// finds the leading eigenvectors of the covariance by power iteration with
// deflation — no matrix library. ok is false when there are too few points or no
// variance to project.
//
// The rng seeds the power-iteration starting vectors; results are deterministic
// for a given rng and do not touch any other RNG.
func Project2D(pts [][]float64, rng *rand.Rand) (coords [][2]float64, ok bool) {
	n := len(pts)
	if n < 2 {
		return nil, false
	}
	dim := len(pts[0])

	// Mean-centre.
	mean := make([]float64, dim)
	for _, p := range pts {
		for d := range p {
			mean[d] += p[d]
		}
	}
	for d := range mean {
		mean[d] /= float64(n)
	}
	centred := make([][]float64, n)
	for i, p := range pts {
		c := make([]float64, dim)
		for d := range p {
			c[d] = p[d] - mean[d]
		}
		centred[i] = c
	}

	pc1 := powerIteration(centred, nil, rng)
	if pc1 == nil {
		return nil, false
	}
	pc2 := powerIteration(centred, pc1, rng) // deflate against pc1

	coords = make([][2]float64, n)
	for i, c := range centred {
		coords[i][0] = dot(c, pc1)
		if pc2 != nil {
			coords[i][1] = dot(c, pc2)
		}
	}
	return coords, true
}

// powerIteration returns the leading eigenvector of the covariance of the centred
// rows. If prev is non-nil, each candidate is orthogonalised against it (deflation)
// so the second principal component is found. Returns nil if there is no variance.
func powerIteration(centred [][]float64, prev []float64, rng *rand.Rand) []float64 {
	dim := len(centred[0])
	v := make([]float64, dim)
	for d := range v {
		v[d] = rng.NormFloat64()
	}
	if prev != nil {
		orthogonalise(v, prev)
	}
	normalise(v)

	for it := 0; it < 64; it++ {
		// w = covariance * v = sum_i (x_i · v) x_i
		w := make([]float64, dim)
		for _, x := range centred {
			s := dot(x, v)
			for d := range x {
				w[d] += s * x[d]
			}
		}
		if prev != nil {
			orthogonalise(w, prev)
		}
		if normalise(w) == 0 {
			return nil
		}
		// Converged when the direction stops moving.
		if math.Abs(dot(w, v)) > 0.999999 {
			return w
		}
		v = w
	}
	return v
}

func orthogonalise(v, basis []float64) {
	s := dot(v, basis)
	for i := range v {
		v[i] -= s * basis[i]
	}
}

func normalise(v []float64) float64 {
	var n float64
	for _, x := range v {
		n += x * x
	}
	n = math.Sqrt(n)
	if n == 0 {
		return 0
	}
	for i := range v {
		v[i] /= n
	}
	return n
}

func dot(a, b []float64) float64 {
	var s float64
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}
