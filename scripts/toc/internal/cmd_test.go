package internal_test

import (
	"errors"
	"io"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"
	"github.com/powerman/kongusage"
	"go.uber.org/mock/gomock"

	"github.com/powerman/go-scripts-example/scripts/toc/internal"
)

var errApp = errors.New("stub app error")

// kongRun parses args like main and runs the CLI with a.
func kongRun(t *check.C, a internal.App, args ...string) error {
	t.Helper()
	var cli internal.CLI
	k, err := kong.New(&cli,
		kong.Writers(io.Discard, io.Discard),
		kong.Exit(func(int) {}),
	)
	t.Nil(err)
	ctx, err := k.Parse(args)
	if err != nil {
		return err
	}
	ctx.BindTo(a, (*internal.App)(nil))
	return ctx.Run()
}

func TestCLI_Run(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name string
		args []string
		want internal.Options
	}{
		{
			name: "Defaults",
			want: internal.Options{MinLevel: 2, MaxLevel: 3},
		},
		{
			name: "All",
			args: []string{"--check", "--min-level=1", "--max-level", "6", "a.md", "docs/b c.md"},
			want: internal.Options{
				Files:    []string{"a.md", "docs/b c.md"},
				MinLevel: 1,
				MaxLevel: 6,
				Check:    true,
			},
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			a := NewMockApp(gomock.NewController(t))
			a.EXPECT().Update(tc.want).Return(errApp)

			err := kongRun(t, a, tc.args...)
			t.Err(err, errApp)
		})
	}
}

func TestCLI_Run_InvalidLevel(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	a := NewMockApp(gomock.NewController(t))

	err := kongRun(t, a, "--max-level=7")
	t.Match(err, `--max-level must be one of "1","2","3","4","5","6" but got "7"`)
}

func TestUsageSpec(t *testing.T) {
	t.Parallel()
	kongusage.CheckSpec(t, kong.Must(&internal.CLI{}))
}
