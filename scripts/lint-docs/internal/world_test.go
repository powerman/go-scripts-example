package internal_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/lint-docs/internal"
)

// fakeScriptEnv makes the test binary act as a binary of a Go script,
// which exits with the status given by this variable.
const fakeScriptEnv = "LINT_DOCS_FAKE_SCRIPT_EXIT"

func TestMain(m *testing.M) {
	if status, ok := os.LookupEnv(fakeScriptEnv); ok {
		os.Exit(fakeScript(status))
	}
	m.Run()
}

// fakeScript prints arguments to stdout, one per line, and a message to stderr.
// It returns the exit status.
func fakeScript(status string) int {
	code, err := strconv.Atoi(status)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(os.Stdout, strings.Join(os.Args[1:], "\n"))
	fmt.Fprintln(os.Stderr, "fake script")
	return code
}

// TestRealWorld_RunScript checks arguments, output and exit status of a binary
// by running the test binary as a binary of a Go script.
// Its name has no .exe, like a name of a script on Windows.
//
//nolint:paralleltest // The fake script gets its exit status by an environment variable.
func TestRealWorld_RunScript(tt *testing.T) {
	t := check.T(tt).MustAll()
	exe, err := os.Executable()
	t.Nil(err)
	name := strings.TrimSuffix(filepath.Base(exe), ".exe")
	var stdout, stderr bytes.Buffer
	w := internal.RealWorld{ScriptsBin: filepath.Dir(exe), Stdout: &stdout, Stderr: &stderr}

	t.Setenv(fakeScriptEnv, "0")
	t.Nil(w.RunScript(name, "a b", ""))
	t.Equal(stdout.String(), "a b\n\n")
	t.Equal(stderr.String(), "fake script\n")

	t.Setenv(fakeScriptEnv, "3")
	t.Match(w.RunScript(name), `exit status 3`)

	w.ScriptsBin = ""
	t.Err(w.RunScript(name), internal.ErrNoScriptsBin)
}
