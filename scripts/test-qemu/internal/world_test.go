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

	"github.com/powerman/go-scripts-example/scripts/test-qemu/internal"
)

// fakeDockerEnv makes the test binary act as docker,
// which exits with the status given by this variable.
const fakeDockerEnv = "TEST_QEMU_FAKE_DOCKER_EXIT"

func TestMain(m *testing.M) {
	if status, ok := os.LookupEnv(fakeDockerEnv); ok {
		os.Exit(fakeDocker(status))
	}
	m.Run()
}

// fakeDocker prints arguments to stdout, one per line, and a message to stderr.
// It returns the exit status.
func fakeDocker(status string) int {
	code, err := strconv.Atoi(status)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(os.Stdout, strings.Join(os.Args[1:], "\n"))
	fmt.Fprintln(os.Stderr, "fake docker")
	return code
}

// TestRealWorld_RunContainer checks arguments, output and exit status of docker
// by running the test binary as docker.
//
//nolint:paralleltest // The fake docker gets its exit status by an environment variable.
func TestRealWorld_RunContainer(tt *testing.T) {
	t := check.T(tt).MustAll()
	var stdout, stderr bytes.Buffer
	w := internal.RealWorld{Docker: os.Args[0], Stdout: &stdout, Stderr: &stderr}
	dir, err := filepath.Abs("a b")
	t.Nil(err)
	c := internal.Container{
		Image:    "golang:1.27-alpine",
		Platform: "linux/arm/v7",
		Workdir:  "/workspace",
		Mounts: []internal.Mount{
			{Source: "a b", Target: "/workspace", Bind: true, ReadOnly: true},
			{Source: "cache", Target: "/cache"},
		},
		Env: []string{"A=1 2", "B="},
		Cmd: []string{"sh", "-c", "echo 'x y'"},
	}

	t.Setenv(fakeDockerEnv, "0")
	t.Nil(w.RunContainer(c))
	t.Equal(stdout.String(), strings.Join([]string{
		"run", "--rm", "--platform", "linux/arm/v7", "--workdir", "/workspace",
		"--volume", dir + ":/workspace:ro", "--volume", "cache:/cache",
		"--env", "A=1 2", "--env", "B=",
		"golang:1.27-alpine", "sh", "-c", "echo 'x y'",
		"",
	}, "\n"))
	t.Equal(stderr.String(), "fake docker\n")

	t.Setenv(fakeDockerEnv, "3")
	t.Match(w.RunContainer(c), `exit status 3`)
}
