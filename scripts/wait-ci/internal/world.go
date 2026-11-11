package internal

import (
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// RealWorld implements [World] by real commands.
type RealWorld struct {
	Git    string // Name or path of the git command.
	GH     string // Name or path of the gh command.
	Stderr io.Writer
}

// run is a JSON value printed by gh run list.
//
//nolint:tagliatelle // Names of fields are defined by gh.
type run struct {
	ID         int64  `json:"databaseId"`
	Workflow   string `json:"workflowName"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	URL        string `json:"url"`
}

// ResolveCommit implements [World].
func (w RealWorld) ResolveCommit(rev string) (string, error) {
	out, err := w.output(w.Git, "rev-parse", "--verify", rev+"^{commit}")
	return strings.TrimSpace(out), err
}

// IsPushed implements [World].
func (w RealWorld) IsPushed(commit string) (bool, error) {
	out, err := w.output(w.Git, "branch", "--remotes", "--contains", commit, "--format=%(refname)")
	return strings.TrimSpace(out) != "", err
}

// Runs implements [World].
func (w RealWorld) Runs(commit string) ([]Run, error) {
	// Without --limit it returns only 20 runs.
	out, err := w.output(w.GH, "run", "list", "--commit", commit, "--limit", "100",
		"--json", "databaseId,workflowName,status,conclusion,url")
	if err != nil {
		return nil, err
	}
	var runs []run
	err = json.Unmarshal([]byte(out), &runs)
	if err != nil {
		return nil, err
	}
	res := make([]Run, len(runs))
	for i, r := range runs {
		res[i] = Run(r)
	}
	return res, nil
}

// FailedLog implements [World].
func (w RealWorld) FailedLog(id int64) (string, error) {
	return w.output(w.GH, "run", "view", strconv.FormatInt(id, 10), "--log-failed")
}

// output runs a command and returns its stdout.
func (w RealWorld) output(name string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), name, args...) //nolint:gosec // Running git and gh is the purpose.
	cmd.Stderr = w.Stderr
	out, err := cmd.Output()
	return string(out), err
}
