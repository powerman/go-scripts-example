package internal_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/lint-docs/internal"
)

var errRun = errors.New("stub run error")

// newWorld returns a fake of binaries which all succeed.
func newWorld() *FakeWorld {
	w := NewFakeWorld()
	w.RunScriptFunc.SetDefaultReturn(nil)
	return w
}

// runs returns command lines of binaries run by w.
func runs(w *FakeWorld) []string {
	history := w.RunScriptFunc.History()
	lines := make([]string, 0, len(history))
	for _, c := range history {
		lines = append(lines, strings.Join(append([]string{c.Name}, c.ArgArgs...), " "))
	}
	return lines
}

var allRuns = []string{"toc --check", "lint-links"}

func TestApp_Check(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name    string
		prepare func(*FakeWorld)
		want    string
	}{
		{
			name:    "OK",
			prepare: func(*FakeWorld) {},
		},
		{
			name:    "FirstFails",
			prepare: func(w *FakeWorld) { w.RunScriptFunc.PushReturn(errRun) },
			want:    "^toc: stub run error$",
		},
		{
			name:    "AllFail",
			prepare: func(w *FakeWorld) { w.RunScriptFunc.SetDefaultReturn(errRun) },
			want:    "^toc: stub run error\nlint-links: stub run error$",
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			w := newWorld()
			tc.prepare(w)

			err := internal.NewApp(w).Check()
			if tc.want == "" {
				t.Nil(err)
			} else {
				t.Err(err, errRun)
				t.Match(err, tc.want)
			}
			t.DeepEqual(runs(w), allRuns)
		})
	}
}
