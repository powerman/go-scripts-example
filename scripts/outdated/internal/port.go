//go:generate mockgen -source=port.go -destination=mock.port_test.go -package=internal_test -exclude_interfaces=World

// Package internal implements task outdated.
package internal

import "errors"

// Errors.
var (
	ErrOutdated = errors.New("something is outdated")
)

// App is the application.
type App interface {
	// Report prints upgrades which `mise run bump` would make.
	Report(opts Options) error
}

// World abstracts commands for testability.
type World interface {
	// MiseTools returns tools of mise.toml which have a newer version
	// allowed by their requested version.
	// Tools of the global config of the user aren't included.
	// A newer Go for scripts/go.mod shows up here as tool go,
	// because bump takes the Go of mise, which respects minimum_release_age.
	MiseTools() ([]Update, error)
	// GoModules returns modules which provide packages to scripts or their tests
	// and have a newer minor or patch version.
	// Upgrades of other modules forced by requirements of these newer versions aren't included.
	GoModules() ([]Update, error)
}

// Update of a tool or a module.
type Update struct {
	Name    string
	Current string
	Latest  string
}

// Level of an upgrade by semantic versioning.
type Level int

// Levels in ascending order.
const (
	LevelPatch Level = iota + 1
	LevelMinor
	LevelMajor
	// LevelOther is an upgrade between versions which aren't semver.
	// It may be of any level, so no minimal level filters it out.
	LevelOther
)

// String returns a name of a level.
func (l Level) String() string {
	switch l {
	case LevelPatch:
		return "patch"
	case LevelMinor:
		return "minor"
	case LevelMajor:
		return "major"
	case LevelOther:
		return "other"
	}
	return "unknown"
}

// Options of [App.Report].
type Options struct {
	// Min is a minimal level of reported upgrades.
	// Zero reports all upgrades.
	Min Level
	// Check makes Report fail when it reports anything.
	Check bool
}
