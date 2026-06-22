package sim

import (
	"encoding/json"
	"os"
)

// LoadConfig reads a JSON config file and overlays it on the defaults: any field
// present in the file overrides the default, everything else is left untouched
// (so partial config files are fine). It is used by both the GUI and headless
// entry points.
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
