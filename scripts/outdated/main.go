// Outdated shows what `mise run bump` would upgrade.
// It is an example of a mise task in Go which filters output of external commands by pipes.
package main

import (
	"github.com/alecthomas/kong"

	"github.com/powerman/go-scripts-example/scripts/outdated/internal"
)

func main() {
	var cli internal.CLI
	ctx := kong.Parse(&cli,
		kong.Description("Show tools of mise.toml and Go modules of scripts which `mise run bump` would upgrade."+
			" It changes nothing."),
		kong.UsageOnError(),
	)
	ctx.BindTo(internal.WireApp(), (*internal.App)(nil))
	ctx.FatalIfErrorf(ctx.Run())
}
