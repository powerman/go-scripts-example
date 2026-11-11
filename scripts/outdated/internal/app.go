package internal

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"golang.org/x/mod/semver"
)

// Names of sources of upgrades, shown in the report.
const (
	sourceMise = "mise"
	sourceGo   = "go"
)

type app struct {
	world  World
	stdout io.Writer
}

// NewApp creates an App which uses world and prints a report to stdout.
func NewApp(world World, stdout io.Writer) *app {
	return &app{world: world, stdout: stdout}
}

type upgrade struct {
	Update

	source string
	level  Level
}

// Report implements [App].
// It prints nothing when nothing is outdated.
func (a *app) Report(opts Options) error {
	sources := []struct {
		name    string
		updates func() ([]Update, error)
	}{
		{sourceMise, a.world.MiseTools},
		{sourceGo, a.world.GoModules},
	}
	var upgrades []upgrade
	for _, src := range sources {
		updates, err := src.updates()
		if err != nil {
			return fmt.Errorf("%s: %w", src.name, err)
		}
		for _, u := range updates {
			level := classify(u.Current, u.Latest)
			if level >= opts.Min {
				upgrades = append(upgrades, upgrade{Update: u, source: src.name, level: level})
			}
		}
	}
	if len(upgrades) == 0 {
		return nil
	}

	slices.SortFunc(upgrades, func(a, b upgrade) int {
		return cmp.Or(cmp.Compare(a.source, b.source), cmp.Compare(a.Name, b.Name))
	})
	var table strings.Builder
	table.WriteString("SOURCE\tNAME\tCURRENT\tLATEST\tLEVEL\n")
	for _, u := range upgrades {
		table.WriteString(strings.Join([]string{u.source, u.Name, u.Current, u.Latest, u.level.String()}, "\t") + "\n")
	}
	w := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0) //nolint:mnd // Padding between columns.
	_, err := io.WriteString(w, table.String())
	if err != nil {
		return err
	}
	err = w.Flush()
	if err != nil {
		return err
	}

	if opts.Check {
		return ErrOutdated
	}
	return nil
}

// classify returns a level of an upgrade from current to latest.
// Versions of mise tools have no "v" prefix of semver.
func classify(current, latest string) Level {
	current, latest = semverOf(current), semverOf(latest)
	switch {
	case !semver.IsValid(current) || !semver.IsValid(latest):
		return LevelOther
	case semver.Major(current) != semver.Major(latest):
		return LevelMajor
	case semver.MajorMinor(current) != semver.MajorMinor(latest):
		return LevelMinor
	default:
		return LevelPatch
	}
}

func semverOf(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}
