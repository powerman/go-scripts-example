package internal_test

import (
	"bytes"
	"errors"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/test-qemu/internal"
)

var (
	errMkdir  = errors.New("stub mkdir error")
	errDocker = errors.New("stub docker error")
	errExec   = errors.New("stub exec format error")
)

// world is a stateful fake of host files and docker.
type world struct {
	*FakeWorld

	Files map[string]string
	Dirs  []string
	// Runs are containers which were run, except probes which do nothing.
	Runs []internal.Container
	// No container of a platform in CantRun runs, and Fail makes tests of a platform fail.
	CantRun map[string]bool
	Fail    map[string]error
}

func newWorld(files map[string]string) *world {
	w := &world{
		FakeWorld: NewFakeWorld(),
		Files:     make(map[string]string),
		CantRun:   make(map[string]bool),
		Fail:      make(map[string]error),
	}
	maps.Copy(w.Files, files)
	w.ReadFileFunc.SetDefaultHook(w.readFile)
	w.MkdirAllFunc.SetDefaultHook(w.mkdirAll)
	w.RunContainerFunc.SetDefaultHook(w.runContainer)
	return w
}

func (w *world) readFile(name string) ([]byte, error) {
	data, ok := w.Files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(data), nil
}

func (w *world) mkdirAll(dir string) error {
	w.Dirs = append(w.Dirs, dir)
	return nil
}

func (w *world) runContainer(c internal.Container) error {
	probe := slices.Equal(c.Cmd, []string{"true"})
	if !probe {
		w.Runs = append(w.Runs, c)
	}
	switch {
	case w.CantRun[c.Platform]:
		return errExec
	case probe:
		return nil
	default:
		return w.Fail[c.Platform]
	}
}

// cmd returns the command of a container and checks its script.
func cmd(t *check.C, c internal.Container) []string {
	t.Helper()
	t.Len(c.Cmd, 4)
	t.DeepEqual(c.Cmd[:3], []string{"sh", "-e", "-c"})
	t.Contains(c.Cmd[3], "go -C scripts test ")
	return c.Cmd
}

// container returns a container of a platform with default caches and a command.
func container(platform, image string, cmd []string) internal.Container {
	return internal.Container{
		Image:    image,
		Platform: platform,
		Workdir:  "/workspace",
		Mounts: []internal.Mount{
			{Source: ".", Target: "/workspace", Bind: true, ReadOnly: true},
			{Source: "gocache", Target: "/root/.cache/go-build"},
			{Source: "gomodcache", Target: "/go/pkg/mod"},
		},
		Env: []string{"SCRIPTS_BIN=/tmp/bin", "KONGUSAGE_TEST_SKIP_USAGE=1"},
		Cmd: cmd,
	}
}

func modFile(lines ...string) map[string]string {
	return map[string]string{internal.ModFile: strings.Join(lines, "\n") + "\n"}
}

var opts = internal.Options{
	Platforms:  []string{"linux/arm/v7"},
	GoCache:    "gocache",
	GoModCache: "gomodcache",
}

func TestApp_Test(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name      string
		files     map[string]string
		opts      internal.Options
		wantImage string
		wantErr   error
	}{
		{
			name:      "Go",
			files:     modFile("module example.com/scripts", "", "go 1.27.3"),
			opts:      opts,
			wantImage: "golang:1.27.3-alpine",
		},
		{
			name:      "Toolchain",
			files:     modFile("module example.com/scripts", "go 1.27", "toolchain go1.27.4"),
			opts:      opts,
			wantImage: "golang:1.27.4-alpine",
		},
		{
			name:    "NoGoVersion",
			files:   modFile("module example.com/scripts"),
			opts:    opts,
			wantErr: internal.ErrNoGoVersion,
		},
		{
			name:    "NoModFile",
			opts:    opts,
			wantErr: fs.ErrNotExist,
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			w := newWorld(tc.files)
			var out bytes.Buffer

			err := internal.NewApp(w, &out).Test(tc.opts)
			t.Err(err, tc.wantErr)
			if tc.wantErr != nil {
				t.Len(w.Runs, 0)
				return
			}
			t.Len(w.Runs, 1)
			t.DeepEqual(w.Runs[0], container("linux/arm/v7", tc.wantImage, cmd(t, w.Runs[0])))
			t.Equal(out.String(), "==> linux/arm/v7\n")
			t.Len(w.Dirs, 0)
		})
	}
}

func TestApp_Test_InvalidModFile(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	w := newWorld(modFile("module"))

	err := internal.NewApp(w, &bytes.Buffer{}).Test(opts)
	t.Match(err, `^scripts/go.mod: scripts/go.mod:1: usage: module module/path`)
	t.Len(w.Runs, 0)
}

func TestApp_Test_Platforms(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	w := newWorld(modFile("module example.com/scripts", "go 1.27.3"))
	w.Fail["linux/ppc64le"] = errDocker
	w.Fail["linux/riscv64"] = errDocker
	var out bytes.Buffer

	o := opts
	o.Platforms = []string{"linux/ppc64le", "linux/s390x", "linux/riscv64"}
	err := internal.NewApp(w, &out).Test(o)
	t.Match(err, "^linux/ppc64le: stub docker error\nlinux/riscv64: stub docker error$")
	platforms := make([]string, 0, len(w.Runs))
	for _, c := range w.Runs {
		platforms = append(platforms, c.Platform)
	}
	t.DeepEqual(platforms, o.Platforms)
	t.Equal(out.String(), "==> linux/ppc64le\n==> linux/s390x\n==> linux/riscv64\n")
}

func TestApp_Test_CantRun(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	w := newWorld(modFile("module example.com/scripts", "go 1.27.3"))
	w.CantRun["linux/arm/v6"] = true
	w.CantRun["linux/arm/v7"] = true
	w.CantRun["linux/arm64"] = true
	var out bytes.Buffer

	o := opts
	o.Platforms = []string{"linux/arm/v6", "linux/s390x", "linux/arm/v7", "linux/arm64"}
	err := internal.NewApp(w, &out).Test(o)
	t.Err(err, internal.ErrCannotRun)
	t.Match(err, "^linux/arm/v6: can't run a container: stub exec format error\n"+
		"linux/arm/v7: can't run a container: stub exec format error\n"+
		"linux/arm64: can't run a container: stub exec format error$")
	t.Len(w.Runs, 1)
	t.Equal(w.Runs[0].Platform, "linux/s390x")
	t.Match(out.String(), "^==> linux/arm/v6\n==> linux/s390x\n==> linux/arm/v7\n==> linux/arm64\n\n")
	t.Contains(out.String(), " --install arm,arm64\n")
}

func TestApp_Test_CacheDirs(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()
	w := newWorld(modFile("module example.com/scripts", "go 1.27.3"))

	o := opts
	o.GoCache = "/tmp/go-build"
	o.GoModCache = ".cache/mod"
	err := internal.NewApp(w, &bytes.Buffer{}).Test(o)
	t.Nil(err)
	t.DeepEqual(w.Dirs, []string{"/tmp/go-build", ".cache/mod"})
	t.Len(w.Runs, 1)
	t.DeepEqual(w.Runs[0].Mounts[1:], []internal.Mount{
		{Source: "/tmp/go-build", Target: "/root/.cache/go-build", Bind: true},
		{Source: ".cache/mod", Target: "/go/pkg/mod", Bind: true},
	})
}

func TestApp_Test_Errors(tt *testing.T) {
	tt.Parallel()

	files := modFile("module example.com/scripts", "go 1.27.3")
	dirs := internal.Options{Platforms: opts.Platforms, GoCache: "/c", GoModCache: "/m"}

	tests := []struct {
		name    string
		prepare func(*world)
		want    string
	}{
		{
			name:    "MkdirGoCache",
			prepare: func(w *world) { w.MkdirAllFunc.PushReturn(errMkdir) },
			want:    "^Go cache: stub mkdir error$",
		},
		{
			name:    "MkdirGoModCache",
			prepare: func(w *world) { w.MkdirAllFunc.PushHook(w.mkdirAll); w.MkdirAllFunc.PushReturn(errMkdir) },
			want:    "^Go module cache: stub mkdir error$",
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			w := newWorld(files)
			tc.prepare(w)

			err := internal.NewApp(w, &bytes.Buffer{}).Test(dirs)
			t.Match(err, tc.want)
			t.Len(w.Runs, 0)
		})
	}
}
