package gui

// Config holds the resolved command-line options for the GUI. Changed reports
// whether a given flag was set explicitly (so config-file values are only
// overridden by flags the user actually passed). It carries no Ebiten types, so
// the CLI builds it in either build and passes it to Run.
type Config struct {
	Seed                           int64
	Width, Height                  float64
	Plants, Herbivores, Carnivores int
	Rescue                         bool
	ConfigPath                     string
	PrintConfig                    bool
	LoadPath, SnapPath             string
	Changed                        func(name string) bool
}
