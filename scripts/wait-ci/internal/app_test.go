package internal_test

import (
	"bytes"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/wait-ci/internal"
)

const sha = "0123456789abcdef0123456789abcdef01234567"

var (
	errGit = errors.New("stub git error")
	errGH  = errors.New("stub gh error")
)

// snapshot of runs of the commit since a moment of the timeline.
type snapshot struct {
	at   time.Duration
	runs []internal.Run
}

// world is a stateful fake of a repository with a single commit HEAD,
// pushed unless NotPushed is set,
// and of CI whose runs of the commit change by a timeline since the creation of the world.
// It must be created inside a bubble of synctest:
// sleeps of the logic advance the fake clock of the bubble, so tests take no real time.
type world struct {
	*FakeWorld

	NotPushed bool
	Timeline  []snapshot // Ordered by time.
	Logs      map[int64]string
	start     time.Time
}

func newWorld(timeline ...snapshot) *world {
	w := &world{
		FakeWorld: NewFakeWorld(),
		Timeline:  timeline,
		Logs:      make(map[int64]string),
		start:     time.Now(),
	}
	w.ResolveCommitFunc.SetDefaultHook(w.resolveCommit)
	w.IsPushedFunc.SetDefaultHook(func(commit string) (bool, error) { return commit == sha && !w.NotPushed, nil })
	w.RunsFunc.SetDefaultHook(w.runs)
	w.FailedLogFunc.SetDefaultHook(w.failedLog)
	return w
}

// Elapsed returns the time passed since the creation of the world.
func (w *world) Elapsed() time.Duration {
	return time.Since(w.start)
}

func (*world) resolveCommit(rev string) (string, error) {
	if rev != "HEAD" {
		return "", errGit
	}
	return sha, nil
}

func (w *world) runs(commit string) ([]internal.Run, error) {
	var runs []internal.Run
	for _, s := range w.Timeline {
		if commit == sha && s.at <= w.Elapsed() {
			runs = s.runs
		}
	}
	return runs, nil
}

func (w *world) failedLog(id int64) (string, error) {
	return w.Logs[id], nil
}

func run(id int64, workflow, status, conclusion string) internal.Run {
	return internal.Run{
		ID:         id,
		Workflow:   workflow,
		Status:     status,
		Conclusion: conclusion,
		URL:        "https://example.com/runs/" + workflow,
	}
}

var opts = internal.Options{
	Commit:  "HEAD",
	Timeout: time.Minute,
	Poll:    10 * time.Second,
	Appear:  30 * time.Second,
}

func TestApp_Wait(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name        string
		timeline    []snapshot
		wantErr     error
		wantOut     string
		wantElapsed time.Duration
	}{
		{
			name: "Succeed",
			timeline: []snapshot{
				{0, []internal.Run{run(1, "Test", "queued", ""), run(2, "Lint", "queued", "")}},
				{10 * time.Second, []internal.Run{run(1, "Test", "in_progress", ""), run(2, "Lint", "queued", "")}},
				{30 * time.Second, []internal.Run{run(1, "Test", "in_progress", ""), run(2, "Lint", "completed", "skipped")}},
				{50 * time.Second, []internal.Run{run(1, "Test", "completed", "success"), run(2, "Lint", "completed", "skipped")}},
			},
			wantOut: "Test #1: queued\n" +
				"Lint #2: queued\n" +
				"Test #1: queued → in_progress\n" +
				"Lint #2: queued → skipped\n" +
				"Test #1: in_progress → success\n",
			wantElapsed: 50 * time.Second,
		},
		{
			name: "AppearLate",
			timeline: []snapshot{
				{25 * time.Second, []internal.Run{run(1, "Test", "in_progress", "")}},
				{40 * time.Second, []internal.Run{run(1, "Test", "completed", "neutral")}},
			},
			wantOut:     "Test #1: in_progress\nTest #1: in_progress → neutral\n",
			wantElapsed: 40 * time.Second,
		},
		{
			name: "NoRuns",
			timeline: []snapshot{
				{35 * time.Second, []internal.Run{run(1, "Test", "queued", "")}},
			},
			wantErr:     internal.ErrNoRuns,
			wantElapsed: 30 * time.Second,
		},
		{
			name: "Timeout",
			timeline: []snapshot{
				{0, []internal.Run{run(1, "Test", "completed", "success"), run(2, "Lint", "waiting", "")}},
			},
			wantErr:     internal.ErrTimeout,
			wantOut:     "Test #1: success\nLint #2: waiting\n",
			wantElapsed: time.Minute,
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			synctest.Test(tt, func(tt *testing.T) {
				t := check.T(tt).MustAll()
				w := newWorld(tc.timeline...)
				var out bytes.Buffer

				err := internal.NewApp(w, &out).Wait(opts)
				t.Err(err, tc.wantErr)
				t.Equal(out.String(), tc.wantOut)
				t.Equal(w.Elapsed(), tc.wantElapsed)
				t.Len(w.FailedLogFunc.History(), 0)
			})
		})
	}
}

func TestApp_Wait_Failed(tt *testing.T) {
	tt.Parallel()
	synctest.Test(tt, testAppWaitFailed)
}

func testAppWaitFailed(tt *testing.T) {
	t := check.T(tt).MustAll()
	w := newWorld(
		snapshot{0, []internal.Run{run(1, "Test", "in_progress", ""), run(2, "Lint", "in_progress", "")}},
		snapshot{10 * time.Second, []internal.Run{
			run(1, "Test", "completed", "failure"),
			run(2, "Lint", "completed", "success"),
			run(3, "Release", "completed", "cancelled"),
		}},
	)
	w.Logs[1] = "test\tstep\tFAIL\n"
	var out bytes.Buffer

	err := internal.NewApp(w, &out).Wait(opts)
	t.Err(err, internal.ErrRunFailed)
	t.Match(err, "^Test #1 failed: failure\nRelease #3 failed: cancelled$")
	t.Equal(out.String(), "Test #1: in_progress\n"+
		"Lint #2: in_progress\n"+
		"Test #1: in_progress → failure\n"+
		"Lint #2: in_progress → success\n"+
		"Release #3: cancelled\n"+
		"\n==> Test #1 failure: https://example.com/runs/Test\n"+
		"test\tstep\tFAIL\n"+
		"\n==> Release #3 cancelled: https://example.com/runs/Release\n")
	history := w.FailedLogFunc.History()
	t.Len(history, 2)
	t.Equal(history[0].ID, int64(1))
	t.Equal(history[1].ID, int64(3))
}

func TestApp_Wait_Errors(tt *testing.T) {
	tt.Parallel()

	timeline := []snapshot{
		{0, []internal.Run{run(1, "Test", "queued", "")}},
		{20 * time.Second, []internal.Run{run(1, "Test", "completed", "failure"), run(2, "Lint", "completed", "failure")}},
	}

	tests := []struct {
		name    string
		prepare func(*world)
		commit  string
		wantErr error
		want    string
	}{
		{
			name:    "ResolveCommit",
			prepare: func(*world) {},
			commit:  "nonexistent",
			wantErr: errGit,
			want:    "^nonexistent: stub git error$",
		},
		{
			name:    "NotPushed",
			prepare: func(w *world) { w.NotPushed = true },
			commit:  "HEAD",
			wantErr: internal.ErrNotPushed,
			want:    "^HEAD: no remote branch contains the commit$",
		},
		{
			name:    "IsPushed",
			prepare: func(w *world) { w.IsPushedFunc.PushReturn(false, errGit) },
			commit:  "HEAD",
			wantErr: errGit,
			want:    "^stub git error$",
		},
		{
			name:    "Runs",
			prepare: func(w *world) { w.RunsFunc.PushHook(w.runs); w.RunsFunc.PushReturn(nil, errGH) },
			commit:  "HEAD",
			wantErr: errGH,
			want:    "^stub gh error$",
		},
		{
			name:    "FailedLog",
			prepare: func(w *world) { w.FailedLogFunc.PushReturn("", errGH) },
			commit:  "HEAD",
			wantErr: internal.ErrRunFailed,
			want:    "^Test #1 failed: failure\nTest #1: stub gh error\nLint #2 failed: failure$",
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			synctest.Test(tt, func(tt *testing.T) {
				t := check.T(tt).MustAll()
				w := newWorld(timeline...)
				tc.prepare(w)

				o := opts
				o.Commit = tc.commit
				err := internal.NewApp(w, &bytes.Buffer{}).Wait(o)
				t.Err(err, tc.wantErr)
				t.Match(err, tc.want)
			})
		})
	}
}
