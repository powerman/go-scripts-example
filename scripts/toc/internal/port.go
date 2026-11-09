//go:generate mockgen -source=port.go -destination=mock.port_test.go -package=internal_test -exclude_interfaces=World

// Package internal implements task toc.
package internal

import "errors"

// Texts of HTML comments around a table of contents.
const (
	MarkerStart = "toc"
	MarkerEnd   = "tocstop"
)

// Errors.
var (
	ErrInvalidLevels = errors.New("invalid range of heading levels")
	ErrNoMarker      = errors.New("no <!-- " + MarkerStart + " --> marker")
	ErrNoEndMarker   = errors.New("no <!-- " + MarkerEnd + " --> marker after <!-- " + MarkerStart + " -->")
	ErrOutdated      = errors.New("outdated table of contents")
)

// App is the application.
type App interface {
	// Update updates tables of contents in markdown files.
	Update(opts Options) error
}

// World abstracts commands and files for testability.
type World interface {
	// GitLsFiles returns paths of tracked and untracked (but not ignored) files
	// which match pathspecs, relative to the current directory and separated by '/'.
	GitLsFiles(pathspecs ...string) ([]string, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte) error
}

// Options of [App.Update].
type Options struct {
	// Files to update.
	// Default is all markdown files with a table of contents.
	Files []string
	// MinLevel and MaxLevel set the range of heading levels to include.
	MinLevel int
	MaxLevel int
	// Check makes Update fail on an outdated table of contents instead of updating it.
	Check bool
}
