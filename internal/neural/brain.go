// Package neural implements a tiny hand-rolled feedforward neural network used as
// an agent's "brain". There is no training by backpropagation: networks are
// initialised randomly and improved purely through evolution (mutation of weights
// in offspring). The implementation deliberately uses flat float64 slices and a
// couple of loops rather than any external ML/matrix library.
package neural

import (
	"math"
	"math/rand"
)

// Brain is a single-hidden-layer feedforward network with tanh activations on
// both the hidden and output layers, so every output is bounded in (-1, 1).
//
// Weights are stored as flat slices in row-major order:
//
//	WIH has Hidden*In entries: WIH[h*In + i] connects input i to hidden unit h.
//	WHO has Out*Hidden entries: WHO[o*Hidden + h] connects hidden h to output o.
type Brain struct {
	In, Hidden, Out int

	WIH []float64 // input -> hidden weights
	BH  []float64 // hidden biases
	WHO []float64 // hidden -> output weights
	BO  []float64 // output biases
}

// New returns a Brain with the given layer sizes and small random weights drawn
// from rng. Weights are seeded in roughly [-1, 1] which is plenty of range given
// the tanh activations; evolution refines them from there.
func New(rng *rand.Rand, in, hidden, out int) *Brain {
	b := &Brain{
		In: in, Hidden: hidden, Out: out,
		WIH: make([]float64, hidden*in),
		BH:  make([]float64, hidden),
		WHO: make([]float64, out*hidden),
		BO:  make([]float64, out),
	}
	for i := range b.WIH {
		b.WIH[i] = rng.NormFloat64()
	}
	for i := range b.WHO {
		b.WHO[i] = rng.NormFloat64()
	}
	// Biases start at zero; mutation will move them as needed.
	return b
}

// Forward runs the network on inputs and returns a freshly allocated slice of
// Out activations. If len(inputs) != In the input is treated as zero-padded or
// truncated, which keeps callers robust to small sensor changes.
func (b *Brain) Forward(inputs []float64) []float64 {
	hidden := make([]float64, b.Hidden)
	for h := 0; h < b.Hidden; h++ {
		sum := b.BH[h]
		base := h * b.In
		for i := 0; i < b.In; i++ {
			var x float64
			if i < len(inputs) {
				x = inputs[i]
			}
			sum += b.WIH[base+i] * x
		}
		hidden[h] = math.Tanh(sum)
	}

	out := make([]float64, b.Out)
	for o := 0; o < b.Out; o++ {
		sum := b.BO[o]
		base := o * b.Hidden
		for h := 0; h < b.Hidden; h++ {
			sum += b.WHO[base+h] * hidden[h]
		}
		out[o] = math.Tanh(sum)
	}
	return out
}

// Clone returns a deep copy of the brain, sharing no backing arrays with b.
func (b *Brain) Clone() *Brain {
	cp := &Brain{
		In: b.In, Hidden: b.Hidden, Out: b.Out,
		WIH: append([]float64(nil), b.WIH...),
		BH:  append([]float64(nil), b.BH...),
		WHO: append([]float64(nil), b.WHO...),
		BO:  append([]float64(nil), b.BO...),
	}
	return cp
}

// Mutate perturbs the brain in place. Each weight and bias is, with probability
// rate, nudged by Gaussian noise scaled by std. This is the sole mechanism by
// which behaviour changes between generations.
func (b *Brain) Mutate(rng *rand.Rand, rate, std float64) {
	mut := func(s []float64) {
		for i := range s {
			if rng.Float64() < rate {
				s[i] += rng.NormFloat64() * std
			}
		}
	}
	mut(b.WIH)
	mut(b.BH)
	mut(b.WHO)
	mut(b.BO)
}
