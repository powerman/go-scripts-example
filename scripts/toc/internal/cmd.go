package internal

import "github.com/powerman/kongusage"

// CLI defines the command-line interface of the program.
type CLI struct {
	Check    bool     `                                                              help:"Fail on an outdated table of contents instead of updating it."`
	MinLevel int      `                   default:"2"             enum:"1,2,3,4,5,6" help:"Include headings starting from this level."                    placeholder:"N"`
	MaxLevel int      `                   default:"3"             enum:"1,2,3,4,5,6" help:"Include headings up to this level."                            placeholder:"N"`
	Files    []string `arg:"" optional:""             name:"file"                    help:"Markdown files (default: all files with a table of contents)."`

	UsageSpec kongusage.Flag
}

// UsageKDL adds examples to the usage spec.
func (CLI) UsageKDL() string {
	return `
example "mise run toc" header="Update all tables of contents"
example "mise run toc --check README.md" header="Check a table of contents in README.md"
`
}

// Run executes the CLI.
func (c CLI) Run(a App) error {
	return a.Update(Options{
		Files:    c.Files,
		MinLevel: c.MinLevel,
		MaxLevel: c.MaxLevel,
		Check:    c.Check,
	})
}
