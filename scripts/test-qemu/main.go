// Test-qemu tests Go scripts in containers of other platforms emulated by QEMU.
// It is an example of a mise task in Go which orchestrates external commands.
package main

import (
	"github.com/alecthomas/kong"

	"github.com/powerman/go-scripts-example/scripts/test-qemu/internal"
)

func main() {
	var cli internal.CLI
	ctx := kong.Parse(&cli,
		kong.Description("Test Go scripts and run them in docker containers of given platforms."+
			" A foreign platform needs QEMU: when its container can't run, the program prints how to set it up."),
		kong.UsageOnError(),
	)
	ctx.BindTo(internal.WireApp(), (*internal.App)(nil))
	ctx.FatalIfErrorf(ctx.Run())
}
