//go:generate mockgen -source=port.go -destination=mock.port_test.go -package=internal_test -exclude_interfaces=World

// Package internal implements task lint:links.
package internal

import "errors"

// Errors.
var (
	ErrBrokenLinks = errors.New("broken links")
)

// App is the application.
type App interface {
	// Check reports broken links in markdown files.
	// It checks all markdown files of the repository when files are not given.
	Check(files []string) error
}

// World abstracts commands and files for testability.
type World interface {
	// GitLsFiles returns paths of tracked and untracked (but not ignored) files
	// which match pathspecs, relative to the current directory and separated by '/'.
	GitLsFiles(pathspecs ...string) ([]string, error)
	ReadFile(name string) ([]byte, error)
}
