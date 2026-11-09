package internal

import "github.com/powerman/kongusage"

// CLI defines the command-line interface of the program.
type CLI struct {
	GoCache    string   `       default:"test-qemu-go-cache"                     help:"Docker volume or host directory (with a slash) for the Go cache."        placeholder:"VOLUME|DIR"`
	GoModCache string   `       default:"test-qemu-go-mod-cache"                 help:"Docker volume or host directory (with a slash) for the Go module cache." placeholder:"VOLUME|DIR"`
	Platforms  []string `arg:""                                  name:"platform" help:"Platforms of containers (e.g. linux/arm/v7)."`

	UsageSpec kongusage.Flag
}

// UsageKDL implements [kongusage.KDLProvider].
// It suggests platforms of the image for completion without limiting them.
func (CLI) UsageKDL() string {
	return `
arg "<platform>..." {
    choices strict=#false {
        choice "linux/386"
        choice "linux/amd64"
        choice "linux/arm/v6"
        choice "linux/arm/v7"
        choice "linux/arm64"
        choice "linux/ppc64le"
        choice "linux/riscv64"
        choice "linux/s390x"
    }
}
`
}

// Run executes the CLI.
func (c CLI) Run(a App) error {
	return a.Test(Options{
		Platforms:  c.Platforms,
		GoCache:    c.GoCache,
		GoModCache: c.GoModCache,
	})
}
