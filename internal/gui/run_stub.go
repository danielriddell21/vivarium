//go:build !ebiten

package gui

import "errors"

func Available() bool { return false }

func Run(o Config) error {
	world, err := buildWorld(o)
	if err != nil || world == nil {
		return err
	}
	// Recording needs no window: the scene is composed in software, so demo
	// media builds anywhere, with no display.
	if o.Rec.Recording() {
		return Render(o, world)
	}
	return errors.New("built without the GUI; rebuild with -tags ebiten, or use the 'vivarium headless' subcommand")
}
