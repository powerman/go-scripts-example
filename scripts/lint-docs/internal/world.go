package internal

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
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
	cmd := exec.CommandContext(context.Background(), filepath.Join(w.ScriptsBin, name), args...) //nolint:gosec // Running a binary of a Go script is the purpose.
	cmd.Stdout = w.Stdout
	cmd.Stderr = w.Stderr
	return cmd.Run()
}
