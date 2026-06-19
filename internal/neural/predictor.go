package neural

// Predictor is a tiny online-trained linear forward model: given a feature vector
// (an agent's current senses concatenated with its chosen action) it predicts the
// next tick's senses. It is trained by ordinary stochastic gradient descent on
// mean-squared error — a genuinely gradient-based learner, hand-rolled, no library.
//
// Its purpose is curiosity: the prediction error on each transition measures how
// "surprising" the new situation was. That error feeds the agent's reward as an
// intrinsic motivation to seek novel, not-yet-predictable states. As the model
// learns a region of the world the error there falls, so curiosity naturally fades
// from the familiar and moves on — the standard intrinsic-motivation dynamic.
//
// Unlike the control brain, a Predictor is not inherited or evolved: every agent
// starts with a blank model (predicting zero) and learns it from scratch in life.
type Predictor struct {
	In, Out int

	W []float64 // Out*In weights, row-major: W[o*In + i]
	B []float64 // Out biases
}

// predictorWeightCap bounds the linear weights so online SGD can't diverge.
const predictorWeightCap = 10.0

// NewPredictor returns a zero-initialised model (it initially predicts all zeros
// and learns from there).
func NewPredictor(in, out int) *Predictor {
	return &Predictor{In: in, Out: out, W: make([]float64, out*in), B: make([]float64, out)}
}

// predict writes the model's output for feat into dst (length Out).
func (p *Predictor) predict(feat, dst []float64) {
	for o := 0; o < p.Out; o++ {
		sum := p.B[o]
		base := o * p.In
		for i := 0; i < p.In; i++ {
			sum += p.W[base+i] * feat[i]
		}
		dst[o] = sum
	}
}

// Train predicts target from feat, takes one SGD step on the mean-squared error to
// reduce it, and returns the mean-squared error measured BEFORE the update — i.e.
// the surprise of this transition. lr is the learning rate.
func (p *Predictor) Train(feat, target []float64, lr float64) float64 {
	pred := make([]float64, p.Out)
	p.predict(feat, pred)

	var sse float64
	for o := 0; o < p.Out; o++ {
		err := pred[o] - target[o] // dMSE/dpred (up to a constant factor)
		sse += err * err
		base := o * p.In
		step := lr * err
		for i := 0; i < p.In; i++ {
			p.W[base+i] = clampPred(p.W[base+i] - step*feat[i])
		}
		p.B[o] = clampPred(p.B[o] - step)
	}
	return sse / float64(p.Out)
}

func clampPred(v float64) float64 {
	if v > predictorWeightCap {
		return predictorWeightCap
	}
	if v < -predictorWeightCap {
		return -predictorWeightCap
	}
	return v
}
