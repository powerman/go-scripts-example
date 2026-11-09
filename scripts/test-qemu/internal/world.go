package internal

import (
	"io"
	"os"
	"path/filepath"

	"github.com/bitfield/script"
)

// RealWorld implements [World] by real commands and files.
type RealWorld struct {
	Docker string // Name or path of the docker command.
	Stdout io.Writer
	Stderr io.Writer
}

// ReadFile implements [World].
func (RealWorld) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // Reading files given by a user is the purpose.
}

// MkdirAll implements [World].
func (RealWorld) MkdirAll(dir string) error {
	return os.MkdirAll(dir, 0o777) //nolint:gosec,mnd // Mode of a new directory, limited by umask.
}

// RunContainer implements [World].
func (w RealWorld) RunContainer(c Container) error {
	args := make([]string, 0, 7+2*len(c.Mounts)+2*len(c.Env)+len(c.Cmd))
	args = append(args, "run", "--rm", "--platform", c.Platform, "--workdir", c.Workdir)
	for _, m := range c.Mounts {
		source := m.Source
		if m.Bind { // Docker takes a relative path for a name of a volume.
			var err error
			source, err = filepath.Abs(source)
			if err != nil {
				return err
			}
		}
		v := source + ":" + m.Target
		if m.ReadOnly {
			v += ":ro"
		}
		args = append(args, "--volume", v)
	}
	for _, env := range c.Env {
		args = append(args, "--env", env)
	}
	args = append(args, c.Image)
	args = append(args, c.Cmd...)

	_, err := script.NewPipe().WithStdout(w.Stdout).WithStderr(w.Stderr).
		ExecCommand(w.Docker, args...).
		Stdout()
	return err
}
