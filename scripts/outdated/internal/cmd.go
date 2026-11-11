package internal

import "github.com/powerman/kongusage"

// CLI defines the command-line interface of the program.
type CLI struct {
	Check bool   `                                         help:"Fail when anything is outdated."`
	Min   string `default:"patch" enum:"patch,minor,major" help:"Show only upgrades of this level or higher (versions which aren't semver are always shown)." placeholder:"LEVEL"`

	UsageSpec kongusage.Flag
}

// Run executes the CLI.
func (c CLI) Run(a App) error {
	opts := Options{Min: LevelPatch, Check: c.Check}
	switch c.Min {
	case "minor":
		opts.Min = LevelMinor
	case "major":
		opts.Min = LevelMajor
	}
	return a.Report(opts)
}
