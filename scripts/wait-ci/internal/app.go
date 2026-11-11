package internal

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"time"
)

type app struct {
	world  World
	stdout io.Writer
}

// NewApp creates an App which uses world and prints changes of runs and logs to stdout.
func NewApp(world World, stdout io.Writer) *app {
	return &app{world: world, stdout: stdout}
}

// Wait implements [App].
func (a *app) Wait(opts Options) error {
	commit, err := a.world.ResolveCommit(opts.Commit)
	if err != nil {
		return fmt.Errorf("%s: %w", opts.Commit, err)
	}
	// CI of a commit which isn't pushed never starts,
	// so waiting for its runs would only delay the error.
	isPushed, err := a.world.IsPushed(commit)
	if err != nil {
		return err
	}
	if !isPushed {
		return fmt.Errorf("%s: %w", opts.Commit, ErrNotPushed)
	}

	start := time.Now()
	states := make(map[int64]string) // Last printed state of each run.
	for {
		runs, err := a.world.Runs(commit)
		if err != nil {
			return err
		}
		err = a.printChanges(runs, states)
		if err != nil {
			return err
		}
		if len(runs) > 0 && allCompleted(runs) {
			return a.report(runs)
		}

		elapsed := time.Since(start)
		switch {
		case len(runs) == 0 && elapsed >= opts.Appear:
			return fmt.Errorf("%w in %s", ErrNoRuns, opts.Appear)
		case elapsed >= opts.Timeout:
			return fmt.Errorf("%w in %s", ErrTimeout, opts.Timeout)
		}
		time.Sleep(opts.Poll)
	}
}

// printChanges prints a line for each run whose state differs from the one in states,
// and updates states.
func (a *app) printChanges(runs []Run, states map[int64]string) error {
	for _, r := range runs {
		cur := state(r)
		prev, ok := states[r.ID]
		var err error
		switch {
		case !ok:
			_, err = fmt.Fprintf(a.stdout, "%s #%d: %s\n", r.Workflow, r.ID, cur)
		case prev != cur:
			_, err = fmt.Fprintf(a.stdout, "%s #%d: %s → %s\n", r.Workflow, r.ID, prev, cur)
		}
		if err != nil {
			return err
		}
		states[r.ID] = cur
	}
	return nil
}

// report prints logs of failed runs and returns an error for each of them.
func (a *app) report(runs []Run) error {
	var errs []error
	for _, r := range runs {
		if !failed(r) {
			continue
		}
		errs = append(errs, fmt.Errorf("%s #%d %w: %s", r.Workflow, r.ID, ErrRunFailed, r.Conclusion))
		_, err := fmt.Fprintf(a.stdout, "\n==> %s #%d %s: %s\n", r.Workflow, r.ID, r.Conclusion, r.URL)
		if err != nil {
			return err
		}
		log, err := a.world.FailedLog(r.ID)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s #%d: %w", r.Workflow, r.ID, err))
			continue
		}
		_, err = io.WriteString(a.stdout, log)
		if err != nil {
			return err
		}
	}
	return errors.Join(errs...)
}

// state of a run to print: its status or, when it is completed, its conclusion.
func state(r Run) string {
	if r.Status == StatusCompleted {
		return r.Conclusion
	}
	return r.Status
}

func allCompleted(runs []Run) bool {
	for _, r := range runs {
		if r.Status != StatusCompleted {
			return false
		}
	}
	return true
}

func failed(r Run) bool {
	return r.Status == StatusCompleted &&
		!slices.Contains([]string{ConclusionSuccess, ConclusionSkipped, ConclusionNeutral}, r.Conclusion)
}
