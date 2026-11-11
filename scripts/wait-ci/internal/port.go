//go:generate mockgen -source=port.go -destination=mock.port_test.go -package=internal_test -exclude_interfaces=World

// Package internal implements task wait-ci.
package internal

import (
	"errors"
	"time"
)

// Errors.
var (
	ErrNotPushed = errors.New("no remote branch contains the commit")
	ErrNoRuns    = errors.New("no runs")
	ErrTimeout   = errors.New("runs not completed")
	ErrRunFailed = errors.New("failed")
)

// App is the application.
type App interface {
	// Wait polls runs of a commit until all of them are completed,
	// and prints logs of failed jobs of failed runs.
	Wait(opts Options) error
}

// World abstracts commands for testability.
type World interface {
	// ResolveCommit returns the full SHA of a commit given by a Git revision.
	ResolveCommit(rev string) (string, error)
	// IsPushed tells whether a remote-tracking branch contains a commit.
	IsPushed(commit string) (bool, error)
	// Runs returns runs of GitHub Actions workflows for a commit given by its full SHA.
	Runs(commit string) ([]Run, error)
	// FailedLog returns logs of failed jobs of a run.
	FailedLog(id int64) (string, error)
}

// Statuses of a run.
// A run which isn't completed may have other statuses (e.g. queued, waiting).
const (
	StatusCompleted = "completed"
)

// Conclusions of a completed run which don't mean a failure.
const (
	ConclusionSuccess = "success"
	ConclusionSkipped = "skipped"
	ConclusionNeutral = "neutral"
)

// Run of a workflow.
type Run struct {
	ID         int64
	Workflow   string
	Status     string
	Conclusion string // Empty until the run is completed.
	URL        string
}

// Options of [App.Wait].
type Options struct {
	Commit  string        // Git revision.
	Timeout time.Duration // Of waiting for completion of all runs.
	Poll    time.Duration // Interval between checks of runs.
	Appear  time.Duration // Of waiting for the first run.
}
