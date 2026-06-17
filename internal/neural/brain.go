// Package neural implements a tiny hand-rolled recurrent neural network used as
// an agent's "brain". There is no training by backpropagation: networks are
// initialised randomly and improved purely through evolution (mutation of weights
// in offspring). The implementation deliberately uses flat float64 slices and a
// couple of loops rather than any external ML/matrix library.
package neural

import (
	"math"
	"math/rand"
)

// Brain is a single-hidden-layer Elman recurrent network with tanh activations on
// both the hidden and output layers, so every output is bounded in (-1, 1).
//
// "Recurrent" means the previous tick's hidden activations are fed back into the
// hidden layer alongside the current inputs (the context, or memory). This gives
// agents a one-step memory, so behaviour can depend on the recent past rather than
// being a pure reflex of the current senses — enabling persistence (keep fleeing
// for a moment after a predator leaves the sense radius), wandering, and similar.
//
// Weights are stored as flat slices in row-major order:
//
//	WIH has Hidden*In entries:     WIH[h*In + i]     connects input i  -> hidden h.
//	WCH has Hidden*Hidden entries: WCH[h*Hidden + c] connects context c -> hidden h.
//	WHO has Out*Hidden entries:    WHO[o*Hidden + h] connects hidden h -> output o.
//
// state holds the previous hidden activations (the recurrent context). It is run
// state, not genome: Clone resets it, and Mutate never touches it.
type Brain struct {
	In, Hidden, Out int

	WIH []float64 // input   -> hidden weights
	WCH []float64 // context -> hidden weights (recurrent feedback)
	BH  []float64 // hidden biases
	WHO []float64 // hidden  -> output weights
	BO  []float64 // output biases

	state []float64 // previous hidden activations (length Hidden)
}

// New returns a Brain with the given layer sizes and small random weights drawn
// from rng. Weights are seeded in roughly [-1, 1] which is plenty of range given
// the tanh activations; evolution refines them from there. The recurrent state
// starts at zero (a blank memory).
func New(rng *rand.Rand, in, hidden, out int) *Brain {
	b := &Brain{
		In: in, Hidden: hidden, Out: out,
		WIH:   make([]float64, hidden*in),
		WCH:   make([]float64, hidden*hidden),
		BH:    make([]float64, hidden),
		WHO:   make([]float64, out*hidden),
		BO:    make([]float64, out),
		state: make([]float64, hidden),
	}
	for i := range b.WIH {
		b.WIH[i] = rng.NormFloat64()
	}
	for i := range b.WCH {
		b.WCH[i] = rng.NormFloat64()
	}
	for i := range b.WHO {
		b.WHO[i] = rng.NormFloat64()
	}
	// Biases start at zero; mutation will move them as needed.
	return b
}

// Forward runs the network on inputs for one tick and returns a freshly allocated
// slice of Out activations. It also advances the recurrent state, so successive
// calls form a sequence — call Reset to clear the memory. If len(inputs) != In the
// input is treated as zero-padded or truncated, which keeps callers robust to
// small sensor changes.
func (b *Brain) Forward(inputs []float64) []float64 {
	hidden := make([]float64, b.Hidden)
	for h := 0; h < b.Hidden; h++ {
		sum := b.BH[h]
		ibase := h * b.In
		for i := 0; i < b.In; i++ {
			var x float64
			if i < len(inputs) {
				x = inputs[i]
			}
			sum += b.WIH[ibase+i] * x
		}
		cbase := h * b.Hidden
		for c := 0; c < b.Hidden; c++ {
			sum += b.WCH[cbase+c] * b.state[c]
		}
		hidden[h] = math.Tanh(sum)
	}
	// The new hidden activations become next tick's context.
	copy(b.state, hidden)

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

// Reset clears the recurrent memory back to a blank slate.
func (b *Brain) Reset() {
	for i := range b.state {
		b.state[i] = 0
	}
}

// State returns a copy of the current recurrent memory (the previous hidden
// activations), for inspection/visualisation.
func (b *Brain) State() []float64 {
	return append([]float64(nil), b.state...)
}

// Clone returns a deep copy of the brain's genome. The recurrent state is NOT
// inherited: offspring start with a blank memory.
func (b *Brain) Clone() *Brain {
	return &Brain{
		In: b.In, Hidden: b.Hidden, Out: b.Out,
		WIH:   append([]float64(nil), b.WIH...),
		WCH:   append([]float64(nil), b.WCH...),
		BH:    append([]float64(nil), b.BH...),
		WHO:   append([]float64(nil), b.WHO...),
		BO:    append([]float64(nil), b.BO...),
		state: make([]float64, b.Hidden),
	}
}

// Mutate perturbs the brain's weights in place. Each weight and bias is, with
// probability rate, nudged by Gaussian noise scaled by std. This is the sole
// mechanism by which behaviour changes between generations. The recurrent state is
// left untouched.
func (b *Brain) Mutate(rng *rand.Rand, rate, std float64) {
	mut := func(s []float64) {
		for i := range s {
			if rng.Float64() < rate {
				s[i] += rng.NormFloat64() * std
			}
		}
	}
	mut(b.WIH)
	mut(b.WCH)
	mut(b.BH)
	mut(b.WHO)
	mut(b.BO)
}
