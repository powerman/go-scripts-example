//go:generate mockgen -source=port.go -destination=mock.port_test.go -package=internal_test -exclude_interfaces=World

// Package internal implements task test-qemu.
package internal

import "errors"

// ModFile is go.mod of scripts, relative to the root of the repository.
// Its Go version selects the image of a container.
const ModFile = "scripts/go.mod"

// Errors.
var (
	ErrNoGoVersion = errors.New("no Go version")
	ErrCannotRun   = errors.New("can't run a container")
)

// App is the application.
type App interface {
	// Test tests and runs scripts in a container of each platform.
	// It continues with the next platform after a failure.
	Test(opts Options) error
}

// World abstracts commands and files for testability.
type World interface {
	ReadFile(name string) ([]byte, error)
	MkdirAll(dir string) error
	// RunContainer runs a container and waits for its exit.
	// Output of the container goes to stdout and stderr of the program.
	RunContainer(c Container) error
}

// Container to run.
type Container struct {
	Image    string
	Platform string
	Workdir  string
	Mounts   []Mount
	Env      []string // NAME=value.
	Cmd      []string
}

// Mount of a docker volume or a host directory into a container.
type Mount struct {
	Source   string // Name of a docker volume or a path of a host directory.
	Target   string
	Bind     bool // Source is a host directory.
	ReadOnly bool
}

// Options of [App.Test].
type Options struct {
	Platforms []string
	// GoCache and GoModCache are names of docker volumes or paths of host directories.
	GoCache    string
	GoModCache string
}
