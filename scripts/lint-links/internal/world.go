package internal

import (
	"os"

	"github.com/bitfield/script"
)

// RealWorld implements [World] by real commands and files.
type RealWorld struct{}

// GitLsFiles implements [World].
func (RealWorld) GitLsFiles() ([]string, error) {
	// core.quotePath=false prints non-ASCII paths as is.
	return script.NewPipe().WithStderr(os.Stderr).
		ExecCommand("git", "-c", "core.quotePath=false", "ls-files", "--cached", "--others", "--exclude-standard").
		Slice()
}

// ReadFile implements [World].
func (RealWorld) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // Reading files given by a user is the purpose.
}
