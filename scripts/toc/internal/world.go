package internal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

// RealWorld implements [World] by real commands and files.
type RealWorld struct{}

// TOCFiles implements [World].
func (RealWorld) TOCFiles() ([]string, error) {
	// core.quotePath=false prints non-ASCII paths as is.
	cmd := exec.CommandContext(context.Background(), "git",
		"-c", "core.quotePath=false", "grep", "--untracked",
		"--files-with-matches", "--extended-regexp", "-e", "^<!-- "+MarkerStart+" -->$",
		"--", "*.md")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err, ok := errors.AsType[*exec.ExitError](err); ok && err.ExitCode() == 1 { //nolint:noinlineerr // Converts err for the condition.
		return nil, nil // Like `git grep … || [ $? -eq 1 ]` in shell: no files found.
	}
	if err != nil {
		return nil, err
	}
	return strings.FieldsFunc(string(out), func(r rune) bool { return r == '\n' }), nil
}

// ReadFile implements [World].
func (RealWorld) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // Reading files given by a user is the purpose.
}

// WriteFile implements [World].
func (RealWorld) WriteFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0o666) //nolint:gosec,mnd // Mode of a new file, limited by umask.
}
