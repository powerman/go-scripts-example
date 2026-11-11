package internal

import (
	"encoding/json"
	"io"
	"regexp"

	"github.com/bitfield/script"
)

var reNonEmpty = regexp.MustCompile(`.`)

// RealWorld implements [World] by real commands.
type RealWorld struct {
	Mise   string // Name or path of the mise command.
	Go     string // Name or path of the go command.
	ModDir string // Directory of the Go module of scripts.
	Stderr io.Writer
}

// update is a JSON value printed by queries of jq.
type update struct {
	Name    string `json:"name"`
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

// MiseTools implements [World].
func (w RealWorld) MiseTools() ([]Update, error) {
	// The output is an object with a field per tool, empty when nothing is outdated.
	return decode(script.NewPipe().WithStderr(w.Stderr).
		ExecCommand(w.Mise, "outdated", "--local", "--json").
		JQ(`.[] | {name, current, latest}`))
}

// GoModules implements [World].
func (w RealWorld) GoModules() ([]Update, error) {
	// Packages of the standard library have no module and print empty lines.
	// Freq and Column remove duplicates, like `sort | uniq -c | awk '{print $2}'` in shell.
	modules, err := script.NewPipe().WithStderr(w.Stderr).
		ExecCommand(w.Go, "-C", w.ModDir, "list", "-deps", "-test",
			"-f", "{{with .Module}}{{if not .Main}}{{.Path}}{{end}}{{end}}", "./...").
		MatchRegexp(reNonEmpty).
		Freq().
		Column(2). //nolint:mnd // Column after the count of Freq.
		Slice()
	if err != nil {
		return nil, err
	}

	// Without modules in arguments it prints only the main module, which has no update.
	// The output is a stream of JSON objects, one per module, each on several lines.
	args := append([]string{"-C", w.ModDir, "list", "-m", "-u", "-json"}, modules...)
	return decode(script.NewPipe().WithStderr(w.Stderr).
		ExecCommand(w.Go, args...).
		JQ(`select(.Update) | {name: .Path, current: .Version, latest: .Update.Version}`))
}

// decode returns updates from JSON values printed by p, one per line.
func decode(p *script.Pipe) ([]Update, error) {
	lines, err := p.Slice()
	if err != nil {
		return nil, err
	}
	updates := make([]Update, 0, len(lines))
	for _, line := range lines {
		var u update
		err = json.Unmarshal([]byte(line), &u)
		if err != nil {
			return nil, err
		}
		updates = append(updates, Update(u))
	}
	return updates, nil
}
