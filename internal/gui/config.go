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
}
