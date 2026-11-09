# go-scripts-example

[![License MIT](https://img.shields.io/badge/license-MIT-royalblue.svg)](LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/powerman/go-scripts-example?filename=scripts%2Fgo.mod&color=blue)](https://go.dev/)
[![Test](https://img.shields.io/github/actions/workflow/status/powerman/go-scripts-example/test.yml?label=test)](https://github.com/powerman/go-scripts-example/actions/workflows/test.yml)

![Linux | amd64](https://img.shields.io/badge/Linux-amd64-royalblue)
![macOS | arm64](https://img.shields.io/badge/macOS-arm64-royalblue)
![Windows | amd64](https://img.shields.io/badge/Windows-amd64-royalblue)

An example of scripts of a project written in Go and run as [mise](https://mise.jdx.dev/) tasks.
It's meant for projects in any language, not only in Go:
copy the parts you need and adapt them.

The scripts here are real tasks of this repository:
[toc](scripts/toc/) maintains the table of contents below,
and [lint:links](scripts/lint-links/) checks links in markdown files on CI,
on Linux, macOS and Windows.

<!-- toc -->

- [Why scripts in Go](#why-scripts-in-go)
  - [Where a script lives](#where-a-script-lives)
  - [Why Go](#why-go)
- [What this example shows](#what-this-example-shows)
- [Layout](#layout)
- [Usage](#usage)
  - [Run a script](#run-a-script)
  - [Develop scripts](#develop-scripts)
- [How it works](#how-it-works)
  - [Why mise](#why-mise)
  - [A separate Go module](#a-separate-go-module)
  - [Build on demand](#build-on-demand)
  - [Completion](#completion)
  - [Code of a script](#code-of-a-script)
  - [Tests: Fake for World, Mock for App](#tests-fake-for-world-mock-for-app)
  - [Libraries](#libraries)
- [Windows](#windows)
- [Add Go scripts to your project](#add-go-scripts-to-your-project)
  - [A project not in Go](#a-project-not-in-go)
  - [A project in Go](#a-project-in-go)

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
- One toolchain for all non-trivial scripts:
  no test infrastructure for shell (bats, helpers),
  and no second language with its own environment (venv, `package.json`).
- Go scripts work on Windows without bash.
- Go is a good fit even for a project which isn't written in Go:
  it's easy to read for anybody, and mise installs it like any other tool.

A Go script has a cost: a few files instead of one, and a build step.
This example shows how to keep the cost low,
so a Go script is as convenient to run as a shell script.

## What this example shows

- Two scripts, [toc](scripts/toc/) and [lint:links](scripts/lint-links/),
  with code shared in [scripts/internal/markdown](scripts/internal/markdown/).
- Scripts as mise tasks: a fast rebuild on demand, completion of flags and arguments,
  tasks to build, generate, format, lint, test and cover scripts.
- A layout of a script which makes it testable:
  `World` interface for external commands and files,
  a stateful Fake of `World` for tests of the logic,
  a Mock of `App` for tests of the CLI.
- [Kong](https://github.com/alecthomas/kong) for the CLI
  and [kongusage](https://github.com/powerman/kongusage) for its completion in mise.
- [bitfield/script](https://github.com/bitfield/script) to run commands and read files,
  so an adapter looks almost like a shell pipe.
- Scripts used by tasks of the project ([fmt and lint](mise.toml)) and run on CI.
- Scripts run by mise on [CI](.github/workflows/test.yml) on Linux, macOS and Windows,
  with only tools needed for scripts installed on macOS and Windows.

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
    ├── lint-links/             Program of task lint:links.
    └── toc/                    Program of task toc.
        ├── main.go             Parses the CLI and runs it.
        └── internal/
            ├── port.go         App, World, errors and types.
            ├── app.go          Logic: implements App using World.
            ├── cmd.go          Kong CLI which calls App.
            ├── wire.go         Creates App with real World.
            ├── world.go        Real World: commands and files.
            └── *_test.go       Tests, generated Mock and Fake.
```

## Usage

Install [mise](https://mise.jdx.dev/getting-started.html), then install tools:

```sh
mise install
```

### Run a script

```sh
mise run toc --help
mise run toc --check README.md
mise run lint:links
```

Arguments are passed to the script as is, `--help` and invalid flags included.
Flags and arguments of a task are completed by Tab
when [completion of mise](https://mise.jdx.dev/cli/completion.html) is enabled.

A program is in `.cache/bin/` and may be run directly,
but it may be outdated: run `mise run scripts:build` first.

### Develop scripts

```sh
mise run                  # Format, lint and test everything.
mise run generate         # Generate mocks and fakes.
mise run scripts:test     # Test scripts.
mise run scripts:fix      # Apply automatic fixes of go fix and golangci-lint.
mise run scripts:cover    # Show test coverage of scripts.
mise tasks --hidden       # List all tasks.
```

## How it works

### Why mise

The approach relies on mise, and without it Go scripts would be much less convenient:

- mise installs Go and other tools of scripts, in the same versions for everybody and on CI.
- mise does the work of `make` or `go run`:
  a task rebuilds its program only when files of scripts change.
- mise completes flags and arguments of a task by the spec printed by the program.
- mise keeps tasks to develop scripts (generate, format, lint, test, cover) next to them.

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
run = '.cache/bin/toc'
run_windows = '.cache\bin\toc'
```

- `scripts:build` rebuilds all programs into `.cache/bin/`
  only when files of `scripts/` change (by `sources` and `outputs`),
  so a start of a task takes a few milliseconds.
  `go run` isn't used: its own check of changes takes much longer.
- Another task which runs a program does the same (e.g. `fmt:toc` and `lint:links`):
  `mise run` or `{ task = … }` would add tens of milliseconds to each start.
- `raw_args = true` makes mise pass arguments as is instead of parsing them by the spec,
  so `--help` and invalid flags reach Kong.
  mise removes `--` after the task name,
  so an argument `--` needs `mise run toc -- -- …`.
- A program of task `<name>` is `scripts/<name>/`, with `:` in the name replaced by `-`.
  The binary is named by its directory, so `kong.Name` sets the task name for help.

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
and `scripts:lint:usage` lints specs of all programs.

### Code of a script

A script follows a flat variant of hexagonal architecture,
with layers kept in separate files:

- [port.go](scripts/toc/internal/port.go) declares `App` (what the script does),
  `World` (external commands and files the logic needs), errors and types.
- [app.go](scripts/toc/internal/app.go) implements `App` using `World`.
  It's the only place with logic, and it doesn't touch the OS.
- [cmd.go](scripts/toc/internal/cmd.go) is a Kong CLI which converts flags to a call of `App`.
  The Kong struct is the only spec of the CLI.
- [wire.go](scripts/toc/internal/wire.go) creates `App` with real `World` and stdout.
- [world.go](scripts/toc/internal/world.go) implements `World` by real commands:
  thin adapters without logic.
- [main.go](scripts/toc/main.go) parses the CLI and runs it.
  `Run` returns errors, and `ctx.FatalIfErrorf` exits with the exit status of a failed command,
  so the status reaches mise.

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

### Libraries

[Kong](https://github.com/alecthomas/kong)
with [kongusage](https://github.com/powerman/kongusage)
and [bitfield/script](https://github.com/bitfield/script) are a matter of taste:
they are the ones this example was worked out with.
Any CLI library will do, if you find a way to print a usage spec for mise completion.

With bitfield/script:

- start a pipe by `script.NewPipe().WithStderr(os.Stderr)`,
  otherwise stderr of a command goes into the pipe and its filters;
- run a command built from values by `ExecCommand(name, args...)`,
  a literal shell command line by `Shell`, never by `Exec` (inconsistent quoting);
- never discard the error of the last stage of a pipe;
- it can't set a working directory of a command, so use options like `git -C <dir>`.

## Windows

Go scripts work on Windows, but mise tasks around them need care:

- mise runs inline scripts by `cmd /c` on Windows, which can't run `.cache/bin/toc`,
  so a task which runs a program has `run_windows = '.cache\bin\toc'`.
- Tasks needed to run scripts (`scripts:build`, `scripts:build:clean`, `scripts:test`)
  work in cmd too, with PowerShell in `run_windows` where POSIX sh is needed.
  Other tasks to develop scripts need POSIX sh (e.g. Git Bash).
- Some tools of `mise.toml` may be unavailable on Windows,
  so [CI](.github/workflows/test.yml) installs only tools of scripts
  (`install_args` of `jdx/mise-action`)
  and sets `MISE_AUTO_INSTALL=false`, so `mise run` doesn't try to install others.
- Git returns paths with `/`, while arguments may use `\`:
  [lint:links](scripts/lint-links/internal/app.go) converts them by `filepath.ToSlash`.
- Git on Windows may check out files with CRLF:
  [toc](scripts/toc/internal/app.go) keeps line endings of a file.

## Add Go scripts to your project

### A project not in Go

This repository is such a project. Copy:

- [scripts/tasks.toml](scripts/tasks.toml), [scripts/.golangci.yml](scripts/.golangci.yml),
  and if you need fakes, [scripts/.mockery.yml](scripts/.mockery.yml)
  and [scripts/.mockery/](scripts/.mockery/);
- `scripts/go.mod` with your module path, e.g. `github.com/you/project/scripts`;
- from [mise.toml](mise.toml): `[task_config]`, tools of Go scripts,
  `vars.scripts_cover`, tasks `bump` and `cachedir`,
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
