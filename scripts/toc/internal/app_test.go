package internal_test

import (
	"bytes"
	"errors"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/toc/internal"
)

var (
	errGit   = errors.New("stub git error")
	errRead  = errors.New("stub read error")
	errWrite = errors.New("stub write error")
)

// world is a stateful fake of a repository with files.
type world struct {
	*FakeWorld

	Files map[string]string
}

func newWorld(files map[string]string) *world {
	w := &world{FakeWorld: NewFakeWorld(), Files: make(map[string]string)}
	maps.Copy(w.Files, files)
	w.GitGrepFilesFunc.SetDefaultHook(w.gitGrepFiles)
	w.ReadFileFunc.SetDefaultHook(w.readFile)
	w.WriteFileFunc.SetDefaultHook(w.writeFile)
	return w
}

func (w *world) gitGrepFiles(pattern string, pathspecs ...string) ([]string, error) {
	re := regexp.MustCompile("(?m)" + pattern)
	var names []string
	for name, data := range w.Files {
		for _, pathspec := range pathspecs {
			// Git pathspec "*.md" matches files in subdirectories too.
			ok, _ := path.Match(pathspec, path.Base(name))
			if ok && re.MatchString(data) {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return names, nil
}

func (w *world) readFile(name string) ([]byte, error) {
	data, ok := w.Files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(data), nil
}

func (w *world) writeFile(name string, data []byte) error {
	w.Files[name] = string(data)
	return nil
}

// md joins lines of a markdown file.
func md(lines ...string) string { return strings.Join(lines, "\n") + "\n" }

var (
	opts = internal.Options{MinLevel: 2, MaxLevel: 3}

	readme = md(
		"# Title",
		"",
		"<!-- toc -->",
		"<!-- tocstop -->",
		"",
		"## One",
		"### One `a`",
		"#### One a x",
		"## Two",
	)
	readmeTOC = md(
		"# Title",
		"",
		"<!-- toc -->",
		"",
		"- [One](#one)",
		"  - [One `a`](#one-a)",
		"- [Two](#two)",
		"",
		"<!-- tocstop -->",
		"",
		"## One",
		"### One `a`",
		"#### One a x",
		"## Two",
	)
	plain = md("# Plain", "## Section")
)

func TestApp_Update(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name      string
		files     map[string]string
		opts      internal.Options
		wantFiles map[string]string
		wantOut   string
		wantErr   error
	}{
		{
			name:      "AllFiles",
			files:     map[string]string{"README.md": readme, "docs/a.md": readme, "plain.md": plain, "x.txt": readme},
			opts:      opts,
			wantFiles: map[string]string{"README.md": readmeTOC, "docs/a.md": readmeTOC, "plain.md": plain, "x.txt": readme},
			wantOut:   "updated README.md\nupdated docs/a.md\n",
		},
		{
			name:      "GivenFiles",
			files:     map[string]string{"a.md": readme, "b.md": readme},
			opts:      internal.Options{Files: []string{"b.md"}, MinLevel: 2, MaxLevel: 3},
			wantFiles: map[string]string{"a.md": readme, "b.md": readmeTOC},
			wantOut:   "updated b.md\n",
		},
		{
			name:      "UpToDate",
			files:     map[string]string{"README.md": readmeTOC},
			opts:      opts,
			wantFiles: map[string]string{"README.md": readmeTOC},
		},
		{
			name:      "Check",
			files:     map[string]string{"a.md": readme, "b.md": readmeTOC},
			opts:      internal.Options{MinLevel: 2, MaxLevel: 3, Check: true},
			wantFiles: map[string]string{"a.md": readme, "b.md": readmeTOC},
			wantErr:   internal.ErrOutdated,
		},
		{
			name:  "Levels",
			files: map[string]string{"a.md": readme},
			opts:  internal.Options{MinLevel: 3, MaxLevel: 4},
			wantFiles: map[string]string{"a.md": strings.Replace(readme, "<!-- toc -->\n", md(
				"<!-- toc -->",
				"",
				"- [One `a`](#one-a)",
				"  - [One a x](#one-a-x)",
				"",
			), 1)},
			wantOut: "updated a.md\n",
		},
		{
			name:  "SkippedLevel",
			files: map[string]string{"a.md": md("<!-- toc -->", "<!-- tocstop -->", "#### Deep", "## Two", "#### Deep")},
			opts:  internal.Options{MinLevel: 2, MaxLevel: 4},
			wantFiles: map[string]string{"a.md": md(
				"<!-- toc -->",
				"",
				"- [Deep](#deep)",
				"- [Two](#two)",
				"  - [Deep](#deep-1)",
				"",
				"<!-- tocstop -->",
				"#### Deep",
				"## Two",
				"#### Deep",
			)},
			wantOut: "updated a.md\n",
		},
		{
			name:      "NoHeadings",
			files:     map[string]string{"a.md": md("# Title", "<!-- toc -->", "", "- [Old](#old)", "", "<!-- tocstop -->")},
			opts:      opts,
			wantFiles: map[string]string{"a.md": md("# Title", "<!-- toc -->", "", "<!-- tocstop -->")},
			wantOut:   "updated a.md\n",
		},
		{
			name:      "MarkerInCode",
			files:     map[string]string{"a.md": md("```", "<!-- toc -->", "<!-- tocstop -->", "```")},
			opts:      opts,
			wantFiles: map[string]string{"a.md": md("```", "<!-- toc -->", "<!-- tocstop -->", "```")},
		},
		{
			name:      "NoMarker",
			files:     map[string]string{"a.md": plain},
			opts:      internal.Options{Files: []string{"a.md"}, MinLevel: 2, MaxLevel: 3},
			wantFiles: map[string]string{"a.md": plain},
			wantErr:   internal.ErrNoMarker,
		},
		{
			name:      "NoEndMarker",
			files:     map[string]string{"a.md": md("<!-- tocstop -->", "<!-- toc -->")},
			opts:      opts,
			wantFiles: map[string]string{"a.md": md("<!-- tocstop -->", "<!-- toc -->")},
			wantErr:   internal.ErrNoEndMarker,
		},
		{
			name:      "NoFile",
			opts:      internal.Options{Files: []string{"a.md"}, MinLevel: 2, MaxLevel: 3},
			wantFiles: make(map[string]string),
			wantErr:   fs.ErrNotExist,
		},
		{
			name:      "InvalidLevels",
			files:     map[string]string{"a.md": readme},
			opts:      internal.Options{MinLevel: 3, MaxLevel: 2},
			wantFiles: map[string]string{"a.md": readme},
			wantErr:   internal.ErrInvalidLevels,
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			w := newWorld(tc.files)
			var out bytes.Buffer

			err := internal.NewApp(w, &out).Update(tc.opts)
			t.Err(err, tc.wantErr)
			t.DeepEqual(w.Files, tc.wantFiles)
			t.Equal(out.String(), tc.wantOut)
		})
	}
}

func TestApp_Update_Errors(tt *testing.T) {
	tt.Parallel()

	tt.Run("GitGrepFiles", func(tt *testing.T) {
		tt.Parallel()
		t := check.T(tt).MustAll()
		w := newWorld(nil)
		w.GitGrepFilesFunc.PushReturn(nil, errGit)

		err := internal.NewApp(w, &bytes.Buffer{}).Update(opts)
		t.Err(err, errGit)
	})

	tt.Run("ContinueAfterError", func(tt *testing.T) {
		tt.Parallel()
		t := check.T(tt).MustAll()
		w := newWorld(map[string]string{"a.md": readme, "b.md": readme, "c.md": readme})
		w.ReadFileFunc.PushHook(w.readFile) // a.md
		w.ReadFileFunc.PushReturn(nil, errRead)
		w.WriteFileFunc.PushReturn(errWrite) // a.md
		var out bytes.Buffer

		err := internal.NewApp(w, &out).Update(opts)
		t.Match(err, `^a.md: stub write error\nb.md: stub read error$`)
		t.DeepEqual(w.Files, map[string]string{"a.md": readme, "b.md": readme, "c.md": readmeTOC})
		t.Equal(out.String(), "updated c.md\n")
	})
}
