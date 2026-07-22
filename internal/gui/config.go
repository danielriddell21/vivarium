package gui

type Config struct {
	Seed                           int64
	Width, Height                  float64
	Plants, Herbivores, Carnivores int
	Rescue                         bool
	ConfigPath                     string
	PrintConfig                    bool
	LoadPath, SnapPath             string
	Changed                        func(name string) bool

	// Recording (matches galapagos): when RecordPath is set the run captures
	// frames to a GIF and exits.
	RecordPath   string
	RecordFPS    int
	RecordScale  int
	RecordFrames int
}
