// Lint-links checks local links in markdown files.
// It is an example of a mise task in Go.
package main

import (
	"github.com/alecthomas/kong"

	"github.com/powerman/go-scripts-example/scripts/lint-links/internal"
)

func main() {
	var cli internal.CLI
	ctx := kong.Parse(&cli,
		kong.Name("lint:links"),
		kong.Description("Check that local links in markdown files point to files of the repository"+
			" and to existing headings of markdown files."),
		kong.UsageOnError(),
	)
	ctx.BindTo(internal.WireApp(), (*internal.App)(nil))
	ctx.FatalIfErrorf(ctx.Run())
}
