# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

ageout is a small Go CLI that deletes files once they've aged past a
threshold (`-year`/`-month`/`-day`/`-hour`/`-min`), optionally restricted to
a directory and/or a filename regexp. It can also update itself in place
from GitHub releases (`-update`). Module path:
`github.com/sig9org/ageout`.

Long flags use a single hyphen (`-year`, not `--year`), per Go's `flag`
package convention — though `flag` itself accepts either spelling, since it
strips one or two leading dashes identically. `printUsage` and the README
intentionally show only the single-hyphen form.

External dependencies: `golang.org/x/sys` (Linux birth-time syscall) and
`github.com/creativeprojects/go-selfupdate` (the `-update`
implementation). The latter supports GitHub/GitLab/Gitea as update sources,
so `go.mod` carries a fair number of indirect dependencies (gitea/gitlab/
go-github clients, oauth2, httpsig, etc.) purely as its transitive closure —
only the GitHub path is actually used here.

## Commands

This project uses [Task](https://taskfile.dev) (`Taskfile.yml`). The local
`Taskfile.yml` itself defines almost no logic — it's a thin wrapper that
`includes:` three shared Taskfiles hosted at
`github.com/sig9org/tasks` (fetched over HTTPS as raw YAML: `_go.yml`,
`_cleanup.yml`, `_time.yml`) and exposes their internal tasks under
project-local names. This means:

- The **first** `task` invocation in a fresh clone needs network access to
  download and cache those remote Taskfiles (cached under `.task/`,
  gitignored). `task cleanup` also force-refreshes that cache
  (`task --download --force --list-all`) in addition to deleting cruft
  files/dirs.
- `project.ini` (read via `dotenv:`) supplies `BINARY_NAME`, `GITHUB_USER`,
  `GITHUB_REPO` — the shared `_go.yml` Taskfile derives `MODULE` (
  `github.com/{{.GITHUB_USER}}/{{.GITHUB_REPO}}`) and `VERSION_PKG` (
  `{{.MODULE}}/internal/version`) from those, so renaming the module or the
  version package requires touching `project.ini`, not `Taskfile.yml`.
- Actual build/test/clean/register logic lives in the remote `_go.yml`, not
  in this repo — read it (`curl -fsS
  https://raw.githubusercontent.com/sig9org/tasks/main/tasks/_go.yml`) if a
  build/test command needs to change, rather than editing local task stubs.

```sh
task go-build     # build a binary for the current platform into dist/ (alias: gb)
task go-all-build # cross-compile release binaries for all platforms into dist/ (alias: ga)
task go-clean     # empty dist/ (alias: gc)
task go-test      # go vet ./... + go test ./... (alias: gt)
task go-register  # fetch proxy.golang.org + pkg.go.dev for the latest git tag, to trigger pkg.go.dev indexing (alias: gr)
task cleanup      # remove .DS_Store, .idea, .vscode, terraform cruft, then refresh the remote task cache (alias: c)
```

`task go-test` is just `go vet ./...` then `go test ./...` — there is no
Taskfile-driven end-to-end/integration check (an earlier version of this
project had one that backdated fixture files and ran a real binary against
them; that logic no longer exists anywhere in the repo or the shared
Taskfile).

For iterating on a single package/test, use plain `go test` directly rather
than Task, e.g.:

```sh
go test ./internal/purge/...
go test ./internal/cli/... -run TestRun_SilentFlag -v
go vet ./...
gofmt -l .
```

`gofmt -l .` is not wired into any task — run it by hand.

`task go-register` requires at least one git tag to exist (it runs `git
describe --tags --abbrev=0` and fails fast via `set -e` if there are none);
it makes real outbound requests to `proxy.golang.org` and `pkg.go.dev`, so
don't run it against a module/tag that isn't actually meant to be published.

Version strings are injected at build time via `-ldflags
"-X .../internal/version.Version=..."`; `go build .` alone produces a
binary reporting `version.Version == "dev"`. A `dev` build can't self-update
(see below) — `-update` requires a semver-parseable version.

`version.Commit()` (`internal/version/version.go`) is separate from
`Version` and needs no ldflags: it reads `runtime/debug.ReadBuildInfo`'s
`vcs.revision`/`vcs.modified` settings, which the Go toolchain stamps into
every binary automatically as long as the build happens inside a git
checkout (true for both `task go-build` and `task go-all-build`, and for a
plain local `go build .`). `-v`/`-version` and `-h`/`-help` both print
`Version` and `Commit()` together via `versionLine()` in `cli.go`.

## Architecture

Execution is a straight-line pipeline, wired together entirely in
`internal/cli.Run` (`internal/cli/cli.go`). `ageout.go` (package `main`) is a
thin shim that calls `cli.Main` and prints any returned error via `logx`.

```
cli.Run:
  flag.FlagSet parses args           → age.Duration{Years,Months,Days,Hours,Minutes}, recursive, created, dryRun, silent, debug
  logx.New(cfg.Stdout, debug)        → Logger used for -debug tracing throughout the rest of Run
  scan.Files(root, opts)             → []string of candidate file paths (regex/recursive filtering)
  age.Duration.Cutoff(now)           → cutoff time.Time (calendar-aware: AddDate for Y/M/D, then hours/mins)
  purge.Purger{GetFileTime,Debugf}.Run(...) → walks files, deletes/reports/skips, prints per file
```

`-update` short-circuits this pipeline entirely: it calls
`selfupdate.Update` (`internal/selfupdate/selfupdate.go`) and exits, without
touching `scan`/`age`/`purge`. `selfupdate.Update` wraps
`github.com/creativeprojects/go-selfupdate`'s `UpdateSelf`, pointed at the
`sig9org/ageout` GitHub repository; it matches release assets by the
`{name}_{goos}_{goarch}` suffix that `task go-all-build` already produces
(version embedded in the filename doesn't interfere — matching is
suffix-based), so no asset-naming changes were needed to support it.

Key design points, each with a reason that matters when touching the code:

- **`cli.Config.Now` / `cli.Config.GetFileTime` / `cli.Config.SelfUpdate`**
  exist purely as test seams — production callers (`cli.Main`) always leave
  them zero/nil, and `Run` falls back to `time.Now()` / the real time-source
  picked by `-created` / `selfupdate.Update`. Don't repurpose these for
  anything user-facing. The `-debug` logger has no equivalent seam for its
  timestamps (`logx.Logger.Now` exists for that, but `cli.Run` doesn't
  expose a way to inject it) — tests that assert on debug output only check
  for the presence/shape of a timestamp, not an exact value.
- **`internal/logx`** implements ageout's console output convention:
  warnings are orange, errors red, debug tracing gray, and normal
  status/info output is left uncolored — all gated by `NO_COLOR` (checked
  once in `logx.New`). `Logger.Debugf` is a no-op unless `Debug` is true and
  otherwise prefixes a millisecond timestamp. `purge.Purger.Debugf` is a
  pluggable field (nil-safe, same pattern as `GetFileTime`) that `cli.Run`
  wires to the same `Logger.Debugf` so per-file tracing shares one output
  stream with the rest of `-debug`'s tracing.
- **Age criterion is pluggable.** `purge.ModTime` (last modified time) is
  the default `GetFileTime`; passing `-created`/`-c` swaps in
  `birthtime.Get` instead. `purge.New(useCreated bool)` is the public
  constructor mirroring this choice for API callers.
- **`internal/birthtime`** resolves file creation time via
  platform-specific build-tagged files (`birthtime_darwin.go` — syscall
  `Stat_t.Birthtimespec`, `birthtime_linux.go` — `unix.Statx` with
  `STATX_BTIME`, `birthtime_windows.go` — `Win32FileAttributeData`,
  `birthtime_other.go` — fallback). All fall back to `info.ModTime()` when
  the platform/filesystem doesn't expose a birth time.
  Gotcha: on filesystems that enforce `birth <= mtime` (observed on at
  least one macOS setup here), backdating mtime with `os.Chtimes` also
  drags birthtime down — so tests can't reliably construct "old mtime,
  recent birth time" fixtures via `Chtimes`. See the comment on
  `TestRun_CreatedFlagKeepsFreshFile` in `internal/cli/cli_test.go` before
  writing a test that assumes otherwise.
- **Output is default-on, not opt-in.** There is no `-verbose` flag.
  `purge.Purger.Run` always writes one line per scanned file — target or
  not — tagged `[delete]`, `[dry-run]`, or `[skip]`, plus an
  `(elapsed: <N>d<N>h<N>m)` computed from `now` (see `formatElapsed` in
  `internal/purge/purge.go`). `-silent` is what suppresses this, by
  swapping the writer for `io.Discard` in `cli.Run` — the printing logic
  itself is unconditional. `-debug` tracing is independent of `-silent`, and
  takes priority over it: `-debug` always writes to `cfg.Stdout` directly,
  so passing both flags together still produces debug tracing with the
  per-file status lines suppressed.
- **`-dryrun`** is the only spelling (no internal hyphen); it was
  deliberately chosen over `-dry-run` — don't reintroduce the hyphenated
  form.
- **Flag name duplication is intentional**: every long flag is registered
  twice under a short and long name via separate `fs.BoolVar`/`fs.IntVar`
  calls sharing one variable, rather than an alias mechanism (`dryrun`,
  `silent`, `debug`, and `update` are the exceptions, having no short form —
  `-s` and `-u` were deliberately removed, so don't reintroduce them).
  `printUsage` in `cli.go` is hand-maintained plain text, not generated —
  keep it in sync with flag definitions by hand when adding/renaming flags.
- **`scan.Files`** is the only place recursion/pattern-matching happens;
  `purge.Purger.Run` has no knowledge of directories, patterns, or
  recursion — it only sees a flat file list and per-file timestamps.
- Package boundaries mirror the pipeline stages 1:1: `age` (cutoff math),
  `birthtime` (per-OS creation time), `scan` (file discovery), `purge`
  (evaluate + delete/report), `cli` (flags + wiring), `logx` (colored
  warning/error/debug output), `version` (ldflags target), `selfupdate`
  (GitHub release update). When adding a flag that affects *which files get
  touched*, it likely belongs in `scan`; if it affects *how age is judged or
  reported*, it belongs in `purge`.
