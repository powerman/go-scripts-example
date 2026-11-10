package internal

import "github.com/powerman/kongusage"

// CLI defines the command-line interface of the program.
type CLI struct {
	UsageSpec kongusage.Flag
}

// Run executes the CLI.
func (CLI) Run(a App) error {
	return a.Check()
}
