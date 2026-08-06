//go:build !ebiten

package gui

import "errors"

func Available() bool { return false }

func Run(o Config) error {
	if _, err := buildWorld(o); err != nil {
		return err
	}
	return errors.New("built without the GUI; rebuild with -tags ebiten, or use the 'vivarium headless' subcommand")
}
