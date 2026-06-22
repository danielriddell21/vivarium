package sim

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig writes a partial config file and checks it overlays the defaults.
func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.json")
	if err := os.WriteFile(path, []byte(`{"herbivores":3,"params":{"mutationStd":0.9}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Herbivores != 3 {
		t.Fatalf("herbivores override not applied: %d", cfg.Herbivores)
	}
	if cfg.Params.MutationStd != 0.9 {
		t.Fatalf("mutationStd override not applied: %v", cfg.Params.MutationStd)
	}
	if cfg.Params.ReproThreshold != DefaultParams().ReproThreshold {
		t.Fatal("unspecified param should keep its default")
	}
}

// TestParamsAffectReproduction confirms a tuned Param actually changes behaviour:
// an unreachable reproduction threshold means no agent ever gives birth.
func TestParamsAffectReproduction(t *testing.T) {
	cfg := testConfig()
	cfg.Params.ReproThreshold = 1e9 // nobody can ever reach this
	w := NewWorld(rand.New(rand.NewSource(1)), cfg)
	for i := 0; i < 300; i++ {
		w.Step()
	}
	for _, a := range w.Agents {
		if a.Generation > 0 {
			t.Fatalf("expected no births with an unreachable ReproThreshold, found generation %d", a.Generation)
		}
	}
}

// TestConfigJSONOverlay verifies the partial-override semantics the -config flag
// relies on: fields present in the JSON win, everything else keeps its default.
func TestConfigJSONOverlay(t *testing.T) {
	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(`{"plants":5,"params":{"moveCost":0.5}}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Plants != 5 {
		t.Fatalf("plants override not applied: %d", cfg.Plants)
	}
	if cfg.Params.MoveCost != 0.5 {
		t.Fatalf("moveCost override not applied: %v", cfg.Params.MoveCost)
	}
	// An unspecified param must retain its default rather than zeroing out.
	if cfg.Params.MaxEnergy != DefaultParams().MaxEnergy {
		t.Fatalf("unspecified param should keep default, got MaxEnergy %v", cfg.Params.MaxEnergy)
	}
	if cfg.Carnivores != DefaultConfig().Carnivores {
		t.Fatalf("unspecified field should keep default, got Carnivores %d", cfg.Carnivores)
	}
}
