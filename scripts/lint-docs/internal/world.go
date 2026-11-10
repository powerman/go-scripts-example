package internal

import (
	"io"
	"path/filepath"

	"github.com/bitfield/script"
)

// RealWorld implements [World] by real commands.
type RealWorld struct {
	ScriptsBin string // Directory of binaries of Go scripts.
	Stdout     io.Writer
	Stderr     io.Writer
}

// RunScript implements [World].
func (w RealWorld) RunScript(name string, args ...string) error {
	if w.ScriptsBin == "" { // A bare name would be looked up in PATH.
		return ErrNoScriptsBin
	}
	_, err := script.NewPipe().WithStdout(w.Stdout).WithStderr(w.Stderr).
		ExecCommand(filepath.Join(w.ScriptsBin, name), args...).
		Stdout()
	return err
}
