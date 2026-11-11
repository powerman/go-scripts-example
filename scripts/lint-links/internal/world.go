package internal

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

// RealWorld implements [World] by real commands and files.
type RealWorld struct{}

// GitLsFiles implements [World].
func (RealWorld) GitLsFiles() ([]string, error) {
	// core.quotePath=false prints non-ASCII paths as is.
	cmd := exec.CommandContext(context.Background(), "git",
		"-c", "core.quotePath=false", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.FieldsFunc(string(out), func(r rune) bool { return r == '\n' }), nil
}

// ReadFile implements [World].
func (RealWorld) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // Reading files given by a user is the purpose.
}
