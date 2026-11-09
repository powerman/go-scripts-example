package internal

import "github.com/powerman/kongusage"

// CLI defines the command-line interface of the program.
type CLI struct {
	Files []string `arg:"" optional:"" name:"file" help:"Markdown files (default: all markdown files of the repository)."`

	UsageSpec kongusage.Flag
}

// Run executes the CLI.
func (c CLI) Run(a App) error {
	return a.Check(c.Files)
}
