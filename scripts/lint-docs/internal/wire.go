package internal

import "os"

// WireApp wires the application with real commands.
//
//nolint:iface // Intentionally returning a port interface for wire.go.
func WireApp() App {
	return NewApp(RealWorld{
		ScriptsBin: os.Getenv("SCRIPTS_BIN"),
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	})
}
