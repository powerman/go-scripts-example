package internal

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
)

const (
	workdir = "/workspace"
	// Caches of Go in the image, which runs as root.
	goCacheDir    = "/root/.cache/go-build"
	goModCacheDir = "/go/pkg/mod"
)

// The same rule as docker uses for names of volumes.
var reVolume = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// containerScript runs in the alpine image, which has no git needed by scripts.
// The repository is mounted read-only, so binaries are built outside of it.
// Binaries run in the container to check that they work on its platform.
const containerScript = `
apk add --no-cache git >/dev/null
git config --global --add safe.directory ` + workdir + `
go -C scripts test -timeout=60s ./...
go -C scripts build -o "$SCRIPTS_BIN/" ./...
"$SCRIPTS_BIN/lint-docs"
`

// hintQEMU is printed with architectures of platforms whose containers can't run.
// Docker without the containerd image store keeps one platform of a tag,
// so alpine may refer to an image of a foreign platform: --pull=always gets the native one.
//
//nolint:dupword // Command mount takes a type and a device of binfmt_misc.
const hintQEMU = `
A container of a foreign platform needs QEMU registered in binfmt_misc on Linux:

    docker run --rm --privileged tonistiigi/binfmt --install %s

Rootless docker needs its own binfmt_misc mounted first, by a container kept running:

    docker run -d --pull=always --privileged --name binfmt alpine sh -c \
        'mount -t binfmt_misc binfmt_misc /proc/sys/fs/binfmt_misc && exec sleep infinity'
`

type app struct {
	world  World
	stdout io.Writer
}

// NewApp creates an App which uses world and prints a platform before its test to stdout,
// and a hint how to run containers of a foreign platform when they can't run.
func NewApp(world World, stdout io.Writer) *app {
	return &app{world: world, stdout: stdout}
}

// Test implements [App].
func (a *app) Test(opts Options) error {
	version, err := a.goVersion()
	if err != nil {
		return fmt.Errorf("%s: %w", ModFile, err)
	}

	goCache, err := a.cacheMount(opts.GoCache, goCacheDir)
	if err != nil {
		return fmt.Errorf("Go cache: %w", err)
	}
	goModCache, err := a.cacheMount(opts.GoModCache, goModCacheDir)
	if err != nil {
		return fmt.Errorf("Go module cache: %w", err)
	}

	c := Container{
		Image:   "golang:" + version + "-alpine",
		Workdir: workdir,
		Mounts: []Mount{
			{Source: ".", Target: workdir, Bind: true, ReadOnly: true},
			goCache,
			goModCache,
		},
		Env: []string{
			"SCRIPTS_BIN=/tmp/bin",
			"KONGUSAGE_TEST_SKIP_USAGE=1", // Tool usage isn't in the image.
		},
		Cmd: []string{"sh", "-e", "-c", containerScript},
	}
	// A failure of tests and a failure to start a container of a platform are told apart
	// by a container which does nothing.
	probe := c
	probe.Cmd = []string{"true"}

	var errs []error
	var archs []string // Of platforms whose containers can't run.
	for _, platform := range opts.Platforms {
		_, err = fmt.Fprintln(a.stdout, "==>", platform)
		if err != nil {
			return err
		}
		probe.Platform = platform
		err = a.world.RunContainer(probe)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w: %w", platform, ErrCannotRun, err))
			if arch := arch(platform); !slices.Contains(archs, arch) {
				archs = append(archs, arch)
			}
			continue
		}
		c.Platform = platform
		err = a.world.RunContainer(c)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", platform, err))
		}
	}

	if len(archs) > 0 {
		_, err = fmt.Fprintf(a.stdout, hintQEMU, strings.Join(archs, ","))
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// arch returns an architecture of a platform like linux/arm/v7.
func arch(platform string) string {
	parts := strings.Split(platform, "/")
	if len(parts) < 2 { //nolint:mnd // OS and architecture.
		return platform
	}
	return parts[1]
}

// goVersion returns the Go version to build scripts with, like the go command selects it.
func (a *app) goVersion() (string, error) {
	data, err := a.world.ReadFile(ModFile)
	if err != nil {
		return "", err
	}
	f, err := modfile.Parse(ModFile, data, nil)
	switch {
	case err != nil:
		return "", err
	case f.Toolchain != nil:
		return strings.TrimPrefix(f.Toolchain.Name, "go"), nil
	case f.Go != nil:
		return f.Go.Version, nil
	default:
		return "", ErrNoGoVersion
	}
}

// cacheMount returns a mount of a cache to target.
// A host directory is created in advance,
// otherwise docker would create it owned by root.
func (a *app) cacheMount(cache, target string) (Mount, error) {
	m := Mount{Source: cache, Target: target, Bind: !reVolume.MatchString(cache)}
	if m.Bind {
		err := a.world.MkdirAll(cache)
		if err != nil {
			return Mount{}, err
		}
	}
	return m, nil
}
