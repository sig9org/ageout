<p align="center">
  <img src="https://raw.githubusercontent.com/sig9org/ageout/main/assets/logo.webp" alt="ageout">
</p>

# ageout

ageout is a lightweight, fast command-line tool written in Go that helps you automatically clean up old files based on their age and specific filename patterns. Safely manage log files, temporary archives, or build artifacts with dry-run support and powerful regex filtering.

## Usage

```
ageout [flags] [directory|regexp]
```

With no target, the current directory is scanned. If the target is an existing
directory, its files are scanned instead. Otherwise the target is treated as a
regular expression matched against file names in the current directory.

Running `ageout` with no flags or arguments at all prints the help message.

A file is deleted once it has aged past the threshold built from `-year`,
`-month`, `-day`, `-hour`, and `-min` (at least one must be given).

By default, age is judged by each file's **last modified time**. Pass
`-created` to judge age by the file's **creation time** instead.

| Short | Long | Description |
| --- | --- | --- |
| `-y` | `-year` | Years component of the age threshold |
| `-m` | `-month` | Months component of the age threshold |
| `-d` | `-day` | Days component of the age threshold |
| `-H` | `-hour` | Hours component of the age threshold |
| `-M` | `-min` | Minutes component of the age threshold |
| `-r` | `-recursive` | Search directories recursively |
| `-c` | `-created` | Use file creation time instead of last modified time |
|  | `-dryrun` | Report target files without deleting them |
|  | `-silent` | Suppress printing of every scanned file's status and elapsed age |
|  | `-debug` | Print timestamped debug tracing of ageout's internal steps to stdout |
|  | `-update` | Update ageout to the latest release and exit |
| `-v` | `-version` | Print the version number and exit |
| `-h` | `-help` | Show the help message and exit |

By default, every scanned file is printed — both files that are purge targets
and files that are kept — along with how long ago its age criterion (last
modified time, or creation time with `-created`) was, relative to now:

```
[delete] old.log (elapsed: 3d2h10m)
[skip] recent.log (elapsed: 0d1h5m)
```

The leading tag reflects what happened to the file: `[delete]` for a file
that was actually removed, `[dry-run]` for a file that would have been
removed under `-dryrun`, and `[skip]` for a file that was younger than the
age threshold and therefore kept. Pass `-silent` to suppress this output
entirely. If both `-silent` and `-debug` are given, `-debug` tracing still
prints — `-debug` takes priority over `-silent`.

`-v`/`-version` and `-h`/`-help` both print the tool name, released
version, and the commit it was built from, e.g. `ageout v1.2.3 (commit
abc1234def0)`; `-h`/`-help` additionally prints full usage.

Warnings are printed in orange, errors in red, and debug tracing in gray;
normal status output is left uncolored. Set `NO_COLOR` (see
https://no-color.org) to disable this. Pass `-debug` to additionally print
timestamped tracing of ageout's internal steps (target resolution, scan
results, per-file age evaluation, etc.) to stdout.

Examples:

```sh
# Delete files whose last modified time is older than 30 days
ageout -d 30

# Dry-run: list files whose last modified time is older than 1 year under
# ./logs, recursively
ageout -year 1 -recursive -dryrun ./logs

# Delete *.log files whose last modified time is older than 6 hours in the
# current directory
ageout -H 6 '\.log$'

# Delete files created (not just modified) more than 30 days ago
ageout -d 30 -created

# Delete files older than 30 days without printing anything
ageout -d 30 -silent

# Update ageout itself to the latest GitHub release
ageout -update
```

## Development

```sh
task go-build      # build a "ageout" binary for the current platform (alias: gb)
task go-all-build  # cross-compile release binaries into dist/ (alias: ga)
task go-clean      # empty dist/ (alias: gc)
task go-test       # go vet ./... + go test ./... (alias: gt)
task go-register   # register the latest tagged version on pkg.go.dev (alias: gr)
task cleanup       # remove editor/OS cruft files (alias: c)
```
