package internal_test

import (
	"bytes"
	"errors"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/lint-links/internal"
)

var (
	errGit  = errors.New("stub git error")
	errRead = errors.New("stub read error")
)

// world is a stateful fake of a repository with files.
type world struct {
	*FakeWorld

	Files map[string]string
}

func newWorld(files map[string]string) *world {
	w := &world{FakeWorld: NewFakeWorld(), Files: make(map[string]string)}
	maps.Copy(w.Files, files)
	w.GitLsFilesFunc.SetDefaultHook(w.gitLsFiles)
	w.ReadFileFunc.SetDefaultHook(w.readFile)
	return w
}

func (w *world) gitLsFiles() ([]string, error) {
	return slices.Sorted(maps.Keys(w.Files)), nil
}

func (w *world) readFile(name string) ([]byte, error) {
	data, ok := w.Files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(data), nil
}

// md joins lines of a markdown file.
func md(lines ...string) string { return strings.Join(lines, "\n") + "\n" }

var files = map[string]string{
	"README.md": md(
		"# Title",
		"## Section `x`",
		"See [docs](docs/guide.md), [section](#section-x),",
		"[guide section](docs/guide.md#usage), ![logo](docs/img/logo%20v2.png),",
		"[dir](docs/), [code](main.go#L10), [root](/LICENSE?plain=1),",
		"[site](https://example.com/missing.md), <https://example.com>, [mail](mailto:a@b.c).",
	),
	"LICENSE": "MIT",
	"main.go": "package main",
	"docs/guide.md": md(
		"# Guide",
		"## Usage",
		"Back to [README](../README.md#title) and [usage](#usage).",
	),
	"docs/img/logo v2.png": "",
}

func TestApp_Check(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name    string
		files   map[string]string
		args    []string
		want    string
		wantErr error
	}{
		{
			name:  "Valid",
			files: files,
		},
		{
			name: "Broken",
			files: map[string]string{
				"README.md": md(
					"# Title",
					"[a](missing.md) [b](#missing)",
					"[c](docs/guide.md#missing) [d](../outside.md)",
					"[e](%zz)",
				),
				"docs/guide.md": md("# Guide", "[f](/docs/README.md)"),
				"notes.txt":     "[g](missing.md)",
			},
			want: strings.Join([]string{
				`README.md:2: link "missing.md": no missing.md in the repository`,
				`README.md:2: link "#missing": no heading #missing in README.md`,
				`README.md:3: link "docs/guide.md#missing": no heading #missing in docs/guide.md`,
				`README.md:3: link "../outside.md": no ../outside.md in the repository`,
				`README.md:4: invalid link "%zz"`,
				`docs/guide.md:2: link "/docs/README.md": no docs/README.md in the repository`,
				``,
			}, "\n"),
			wantErr: internal.ErrBrokenLinks,
		},
		{
			name: "GivenFiles",
			files: map[string]string{
				"a.md":      "[x](missing.md)",
				"docs/b.md": "[y](missing.md)",
			},
			args:    []string{`docs/./b.md`},
			want:    "docs/b.md:1: link \"missing.md\": no docs/missing.md in the repository\n",
			wantErr: internal.ErrBrokenLinks,
		},
		{
			name:    "NoFile",
			args:    []string{"none.md"},
			wantErr: fs.ErrNotExist,
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			w := newWorld(tc.files)
			var out bytes.Buffer

			err := internal.NewApp(w, &out).Check(tc.args)
			t.Err(err, tc.wantErr)
			t.Equal(out.String(), tc.want)
		})
	}
}

func TestApp_Check_ParseOnce(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	w := newWorld(files)

	err := internal.NewApp(w, &bytes.Buffer{}).Check(nil)
	t.Nil(err)
	history := w.ReadFileFunc.History()
	names := make([]string, 0, len(history))
	for _, call := range history {
		names = append(names, call.Name)
	}
	t.DeepEqual(names, []string{"README.md", "docs/guide.md"})
}

func TestApp_Check_Errors(tt *testing.T) {
	tt.Parallel()

	tt.Run("GitLsFiles", func(tt *testing.T) {
		tt.Parallel()
		t := check.T(tt).MustAll()
		w := newWorld(files)
		w.GitLsFilesFunc.PushReturn(nil, errGit)

		err := internal.NewApp(w, &bytes.Buffer{}).Check(nil)
		t.Err(err, errGit)
	})

	tt.Run("ReadTarget", func(tt *testing.T) {
		tt.Parallel()
		t := check.T(tt).MustAll()
		w := newWorld(files)
		w.ReadFileFunc.PushHook(w.readFile) // README.md
		w.ReadFileFunc.PushReturn(nil, errRead)

		err := internal.NewApp(w, &bytes.Buffer{}).Check(nil)
		t.Err(err, errRead)
	})
}
