// Wait-ci waits for CI runs of a commit and reports failed ones.
// It is an example of a mise task in Go which orchestrates external commands
// with polling, time and timeouts.
package main

import (
	"github.com/alecthomas/kong"

	"github.com/powerman/go-scripts-example/scripts/wait-ci/internal"
)

func main() {
	var cli internal.CLI
	ctx := kong.Parse(&cli,
		kong.Description("Wait until GitHub Actions runs of a pushed commit are completed."+
			" It prints a line on each change of a status of a run,"+
			" and logs of failed jobs when some runs failed."),
		kong.UsageOnError(),
	)
	ctx.BindTo(internal.WireApp(), (*internal.App)(nil))
	ctx.FatalIfErrorf(ctx.Run())
}
