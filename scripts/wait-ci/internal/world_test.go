package internal_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/wait-ci/internal"
)

// fakeDataEnv makes the test binary act as gh,
// which prints outputs from the directory given by this variable.
// The real gh needs the network and a repository on GitHub.
const fakeDataEnv = "WAIT_CI_FAKE_DATA"

func TestMain(m *testing.M) {
	if dir, ok := os.LookupEnv(fakeDataEnv); ok {
		os.Exit(fakeGH(dir))
	}
	m.Run()
}

// fakeGH prints arguments to stderr and a file of dir chosen by a subcommand to stdout.
// It returns the exit status, which isn't 0 when there is no such file.
func fakeGH(dir string) int {
	args := os.Args[1:]
	fmt.Fprintln(os.Stderr, strings.Join(args, " "))
	name := "runs.json"
	if args[1] == "view" {
		name = "log-failed.txt"
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	os.Stdout.Write(data)
	return 0
}

// realWorld returns a RealWorld whose gh is the test binary with outputs from dir.
func realWorld(t *check.C, dir string, stderr *bytes.Buffer) internal.RealWorld {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", dir))
	t.Nil(err)
	t.Setenv(fakeDataEnv, dir)
	return internal.RealWorld{Git: "git", GH: os.Args[0], Stderr: stderr}
}

// git runs a git command in the current directory and returns its output.
func git(t *check.C, args ...string) string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "git", args...).Output()
	t.Nil(err)
	return strings.TrimSpace(string(out))
}

// newRepo makes a git repository with a commit in a temporary directory,
// changes the current directory to it and returns the full SHA of the commit.
func newRepo(t *check.C) string {
	t.Helper()
	t.Chdir(t.TempDir())
	git(t, "init", "-q")
	git(t, "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgSign=false",
		"commit", "-q", "--allow-empty", "-m", "first")
	return git(t, "log", "-1", "--format=%H")
}

// TestRealWorld_ResolveCommit checks that git resolves revisions to commits on each OS.
//
//nolint:paralleltest // ResolveCommit runs git in the current directory, changed by t.Chdir.
func TestRealWorld_ResolveCommit(tt *testing.T) {
	t := check.T(tt).MustAll()
	head := newRepo(t)
	git(t, "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "tag.gpgSign=false",
		"tag", "-a", "-m", "annotated", "v1")
	var stderr bytes.Buffer
	w := internal.RealWorld{Git: "git", Stderr: &stderr}

	for _, rev := range []string{"HEAD", head[:7], "v1"} {
		commit, err := w.ResolveCommit(rev)
		t.Nil(err, rev)
		t.Equal(commit, head, rev)
	}
	t.Equal(stderr.String(), "")

	_, err := w.ResolveCommit("HEAD^{tree}")
	t.Match(err, "exit status 128")
	t.Contains(stderr.String(), "fatal: ")
}

//nolint:paralleltest // IsPushed runs git in the current directory, changed by t.Chdir.
func TestRealWorld_IsPushed(tt *testing.T) {
	t := check.T(tt).MustAll()
	head := newRepo(t)
	var stderr bytes.Buffer
	w := internal.RealWorld{Git: "git", Stderr: &stderr}

	// A local branch doesn't count, a remote-tracking branch does.
	pushed, err := w.IsPushed(head)
	t.Nil(err)
	t.False(pushed)

	git(t, "update-ref", "refs/remotes/origin/main", head)
	pushed, err = w.IsPushed(head)
	t.Nil(err)
	t.True(pushed)
	t.Equal(stderr.String(), "")

	_, err = w.IsPushed("0123456789abcdef0123456789abcdef01234567")
	t.Match(err, "exit status")
	t.Contains(stderr.String(), "0123456789abcdef0123456789abcdef01234567")
}

//nolint:paralleltest // The fake gh gets its outputs by an environment variable.
func TestRealWorld_Runs(tt *testing.T) {
	t := check.T(tt).MustAll()
	var stderr bytes.Buffer
	const args = "run list --commit 0123abc --limit 100 --json databaseId,workflowName,status,conclusion,url\n"

	runs, err := realWorld(t, "gh", &stderr).Runs("0123abc")
	t.Nil(err)
	t.DeepEqual(runs, []internal.Run{
		{
			ID:       43069843105,
			Workflow: "Release",
			Status:   "in_progress",
			URL:      "https://github.com/powerman/go-scripts-example/actions/runs/43069843105",
		},
		{
			ID:         43069842686,
			Workflow:   "Test",
			Status:     "completed",
			Conclusion: "failure",
			URL:        "https://github.com/powerman/go-scripts-example/actions/runs/43069842686",
		},
	})
	t.Equal(stderr.String(), args)

	stderr.Reset()
	_, err = realWorld(t, "nonexistent", &stderr).Runs("0123abc")
	t.Match(err, "exit status 3")
	t.HasPrefix(stderr.String(), args)

	stderr.Reset()
	_, err = realWorld(t, "invalid", &stderr).Runs("0123abc")
	t.Match(err, "unexpected end of JSON input")
}

//nolint:paralleltest // The fake gh gets its outputs by an environment variable.
func TestRealWorld_FailedLog(tt *testing.T) {
	t := check.T(tt).MustAll()
	var stderr bytes.Buffer
	const args = "run view 43069842686 --log-failed\n"

	log, err := realWorld(t, "gh", &stderr).FailedLog(43069842686)
	t.Nil(err)
	t.HasPrefix(log, "windows\tUNKNOWN STEP\t")
	t.Contains(log, "\nwindows\tUNKNOWN STEP\t2026-11-09T16:09:41.0712345Z ##[error]Process completed")
	t.Equal(stderr.String(), args)

	stderr.Reset()
	_, err = realWorld(t, "nonexistent", &stderr).FailedLog(43069842686)
	t.Match(err, "exit status 3")
	t.HasPrefix(stderr.String(), args)
}
