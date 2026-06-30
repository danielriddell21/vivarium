package neural

import "testing"

func TestPredictorLearns(t *testing.T) {
	p := NewPredictor(3, 2)
	feat := []float64{0.5, -0.2, 1.0}
	target := []float64{0.8, -0.4}

	first := p.Train(feat, target, 0.05)
	var last float64
	for i := 0; i < 200; i++ {
		last = p.Train(feat, target, 0.05)
	}
	if last >= first {
		t.Fatalf("expected prediction error to fall: first %.4f, last %.4f", first, last)
	}
	if last > 1e-3 {
		t.Fatalf("expected error to approach zero, got %.4f", last)
	}
}

func TestPredictorSurpriseDropsOnFamiliarity(t *testing.T) {
	p := NewPredictor(2, 2)
	feat := []float64{1.0, 0.5}
	target := []float64{0.6, -0.3}

	novel := p.Train(feat, target, 0.1)
	for i := 0; i < 100; i++ {
		p.Train(feat, target, 0.1)
	}
	familiar := p.Train(feat, target, 0.1)
	if familiar >= novel {
		t.Fatalf("surprise should drop with familiarity: novel %.4f, familiar %.4f", novel, familiar)
	}
}
