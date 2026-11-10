package internal_test

import (
	"errors"
	"io"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"
	"github.com/powerman/kongusage"
	"go.uber.org/mock/gomock"

	"github.com/powerman/go-scripts-example/scripts/lint-links/internal"
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
		want []string
	}{
		{name: "Defaults"},
		{name: "Files", args: []string{"a.md", "b c.md"}, want: []string{"a.md", "b c.md"}},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			a := NewMockApp(gomock.NewController(t))
			a.EXPECT().Check(tc.want).Return(errApp)

			err := kongRun(t, a, tc.args...)
			t.Err(err, errApp)
		})
	}
}

func TestUsageSpec(t *testing.T) {
	t.Parallel()
	kongusage.CheckSpec(t, kong.Must(&internal.CLI{}))
}
