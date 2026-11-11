package internal_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/outdated/internal"
)

var (
	errMise = errors.New("stub mise error")
	errGo   = errors.New("stub go error")
)

// world is a stateful fake of tools and modules with newer versions.
type world struct {
	*FakeWorld

	Tools   []internal.Update
	Modules []internal.Update
}

func newWorld(tools, modules []internal.Update) *world {
	w := &world{FakeWorld: NewFakeWorld(), Tools: tools, Modules: modules}
	w.MiseToolsFunc.SetDefaultHook(func() ([]internal.Update, error) { return w.Tools, nil })
	w.GoModulesFunc.SetDefaultHook(func() ([]internal.Update, error) { return w.Modules, nil })
	return w
}

// table returns lines of a report.
func table(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func TestApp_Report(tt *testing.T) {
	tt.Parallel()

	tools := []internal.Update{
		{Name: "usage", Current: "7.1.5", Latest: "7.2.0"},
		{Name: "go", Current: "1.27.3", Latest: "1.27.4"},
		{Name: "java", Current: "temurin-21", Latest: "temurin-25"},
		{Name: "node", Current: "24.11.0", Latest: "26.1.0"},
	}
	modules := []internal.Update{
		{Name: "golang.org/x/mod", Current: "v0.42.0", Latest: "v0.43.0"},
		{Name: "example.com/a", Current: "v1.2.3", Latest: "v1.2.4"},
		{Name: "example.com/b", Current: "v0.0.0-20260101000000-abcdefabcdef", Latest: "v0.0.0-20261101000000-abcdefabcdef"},
	}

	tests := []struct {
		name    string
		opts    internal.Options
		tools   []internal.Update
		modules []internal.Update
		want    string
		wantErr error
	}{
		{
			name:    "All",
			tools:   tools,
			modules: modules,
			want: table(
				"SOURCE  NAME              CURRENT                             LATEST                              LEVEL",
				"go      example.com/a     v1.2.3                              v1.2.4                              patch",
				"go      example.com/b     v0.0.0-20260101000000-abcdefabcdef  v0.0.0-20261101000000-abcdefabcdef  patch",
				"go      golang.org/x/mod  v0.42.0                             v0.43.0                             minor",
				"mise    go                1.27.3                              1.27.4                              patch",
				"mise    java              temurin-21                          temurin-25                          other",
				"mise    node              24.11.0                             26.1.0                              major",
				"mise    usage             7.1.5                               7.2.0                               minor",
			),
		},
		{
			name:    "MinPatch",
			opts:    internal.Options{Min: internal.LevelPatch},
			modules: modules[:2],
			want: table(
				"SOURCE  NAME              CURRENT  LATEST   LEVEL",
				"go      example.com/a     v1.2.3   v1.2.4   patch",
				"go      golang.org/x/mod  v0.42.0  v0.43.0  minor",
			),
		},
		{
			name:    "MinMinor",
			opts:    internal.Options{Min: internal.LevelMinor},
			tools:   tools,
			modules: modules,
			want: table(
				"SOURCE  NAME              CURRENT     LATEST      LEVEL",
				"go      golang.org/x/mod  v0.42.0     v0.43.0     minor",
				"mise    java              temurin-21  temurin-25  other",
				"mise    node              24.11.0     26.1.0      major",
				"mise    usage             7.1.5       7.2.0       minor",
			),
		},
		{
			name:    "MinMajor",
			opts:    internal.Options{Min: internal.LevelMajor},
			tools:   tools,
			modules: modules,
			want: table(
				"SOURCE  NAME  CURRENT     LATEST      LEVEL",
				"mise    java  temurin-21  temurin-25  other",
				"mise    node  24.11.0     26.1.0      major",
			),
		},
		{
			name:    "Filtered",
			opts:    internal.Options{Min: internal.LevelMajor, Check: true},
			modules: modules,
		},
		{
			name: "Nothing",
			opts: internal.Options{Check: true},
		},
		{
			name:    "Check",
			opts:    internal.Options{Check: true},
			tools:   tools[:1],
			want:    table("SOURCE  NAME   CURRENT  LATEST  LEVEL", "mise    usage  7.1.5    7.2.0   minor"),
			wantErr: internal.ErrOutdated,
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			w := newWorld(tc.tools, tc.modules)
			var out bytes.Buffer

			err := internal.NewApp(w, &out).Report(tc.opts)
			t.Err(err, tc.wantErr)
			t.Equal(out.String(), tc.want)
		})
	}
}

func TestApp_Report_Errors(tt *testing.T) {
	tt.Parallel()

	tools := []internal.Update{{Name: "usage", Current: "7.1.5", Latest: "7.2.0"}}

	tests := []struct {
		name    string
		prepare func(*world)
		want    string
	}{
		{
			name:    "Mise",
			prepare: func(w *world) { w.MiseToolsFunc.PushReturn(nil, errMise) },
			want:    "^mise: stub mise error$",
		},
		{
			name:    "Go",
			prepare: func(w *world) { w.GoModulesFunc.PushReturn(nil, errGo) },
			want:    "^go: stub go error$",
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			w := newWorld(tools, nil)
			tc.prepare(w)
			var out bytes.Buffer

			err := internal.NewApp(w, &out).Report(internal.Options{})
			t.Match(err, tc.want)
			t.Equal(out.String(), "")
		})
	}
}
