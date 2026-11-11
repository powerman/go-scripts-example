package internal_test

import (
	"errors"
	"io"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"
	"github.com/powerman/kongusage"
	"go.uber.org/mock/gomock"

	"github.com/powerman/go-scripts-example/scripts/outdated/internal"
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
			want: internal.Options{Min: internal.LevelPatch},
		},
		{
			name: "MinPatch",
			args: []string{"--min=patch"},
			want: internal.Options{Min: internal.LevelPatch},
		},
		{
			name: "MinMinor",
			args: []string{"--min", "minor"},
			want: internal.Options{Min: internal.LevelMinor},
		},
		{
			name: "All",
			args: []string{"--check", "--min=major"},
			want: internal.Options{Min: internal.LevelMajor, Check: true},
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			a := NewMockApp(gomock.NewController(t))
			a.EXPECT().Report(tc.want).Return(errApp)

			err := kongRun(t, a, tc.args...)
			t.Err(err, errApp)
		})
	}
}

func TestCLI_Run_InvalidMin(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	a := NewMockApp(gomock.NewController(t))

	err := kongRun(t, a, "--min=other")
	t.Match(err, `--min must be one of "patch","minor","major" but got "other"`)
}

func TestUsageSpec(t *testing.T) {
	t.Parallel()
	kongusage.CheckSpec(t, kong.Must(&internal.CLI{}))
}
