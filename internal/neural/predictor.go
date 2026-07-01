package neural

type Predictor struct {
	In, Out int

	W []float64
	B []float64
}

const predictorWeightCap = 10.0

func NewPredictor(in, out int) *Predictor {
	return &Predictor{In: in, Out: out, W: make([]float64, out*in), B: make([]float64, out)}
}

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
