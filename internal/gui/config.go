package gui

import "github.com/danielriddell21/crucible/record"

type Config struct {
	Seed                           int64
	Width, Height                  float64
	Plants, Herbivores, Carnivores int
	Rescue                         bool
	ConfigPath                     string
	PrintConfig                    bool
	LoadPath, SnapPath             string
	Changed                        func(name string) bool

	// Rec holds the shared --record flags; when its path is set the run
	// captures frames to a GIF and exits.
	Rec record.Options
}
