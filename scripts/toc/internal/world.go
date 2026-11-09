package internal

import (
	"os"

	"github.com/bitfield/script"
)

// RealWorld implements [World] by real commands and files.
type RealWorld struct{}

// GitGrepFiles implements [World].
func (RealWorld) GitGrepFiles(regexp string, pathspecs ...string) ([]string, error) {
	// core.quotePath=false prints non-ASCII paths as is.
	args := append([]string{
		"-c", "core.quotePath=false", "grep", "--untracked",
		"--files-with-matches", "--extended-regexp", "-e", regexp, "--",
	}, pathspecs...)
	p := script.NewPipe().WithStderr(os.Stderr).ExecCommand("git", args...)
	files, err := p.Slice()
	if p.ExitStatus() == 1 { // Like `git grep … || [ $? -eq 1 ]` in shell: no files found.
		return nil, nil
	}
	return files, err
}

// ReadFile implements [World].
func (RealWorld) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // Reading files given by a user is the purpose.
}

// WriteFile implements [World].
func (RealWorld) WriteFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0o666) //nolint:gosec,mnd // Mode of a new file, limited by umask.
}
