//go:build !ebiten

package gui

import "errors"

func Available() bool { return false }

func Run(o Config) error {
	world, err := buildWorld(o)
	if err != nil || world == nil {
		// A nil world with no error means --print-config already printed
		// everything the run was asked for; there is no window to miss.
		return err
	}
	return errors.New("built without the GUI; rebuild with -tags ebiten, or use the 'vivarium headless' subcommand")
}
