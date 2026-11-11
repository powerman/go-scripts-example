package internal_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/outdated/internal"
)

// fakeDataEnv makes the test binary act as go and mise,
// which print outputs from the directory given by this variable.
// The real commands need the network.
const fakeDataEnv = "OUTDATED_FAKE_DATA"

func TestMain(m *testing.M) {
	if dir, ok := os.LookupEnv(fakeDataEnv); ok {
		os.Exit(fakeCommand(dir))
	}
	m.Run()
}

// fakeCommand prints arguments to stderr and a file of dir chosen by arguments to stdout.
// It returns the exit status, which isn't 0 when there is no such file.
func fakeCommand(dir string) int {
	args := os.Args[1:]
	fmt.Fprintln(os.Stderr, strings.Join(args, " "))
	name := "go-list-m.json"
	switch {
	case slices.Contains(args, "outdated"):
		name = "mise-outdated.json"
	case slices.Contains(args, "-deps"):
		name = "go-list-deps.txt"
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	os.Stdout.Write(data)
	return 0
}

// realWorld returns a RealWorld whose commands are the test binary with outputs from dir.
func realWorld(t *check.C, dir string, stderr *bytes.Buffer) internal.RealWorld {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", dir))
	t.Nil(err)
	t.Setenv(fakeDataEnv, dir)
	return internal.RealWorld{Mise: os.Args[0], Go: os.Args[0], ModDir: "mod dir", Stderr: stderr}
}

const (
	argsMise   = "outdated --local --json\n"
	argsGoDeps = "-C mod dir list -deps -test -f {{with .Module}}{{if not .Main}}{{.Path}}{{end}}{{end}} ./...\n"
	argsGoMods = "-C mod dir list -m -u -json"
)

//nolint:paralleltest // The fake commands get their outputs by an environment variable.
func TestRealWorld_MiseTools(tt *testing.T) {
	t := check.T(tt).MustAll()
	var stderr bytes.Buffer

	tools, err := realWorld(t, "updates", &stderr).MiseTools()
	t.Nil(err)
	t.DeepEqual(tools, []internal.Update{
		{Name: "node", Current: "24.11.0", Latest: "24.12.1"},
		{Name: "usage", Current: "7.1.5", Latest: "7.2.0"},
	})
	t.Equal(stderr.String(), argsMise)

	stderr.Reset()
	tools, err = realWorld(t, "none", &stderr).MiseTools()
	t.Nil(err)
	t.Len(tools, 0)
	t.Equal(stderr.String(), argsMise)

	stderr.Reset()
	_, err = realWorld(t, "no-go-list-m", &stderr).MiseTools()
	t.Match(err, "exit status 3")
	t.HasPrefix(stderr.String(), argsMise)
}

//nolint:paralleltest // The fake commands get their outputs by an environment variable.
func TestRealWorld_GoModules(tt *testing.T) {
	t := check.T(tt).MustAll()
	var stderr bytes.Buffer

	modules, err := realWorld(t, "updates", &stderr).GoModules()
	t.Nil(err)
	t.DeepEqual(modules, []internal.Update{
		{Name: "example.com/a", Current: "v0.0.0-20260101000000-abcdefabcdef", Latest: "v0.1.0"},
		{Name: "golang.org/x/mod", Current: "v0.42.0", Latest: "v0.43.0"},
	})
	// Modules without duplicates, the most frequent first.
	t.Equal(stderr.String(), argsGoDeps+argsGoMods+" golang.org/x/mod example.com/a github.com/alecthomas/kong\n")

	stderr.Reset()
	modules, err = realWorld(t, "none", &stderr).GoModules()
	t.Nil(err)
	t.Len(modules, 0)
	t.Equal(stderr.String(), argsGoDeps+argsGoMods+"\n")

	stderr.Reset()
	_, err = realWorld(t, "no-go-list-m", &stderr).GoModules()
	t.Match(err, "exit status 3")
	t.HasPrefix(stderr.String(), argsGoDeps+argsGoMods+" golang.org/x/mod\n")

	stderr.Reset()
	_, err = realWorld(t, "nonexistent", &stderr).GoModules()
	t.Match(err, "exit status 3")
	t.HasPrefix(stderr.String(), argsGoDeps)
	t.NotContains(stderr.String(), argsGoMods)
}
