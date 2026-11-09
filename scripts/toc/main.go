// Toc updates tables of contents in markdown files.
// It is an example of a mise task in Go.
package main

import (
	"github.com/alecthomas/kong"

	"github.com/powerman/go-scripts-example/scripts/toc/internal"
)

func main() {
	var cli internal.CLI
	ctx := kong.Parse(&cli,
		kong.Name("toc"),
		kong.Description("Update tables of contents between <!-- toc --> and <!-- tocstop --> in markdown files."),
		kong.UsageOnError(),
	)
	ctx.BindTo(internal.WireApp(), (*internal.App)(nil))
	ctx.FatalIfErrorf(ctx.Run())
}
