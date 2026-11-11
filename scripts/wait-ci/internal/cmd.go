package internal

import (
	"time"

	"github.com/powerman/kongusage"
)

// CLI defines the command-line interface of the program.
type CLI struct {
	Timeout time.Duration `                   default:"15m"  help:"Fail when runs aren't completed in this time."  placeholder:"DURATION"`
	Poll    time.Duration `                   default:"10s"  help:"Interval between checks of runs."               placeholder:"DURATION"`
	Appear  time.Duration `                   default:"60s"  help:"Fail when the commit has no runs in this time." placeholder:"DURATION"`
	Commit  string        `arg:"" optional:"" default:"HEAD" help:"Git revision of a pushed commit."`

	UsageSpec kongusage.Flag
}

// Run executes the CLI.
func (c CLI) Run(a App) error {
	return a.Wait(Options{
		Commit:  c.Commit,
		Timeout: c.Timeout,
		Poll:    c.Poll,
		Appear:  c.Appear,
	})
}
