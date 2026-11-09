package internal

import (
	"os"
	"strings"

	"github.com/bitfield/script"
)

// RealWorld implements [World] by real commands and files.
type RealWorld struct{}

// GitLsFiles implements [World].
func (RealWorld) GitLsFiles(pathspecs ...string) ([]string, error) {
	args := append([]string{"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--"}, pathspecs...)
	out, err := script.NewPipe().WithStderr(os.Stderr).ExecCommand("git", args...).String()
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(strings.TrimSuffix(out, "\x00"), "\x00"), nil
}

// ReadFile implements [World].
func (RealWorld) ReadFile(name string) ([]byte, error) {
	return script.File(name).Bytes()
}

// WriteFile implements [World].
func (RealWorld) WriteFile(name string, data []byte) error {
	_, err := script.Echo(string(data)).WriteFile(name)
	return err
}
