package internal_test

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"
	"github.com/powerman/kongusage"
	"go.uber.org/mock/gomock"

	"github.com/powerman/go-scripts-example/scripts/wait-ci/internal"
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
			want: internal.Options{
				Commit:  "HEAD",
				Timeout: 15 * time.Minute,
				Poll:    10 * time.Second,
				Appear:  time.Minute,
			},
		},
		{
			name: "All",
			args: []string{"--timeout=1h", "--poll", "1m30s", "--appear=5s", "v1.0.0"},
			want: internal.Options{
				Commit:  "v1.0.0",
				Timeout: time.Hour,
				Poll:    90 * time.Second,
				Appear:  5 * time.Second,
			},
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			a := NewMockApp(gomock.NewController(t))
			a.EXPECT().Wait(tc.want).Return(errApp)

			err := kongRun(t, a, tc.args...)
			t.Err(err, errApp)
		})
	}
}

func TestCLI_Run_InvalidDuration(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	a := NewMockApp(gomock.NewController(t))

	err := kongRun(t, a, "--poll=10")
	t.Match(err, `--poll: .*missing unit`)
}

func TestUsageSpec(t *testing.T) {
	t.Parallel()
	kongusage.CheckSpec(t, kong.Must(&internal.CLI{}))
}
