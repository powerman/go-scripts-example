package internal_test

import (
	"errors"
	"io"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/powerman/check"
	"github.com/powerman/kongusage"
	"go.uber.org/mock/gomock"

	"github.com/powerman/go-scripts-example/scripts/test-qemu/internal"
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
			args: []string{"linux/arm/v7"},
			want: internal.Options{
				Platforms:  []string{"linux/arm/v7"},
				GoCache:    "test-qemu-go-cache",
				GoModCache: "test-qemu-go-mod-cache",
			},
		},
		{
			name: "All",
			args: []string{"--go-cache=/c", "--go-mod-cache", "m/", "linux/s390x", "linux/riscv64"},
			want: internal.Options{
				Platforms:  []string{"linux/s390x", "linux/riscv64"},
				GoCache:    "/c",
				GoModCache: "m/",
			},
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			a := NewMockApp(gomock.NewController(t))
			a.EXPECT().Test(tc.want).Return(errApp)

			err := kongRun(t, a, tc.args...)
			t.Err(err, errApp)
		})
	}
}

func TestCLI_Run_NoPlatform(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	a := NewMockApp(gomock.NewController(t))

	err := kongRun(t, a)
	t.Match(err, `expected "<platform> ..."`)
}

func TestUsageSpec(t *testing.T) {
	t.Parallel()
	kongusage.CheckSpec(t, kong.Must(&internal.CLI{}))
}
