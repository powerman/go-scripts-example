# go-scripts-example

[![License MIT](https://img.shields.io/badge/license-MIT-royalblue.svg)](LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/powerman/go-scripts-example?filename=scripts%2Fgo.mod&color=blue)](https://go.dev/)
[![Test](https://img.shields.io/github/actions/workflow/status/powerman/go-scripts-example/test.yml?label=test)](https://github.com/powerman/go-scripts-example/actions/workflows/test.yml)

![Linux | amd64 arm64 armv7 ppc64le s390x riscv64](https://img.shields.io/badge/Linux-amd64%20arm64%20armv7%20ppc64le%20s390x%20riscv64-royalblue)
![macOS | amd64 arm64](https://img.shields.io/badge/macOS-amd64%20arm64-royalblue)
![Windows | amd64 arm64](https://img.shields.io/badge/Windows-amd64%20arm64-royalblue)

An example of scripts of a project written in Go and run as [mise](https://mise.jdx.dev/) tasks.
It's meant for projects in any language, not only in Go:
copy the parts you need and adapt them.

The scripts here are real tasks of this repository:
[toc](scripts/toc/) maintains the table of contents below,
[lint:docs](scripts/lint-docs/) checks it and links in markdown files
by `toc --check` and [lint-links](scripts/lint-links/) on CI,
on Linux, macOS and Windows,
[test-qemu](scripts/test-qemu/) tests scripts in docker on platforms emulated by QEMU,
[outdated](scripts/outdated/) shows what `mise run bump` would upgrade,
and [wait-ci](scripts/wait-ci/) waits for CI runs of a pushed commit.

The approach itself is only the task layer:
Go programs live in a separate module in [scripts/](scripts/),
root mise tasks build them on demand into `$SCRIPTS_BIN`,
run the binaries directly
and complete their flags and arguments by a usage spec printed by the program.
A task of a script is a few lines of [mise.toml](mise.toml):

```toml
[tasks.toc]
description = 'Update tables of contents in markdown files'
depends = ['scripts:build']
usage = 'mount run="mise run -q toc --usage-spec"'
raw_args = true
run = '"${SCRIPTS_BIN:?}/toc"'
run_windows = '"%SCRIPTS_BIN%\toc"'
```

Everything about the Go code itself is outside the approach:
its layout, [libraries](#libraries), code generators and linter config.
This example shows one recommended way to write and test it.

It isn't "Go instead of shell":
Go is where a shell script goes when it needs a test,
while linear shell scripts stay in shell.
The verbosity of Go costs little when AI agents write the code,
and the compiler, the linter and tests guard their changes.

<!-- toc -->

- [Why scripts in Go](#why-scripts-in-go)
  - [Where a script lives](#where-a-script-lives)
  - [Why Go](#why-go)
- [What this example shows](#what-this-example-shows)
- [Layout](#layout)
- [Usage](#usage)
  - [Requirements](#requirements)
  - [Run a script](#run-a-script)
  - [Working directory of tasks](#working-directory-of-tasks)
  - [Develop scripts](#develop-scripts)
- [How it works: the task layer](#how-it-works-the-task-layer)
  - [Why mise](#why-mise)
  - [A separate Go module](#a-separate-go-module)
  - [Build on demand](#build-on-demand)
  - [Running binaries](#running-binaries)
  - [Completion](#completion)
  - [Windows](#windows)
- [Go code of this example](#go-code-of-this-example)
  - [Code of a script](#code-of-a-script)
  - [Tests: Fake for World, Mock for App](#tests-fake-for-world-mock-for-app)
  - [Libraries](#libraries)
- [Add Go scripts to your project](#add-go-scripts-to-your-project)
  - [A project not in Go](#a-project-not-in-go)
  - [A project in Go](#a-project-in-go)
- [FAQ](#faq)

<!-- tocstop -->

## Why scripts in Go

Shell is a poor language for programming, so projects should keep logic out of shell scripts.
When a script can't be avoided, it climbs a ladder of places,
and a script which gets complex is rewritten in Go instead of growing in shell.

### Where a script lives

1. **Inline in `mise.toml`** — a trivial script.
   shfmt and shellcheck don't see inline scripts, which is one more reason to keep them trivial.
2. **A shell file in `mise/tasks/`** — a long but linear script:
   mostly runs of external commands, little logic and few variables
   (e.g. bootstrap of a database: create users, grant rights).
   Rewriting such a script in Go would be pointless over-engineering.
3. **A Go program in `scripts/`** — a script which needs a test,
   code shared with another script, or real logic (data processing, branching, error handling).
4. **A script whose language is dictated by the place where it runs** stays in that language:
   e.g. a setup script mounted into a container by docker compose.

The wish to write a test for a shell script or to extract a library from shell scripts
is the signal to rewrite them in Go now.
Linear shell scripts stay where they are:
only scripts which became too hard to write and test are rewritten.

### Why Go

- Go gives clear code and real tests:
  external commands are hidden behind an interface, so a test runs without them.
- Code shared by scripts is an ordinary Go package, not a sourced shell library.
- One toolchain for all non-trivial scripts, with tests built in:
  no test infrastructure for shell (bats, helpers).
- Go is a good fit even for a project which isn't written in Go:
  it's easy to read for anybody, and mise installs it like any other tool.

A Go script has a cost: a few files instead of one, and a build step.
This example shows how to keep the cost low,
so a Go script is as convenient to run as a shell script.

## What this example shows

- Two scripts which process files, [toc](scripts/toc/) and [lint-links](scripts/lint-links/),
  with code shared in [scripts/internal/markdown](scripts/internal/markdown/).
- A simple script which orchestrates external commands, [test-qemu](scripts/test-qemu/):
  it has little logic and a `World` which runs docker.
- A hard script which orchestrates external commands, [wait-ci](scripts/wait-ci/):
  it runs git and gh, polls runs of a commit until they are completed or a timeout expires,
  and tracks changes of their statuses;
  tests of the logic run in a bubble of `testing/synctest`, so they take no real time.
- A script which filters output of external commands by pipes, [outdated](scripts/outdated/):
  it asks go and mise what `mise run bump` would upgrade, without changing anything.
- A script which runs binaries of other scripts, [lint:docs](scripts/lint-docs/):
  it finds them by `$SCRIPTS_BIN`, like tasks do, without a nested `mise run`.
  It's a demo of this only: in a real project such a linear script is an inline task in shell.
- Scripts as mise tasks: a fast rebuild on demand, completion of flags and arguments,
  tasks to build, generate, format, lint, test and cover scripts.
- A layout of a script which makes it testable:
  `World` interface for external commands and files,
  a stateful Fake of `World` for tests of the logic,
  a Mock of `App` for tests of the CLI.
- [Kong](https://github.com/alecthomas/kong) for the CLI
  and [kongusage](https://github.com/powerman/kongusage) for its completion in mise.
- [bitfield/script](https://github.com/bitfield/script) for pipes with filters,
  so an adapter looks almost like a shell pipeline:
  [outdated](scripts/outdated/internal/world.go).
- Scripts used by tasks of the project ([fmt and lint](mise.toml)) and run on CI.
- Scripts run on [CI](.github/workflows/test.yml) on Linux, macOS and Windows:
  by mise on Linux and Windows,
  and by Go alone on other native platforms and on Linux platforms emulated by QEMU.
  Task test-qemu does the same as the QEMU job of CI, but locally.

## Layout

```text
.
├── mise.toml                   Tools, root tasks, tasks which run scripts.
└── scripts/                    A separate Go module.
    ├── go.mod
    ├── tasks.toml              Tasks to develop scripts, included by mise.toml.
    ├── .golangci.yml           Only in a project not in Go.
    ├── .mockery.yml            Fakes generated by mockery.
    ├── .mockery/               Template of fakes.
    ├── internal/
    │   └── markdown/           Code shared by scripts.
    ├── lint-docs/              Program of task lint:docs.
    ├── lint-links/             Program without a task, run by lint-docs.
    ├── outdated/               Program of task outdated.
    ├── test-qemu/              Program of task test-qemu.
    ├── wait-ci/                Program of task wait-ci.
    └── toc/                    Program of task toc.
        ├── main.go             Boilerplate: runs the CLI.
        └── internal/
            ├── app.go          App: the logic, which uses World.
            ├── world.go        RealWorld: adapters of commands and files.
            ├── *_test.go       Tests, generated Fake of World and Mock of App.
            └── …               Boilerplate: port.go, cmd.go, wire.go.
```

## Usage

Install [mise](https://mise.jdx.dev/getting-started.html), then install tools:

```sh
mise install
```

### Requirements

The task layer needs mise 2026.8.11 or newer:
it's the first release which accepts `usage = 'mount run=…'` of a task,
used to complete arguments by the spec printed by the program.
With `lockfile = true`, as in [mise.toml](mise.toml), use 2026.10.19 or newer.

### Run a script

```sh
mise run toc --help
mise run toc --check README.md
mise run lint:docs
mise run test-qemu linux/arm/v7 linux/riscv64
mise run wait-ci --timeout=30m
```

Arguments are passed to the script as is, `--help` and invalid flags included.
Flags and arguments of a task are completed by Tab
when [completion of mise](https://mise.jdx.dev/cli/completion.html) is enabled.

Binaries of scripts are built into `.cache/scripts/`,
but run them by tasks:
a task rebuilds a binary before the run and completes its arguments.

### Working directory of tasks

mise runs a task in its `dir`, by default the root directory of the config,
whatever the directory of the shell is.
So paths in arguments are relative to the root,
while Tab completes them relative to the directory of the shell:

```sh
cd scripts
mise run toc --check ../README.md # Fails: toc looks for ../README.md outside the root.
```

kongusage lists the ways out
in [Path completion](https://github.com/powerman/kongusage#path-completion):
a program resolves paths relative to `MISE_ORIGINAL_CWD` when it's set,
a task runs in the directory of the shell (`dir = '{{cwd}}'`),
or tasks are run only from the root.
Scripts of this example expect to be run from the root.

### Develop scripts

```sh
mise run               # Format, lint and test everything.
mise run generate      # Generate mocks and fakes.
mise run scripts:test  # Test scripts.
mise run scripts:fix   # Apply automatic fixes of go fix and golangci-lint.
mise run scripts:cover # Show test coverage of scripts.
```

## How it works: the task layer

This is the approach itself; the Go code of a script is up to you.

### Why mise

The approach needs a task runner which:

- installs Go and the tools of scripts, in the same versions for everybody and on CI;
- does the work of `make` or `go run`,
  but rebuilds a program only when files of scripts change;
- completes flags and arguments of a task by the spec printed by the program;
- keeps tasks to develop scripts (generate, format, lint, test, cover) next to them.

mise provides all of it and is the most convenient runner known to the author;
this example uses mise.
A script still doesn't depend on mise:
it's an ordinary Go program which can be built by `go build` and run by hand or on CI.

### A separate Go module

`scripts/` is a separate Go module with its own [go.mod](scripts/go.mod):

- dependencies of scripts don't get into `go.mod` of the project;
- the same layout works in projects of every language;
- in a project in Go, `go list ./...` of the root skips the nested module,
  so linters, tests and coverage of the project don't see scripts.

Go commands for scripts run in `scripts/`:
`go run ./scripts/toc` from the root fails, because the root has no `go.mod`.
That's why each task in [scripts/tasks.toml](scripts/tasks.toml) has `dir = 'scripts'`.

`scripts/` has no own `mise.toml`: mise loads configs of parent directories too,
so tools, tasks and settings of both would mix.
Tools of scripts are tools of the root [mise.toml](mise.toml),
and tasks to develop scripts are included by it:

```toml
[task_config]
includes = ['mise/tasks', 'scripts/tasks.toml']
```

`includes` replaces the default directories of file tasks, so `mise/tasks` must be listed too.

### Build on demand

A task in Go is a few lines of `mise.toml`:

```toml
[tasks.toc]
description = 'Update tables of contents in markdown files'
depends = ['scripts:build']
usage = 'mount run="mise run -q toc --usage-spec"'
raw_args = true
run = '"${SCRIPTS_BIN:?}/toc"'
run_windows = '"%SCRIPTS_BIN%\toc"'
```

- `scripts:build` rebuilds binaries of all programs into `$SCRIPTS_BIN`
  only when files of `scripts/` change (by `sources` and `outputs`).
- A task runs the binary, not `go run`:
  `go run` adds a line `exit status N` to the output of a failed program
  and exits with status 1 whatever the status of the program was.
- `raw_args = true` makes mise pass arguments as is instead of parsing them by the spec,
  so `--help` and invalid flags reach Kong.
  mise removes `--` after the task name,
  so an argument `--` needs `mise run toc -- -- …`.
- A binary which only tasks and other binaries run has no task:
  a task is needed only to run a binary by hand.

Ways to start a script differ in cost (on the author's machine, warm caches):

- `mise run toc`: about 90 ms, mostly the start of mise and of two tasks
  (`scripts:build` and `toc`);
- a task with `go run` instead of the binary: about 130 ms, the slowest;
- `go run` alone: about 70 ms, which adds up when a script is run in a loop;
- the binary alone: a few milliseconds, but nothing builds and updates it.

So a script is run by hand by its task,
while inside a task (by another script or in a loop) it's run as the binary.

### Running binaries

[mise.toml](mise.toml) gives every task and binary the directory of binaries:

```toml
[env]
SCRIPTS_BIN = '{{config_root}}/.cache/scripts' # Go scripts built by scripts:build.
```

Tasks and binaries find binaries only by this variable,
so it works from any directory, and the path is written in one place.

- A task which runs a binary depends on `scripts:build` and runs `"${SCRIPTS_BIN:?}/<name>"`.
- A binary runs another binary by `filepath.Join(os.Getenv("SCRIPTS_BIN"), "<name>")`
  ([lint-docs](scripts/lint-docs/internal/world.go)),
  without a build: the task which started it has built all binaries.
- A binary isn't run by a nested `mise run`:
  mise resolves tools and env for each task it runs,
  so a nested `mise run` takes about 90 ms against a few milliseconds of a direct start,
  and mise adds `ERROR task failed` to the output on any non-zero exit status.
- `$SCRIPTS_BIN` isn't added to `PATH`:
  binaries have short common names and would shadow commands of the system
  in tasks and in the shell activated by mise.

A shell file task (e.g. `mise/tasks/check-release-docs`) runs a binary the same way,
so a linear shell script can leave its hard part to a Go script:

```sh
#!/usr/bin/env bash
#MISE description="Check links in documentation of a release"
#MISE depends=["scripts:build"]
set -euo pipefail
"${SCRIPTS_BIN:?}/lint-links" README.md CHANGELOG.md
```

### Completion

Each program has a flag `--usage-spec`
which prints [usage spec](https://usage.jdx.dev/) of its Kong CLI,
generated by [kongusage](https://github.com/powerman/kongusage):

```go
type CLI struct {
	// …
	UsageSpec kongusage.Flag
}
```

`usage = 'mount run="…"'` of a task makes mise run the program to complete its arguments.
mise does it only on Tab, so the spec is never stale,
and a broken spec breaks completion of only this task.
A test checks the spec by `kongusage.CheckSpec`,
and `scripts:lint:usage` lints specs of all binaries.

### Windows

Go scripts work on Windows, but mise tasks around them need care:

- mise runs inline scripts by `cmd /c` on Windows, which doesn't expand `$SCRIPTS_BIN`,
  so a task which runs a binary has `run_windows = '"%SCRIPTS_BIN%\toc"'`.
- Tasks needed to run scripts (`scripts:build`, `scripts:test`)
  work in cmd too, with `run_windows` where they use sh:
  `%SCRIPTS_BIN%` instead of `$SCRIPTS_BIN`, PowerShell instead of a loop.
  Other tasks to develop scripts need POSIX sh (e.g. Git Bash).
- Some tools of `mise.toml` may be unavailable on Windows,
  so [CI](.github/workflows/test.yml) installs only tools of scripts
  (`install_args` of `jdx/mise-action`)
  and sets `MISE_AUTO_INSTALL=false`, so `mise run` doesn't try to install others.

## Go code of this example

This is one recommended way to write and test the Go code of a script.

### Code of a script

A script follows a flat variant of hexagonal architecture
and has only two concepts:

- **`App`** with the logic:
  [app.go](scripts/toc/internal/app.go) implements it
  using `World` (external commands and files the logic needs).
  It's the only place with logic, and it doesn't touch the OS.
- **`RealWorld`** with adapters of the outside world:
  [world.go](scripts/toc/internal/world.go) implements `World` by real commands and files,
  thin adapters without logic.

Everything else is mechanical boilerplate, the same in every script:
the CLI, and the glue which lets tests replace either concept.

- [port.go](scripts/toc/internal/port.go) declares interfaces `App` and `World`,
  errors and types.
- [wire.go](scripts/toc/internal/wire.go) creates `App` with `RealWorld` and stdout.
- [cmd.go](scripts/toc/internal/cmd.go) is a Kong CLI which converts flags to a call of `App`.
  The Kong struct is the only spec of the CLI.
- [main.go](scripts/toc/main.go) parses the CLI and runs it.
  `Run` returns errors, and `ctx.FatalIfErrorf` exits with the exit status of a failed command,
  so the status reaches mise.
- The Fake of `World` and the Mock of `App` for tests are generated, see below.

A single-file Go script without tests (shell rewritten line by line)
gets neither clear code nor tests,
so it's usually a sign that the script should have stayed in shell.

### Tests: Fake for World, Mock for App

- **Logic is tested with a Fake of `World`** ([app_test.go](scripts/toc/internal/app_test.go)).
  The Fake is generated by [mockery](https://github.com/vektra/mockery)
  with the template [fake](https://github.com/powerman/mockery-templates)
  (see [.mockery.yml](scripts/.mockery.yml)).
  A test helper embeds it and keeps the state of the world (here, files of a repository),
  which default hooks read and change.
  A test sets the initial state, runs `App` and checks the final state,
  instead of listing each expected call:
  such a test doesn't break when the logic reads a file once more or in another order.
  A failure is injected for one call by `PushReturn`, calls are checked by `History`.
- **CLI is tested with a Mock of `App`** ([cmd_test.go](scripts/toc/internal/cmd_test.go)),
  generated by [mockgen](https://github.com/uber-go/mock):
  here the exact call of `App` made for given flags is exactly what a test should check.
- **Real `World` is tested by an integration test**
  ([world_test.go](scripts/toc/internal/world_test.go))
  with a real git repository in a temporary directory,
  which checks that commands work on each OS.
  A command which can't run in a test
  (like `docker run` of [test-qemu](scripts/test-qemu/internal/world_test.go))
  is replaced by the test binary, which prints its arguments and exits with a given status:
  the test checks the arguments, the output and the exit status seen by the adapter.

### Libraries

[Kong](https://github.com/alecthomas/kong)
with [kongusage](https://github.com/powerman/kongusage)
and [bitfield/script](https://github.com/bitfield/script) are a matter of taste:
they are the ones this example was worked out with.
Any CLI library will do, if you find a way to print a usage spec for mise completion.

A single command is run by `os/exec` ([toc](scripts/toc/internal/world.go)):
bitfield/script would add a dependency and the pitfalls below without a gain there.
It pays off for pipes with filters ([outdated](scripts/outdated/internal/world.go)).

With bitfield/script:

- start a pipe by `script.NewPipe().WithStderr(os.Stderr)`,
  otherwise stderr of a command goes into the pipe and its filters;
- run a command built from values by `ExecCommand(name, args...)`,
  a literal shell command line by `Shell`, never by `Exec` (inconsistent quoting);
- never discard the error of the last stage of a pipe;
- it can't set a working directory of a command, so use options like `git -C <dir>`;
- a command which exits with status 1 when nothing is found (like `git grep`)
  needs a check of `ExitStatus()`, like `|| [ $? -eq 1 ]` in shell.

## Add Go scripts to your project

The approach is also described as [agent skills](https://github.com/powerman/skills):
with `project-scripts` and `mise-toml-style` installed
(and `go-bounded-context-hexagonal` with `go-engineering-policy` for the Go code),
an AI agent can add Go scripts to your project and keep them in this style.

### A project not in Go

This repository is such a project. Copy:

- [scripts/tasks.toml](scripts/tasks.toml), [scripts/.golangci.yml](scripts/.golangci.yml),
  and if you need fakes, [scripts/.mockery.yml](scripts/.mockery.yml)
  and [scripts/.mockery/](scripts/.mockery/);
- `scripts/go.mod` with your module path, e.g. `github.com/you/project/scripts`;
- from [mise.toml](mise.toml): `[task_config]`, tools of Go scripts,
  `env.SCRIPTS_BIN`, `vars.scripts_cover`, tasks `bump` and `cachedir`,
  and `scripts:*` tasks in root tasks `generate`, `fmt`, `lint` and `test`.

### A project in Go

The same, with a few differences:

- Go and tools for Go are tools of the project already.
- There is no `scripts/.golangci.yml`:
  golangci-lint looks for its config from `scripts/` upwards,
  so scripts are linted by the root `.golangci.yml`, with the same rules as the code.
- Coverage of scripts goes to its own file (`vars.scripts_cover`),
  so it counts neither in the total of the project nor in its badge.
- Root tasks of the project (`lint:go`, `test:go`) don't see the nested module,
  while `scripts:*` tasks are kept apart from them by their prefix.

## FAQ

**Is mise required?**
No, but a task runner with the features listed in [Why mise](#why-mise) is.
Without such a runner the scripts still work, but slower and without completion.

A project which can't use mise may list programs by `go get -tool ./<name>` in `scripts/`
and run them from the root by `"$(go -C scripts tool -n <name>)" "$@"`
in a recipe of any runner with a POSIX shell:
`go tool -n` builds the program if needed and prints the path of the cached binary.
The build is always fresh and a start takes about 70 ms, but there is no completion,
and how to wire it into tasks depends on the tools of the project.

**Why not `go tool`?**
It works, and it's simpler:
`go get -tool ./<name>` in `scripts/` lists a program in `scripts/go.mod`,
and `go tool <name>` builds it into the Go build cache if needed and runs it.
There is no build task and no `$SCRIPTS_BIN`, the same command works on Windows,
a binary is never stale, and the exit status passes through, unlike with `go run`.
But a task starts about 40 ms slower,
a binary started by another binary or in a loop costs about 70 ms instead of a few,
and from the root it needs an absolute `GOWORK` or the `go tool -n` form above.
This example prefers speed.

**How is it different from Mage?**
[Mage](https://magefile.org/) is a make-like build tool:
targets are exported Go functions,
their arguments are positional values or `-name=value` flags of a few basic types,
and shell completion covers names of targets but not their arguments.
Here a script is an ordinary CLI which runs without any runner,
while the runner installs Go, rebuilds the program on demand and completes its arguments.
[goyek](https://github.com/goyek/goyek) is a library with the same goal as Mage, run by `go run`.
Pick Mage if it can be the only task runner of the project,
pick this if scripts are separate programs run by tasks of the project's own runner.

**Is a separate Go module required?**
No.
`scripts/go.mod` keeps dependencies of scripts out of the project,
so the same layout works in a project in any language.
If the dependencies don't get in the way,
a project in Go may keep scripts in its own `go.mod`.
The same is true for `scripts/tasks.toml`:
it's how this example keeps development tasks next to scripts
and includes them from the root `mise.toml`;
manage the Go build however you like.

**Why `scripts/`?**
It's the usual place for scripts of a project, so keeping Go scripts here is a reminder
to rewrite complex shell scripts from this directory in Go.
Simple linear shell scripts may stay in shell.
The name is not a requirement: put Go scripts wherever you like.

**Which shell scripts should be rewritten in Go?**
Only those which need a test, shared code, or real logic.
Trivial and linear shell scripts stay in shell,
and a variable, an `if` or a loop doesn't make a script complex.
See [Where a script lives](#where-a-script-lives).

**Why so many small files instead of one?**
The files hold only two concepts:
`App` with the logic and `RealWorld` with adapters of the outside world.
The rest is mechanical boilerplate, the same in every script,
which lets tests replace either concept:
the logic is tested with a Fake of `World`, and the CLI with a Mock of `App`.
It's the simplest form of hexagonal architecture known to the author,
and it makes the boundary between the logic and the OS visible.
Merging the files is possible, but it removes none of the concepts,
so it doesn't make the code simpler.
A single `main.go` with tests of pure functions isn't a simpler alternative:
the logic of a script is mostly decisions made on results of commands,
and they can't be tested without a boundary between the logic and the OS.

**Does the layout pay off for a script which mostly runs commands?**
Yes, see [wait-ci](scripts/wait-ci/).
It runs only a few commands of git and gh,
but in shell it would be a polling loop with time, timeouts and statuses of runs,
which is hard to get right and to test.
In Go the logic uses the real clock, and its tests run in a bubble of `testing/synctest`,
where `time.Sleep` advances a fake clock at once.
The Fake of `World` in [app_test.go](scripts/wait-ci/internal/app_test.go)
has a timeline of runs read by this clock,
so tests of polling and timeouts take no real time.
Even [test-qemu](scripts/test-qemu/), whose logic is small, still makes decisions:
which Go version to use, a volume or a directory for a cache,
whether a platform failed its tests or can't run containers at all.
`World` takes structured values (a `Container`) instead of command-line arguments,
so the Fake doesn't emulate the command-line interface of docker:
a test of the logic checks which containers were run,
and a test of the real `World` checks the arguments by running the test binary as docker.

**Can a Fake differ from the real command?**
Yes, and then tests of the logic pass while the script fails.
Keep the Fake trivial by keeping `World` at the level the logic needs
(e.g. "run this container", not "run docker with these arguments"),
and check the real `World` by an integration test.

**Are both code generators required?**
No.
[mockery](https://github.com/vektra/mockery) generates the Fake of `World`,
and [mockgen](https://github.com/uber-go/mock) generates the Mock of `App`.
Any way to test the logic without external commands will do,
e.g. hand-written fakes or a temporary directory in an integration test.

**Is Go a heavy dependency for a project not in Go?**
No.
mise installs Go like any other tool,
and Go gets into neither the repository nor docker images and releases of the project.
A program is rebuilt only after a change of scripts, and Go builds fast.

**Who maintains Go scripts in a team which doesn't write Go?**
A team which doesn't know Go at all is unlikely to rewrite its scripts in Go.
For others Go is easy to read, and AI agents write such scripts well,
because the layout is the same in every script.
The approach is also described as agent skills,
see [Add Go scripts to your project](#add-go-scripts-to-your-project).
