// Lint-docs checks markdown files by other scripts.
// It is an example of a mise task in Go which runs binaries of other Go scripts.
package main

import (
	"github.com/alecthomas/kong"

	"github.com/powerman/go-scripts-example/scripts/lint-docs/internal"
)

func main() {
	var cli internal.CLI
	ctx := kong.Parse(&cli,
		kong.Description("Check that tables of contents in markdown files are up to date"+
			" and their local links aren't broken."),
		kong.UsageOnError(),
	)
	ctx.BindTo(internal.WireApp(), (*internal.App)(nil))
	ctx.FatalIfErrorf(ctx.Run())
}
