//go:generate mockgen -source=port.go -destination=mock.port_test.go -package=internal_test -exclude_interfaces=World

// Package internal implements task lint:docs.
package internal

import "errors"

// Errors.
var (
	ErrNoScriptsBin = errors.New("SCRIPTS_BIN is empty")
)

// App is the application.
type App interface {
	// Check runs checks of markdown files.
	// It runs all checks, even after a failed one.
	Check() error
}

// World abstracts commands for testability.
type World interface {
	// RunScript runs a binary of a Go script and waits for its exit.
	// Output of the binary goes to stdout and stderr of the program.
	RunScript(name string, args ...string) error
}
