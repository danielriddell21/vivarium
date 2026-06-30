//go:build !ebiten

package gui

import "errors"

func Available() bool { return false }

func Run(_ Config) error {
	return errors.New("built without the GUI; rebuild with -tags ebiten, or use the 'vivarium headless' subcommand")
}
