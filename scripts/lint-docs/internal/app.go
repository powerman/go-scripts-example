package internal

import (
	"errors"
	"fmt"
)

type app struct {
	world World
}

// NewApp creates an App which uses world.
func NewApp(world World) *app {
	return &app{world: world}
}

// Check implements [App].
func (a *app) Check() error {
	checks := [][]string{ // Binaries of Go scripts with their arguments.
		{"toc", "--check"},
		{"lint-links"},
	}
	var errs []error
	for _, c := range checks {
		err := a.world.RunScript(c[0], c[1:]...)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c[0], err))
		}
	}
	return errors.Join(errs...)
}
