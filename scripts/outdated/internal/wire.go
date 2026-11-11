package internal

import "os"

// WireApp wires the application with real commands and stdout.
//
//nolint:iface // Intentionally returning a port interface for wire.go.
func WireApp() App {
	return NewApp(RealWorld{Mise: "mise", Go: "go", ModDir: "scripts", Stderr: os.Stderr}, os.Stdout)
}
