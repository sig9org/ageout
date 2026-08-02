// Package cli implements ageout's command-line interface: flag parsing,
// target resolution, and wiring the scan/age/purge packages together.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/sig9org/ageout/internal/age"
	"github.com/sig9org/ageout/internal/birthtime"
	"github.com/sig9org/ageout/internal/logx"
	"github.com/sig9org/ageout/internal/purge"
	"github.com/sig9org/ageout/internal/scan"
	"github.com/sig9org/ageout/internal/selfupdate"
	"github.com/sig9org/ageout/internal/version"
)

// Config controls a single run. Now and GetFileTime exist so tests can pin
// down behavior that would otherwise depend on the real clock or the
// filesystem's real file timestamps; production callers should leave both
// zero.
type Config struct {
	Args   []string
	Stdout io.Writer
	Stderr io.Writer

	Now         time.Time
	GetFileTime func(path string, info os.FileInfo) time.Time

	// SelfUpdate exists purely as a test seam for -u/-update; production
	// callers (cli.Main) always leave it nil, and Run falls back to
	// selfupdate.Update.
	SelfUpdate func(ctx context.Context, current string) (updated bool, latest string, err error)
}

// Main parses args and runs ageout, writing target files to stdout and
// diagnostics to stderr.
func Main(args []string, stdout, stderr io.Writer) error {
	return Run(Config{Args: args, Stdout: stdout, Stderr: stderr})
}

// Run executes ageout according to cfg.
func Run(cfg Config) error {
	if len(cfg.Args) == 0 {
		printUsage(cfg.Stdout)
		return nil
	}

	fs := flag.NewFlagSet("ageout", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	fs.Usage = func() { printUsage(fs.Output()) }

	var d age.Duration
	fs.IntVar(&d.Years, "y", 0, "years component of the age threshold")
	fs.IntVar(&d.Years, "year", 0, "years component of the age threshold")
	fs.IntVar(&d.Months, "m", 0, "months component of the age threshold")
	fs.IntVar(&d.Months, "month", 0, "months component of the age threshold")
	fs.IntVar(&d.Days, "d", 0, "days component of the age threshold")
	fs.IntVar(&d.Days, "day", 0, "days component of the age threshold")
	fs.IntVar(&d.Hours, "H", 0, "hours component of the age threshold")
	fs.IntVar(&d.Hours, "hour", 0, "hours component of the age threshold")
	fs.IntVar(&d.Minutes, "M", 0, "minutes component of the age threshold")
	fs.IntVar(&d.Minutes, "min", 0, "minutes component of the age threshold")

	var recursive bool
	fs.BoolVar(&recursive, "r", false, "search directories recursively")
	fs.BoolVar(&recursive, "recursive", false, "search directories recursively")

	var created bool
	fs.BoolVar(&created, "c", false, "use file creation time instead of last modified time")
	fs.BoolVar(&created, "created", false, "use file creation time instead of last modified time")

	var dryRun bool
	fs.BoolVar(&dryRun, "dryrun", false, "print target files without deleting them")

	var silent bool
	fs.BoolVar(&silent, "silent", false, "suppress printing of every scanned file's status and elapsed age")

	var showVersion bool
	fs.BoolVar(&showVersion, "v", false, "print the version number and exit")
	fs.BoolVar(&showVersion, "version", false, "print the version number and exit")

	var showHelp bool
	fs.BoolVar(&showHelp, "h", false, "show this help message and exit")
	fs.BoolVar(&showHelp, "help", false, "show this help message and exit")

	var doUpdate bool
	fs.BoolVar(&doUpdate, "update", false, "update ageout to the latest release and exit")

	var debug bool
	fs.BoolVar(&debug, "debug", false, "print timestamped debug tracing of ageout's internal steps to stdout")

	if err := fs.Parse(cfg.Args); err != nil {
		return err
	}

	logger := logx.New(cfg.Stdout, debug)
	logger.Debugf("parsed args: %v", cfg.Args)

	if showHelp {
		printUsage(cfg.Stdout)
		return nil
	}
	if showVersion {
		fmt.Fprintln(cfg.Stdout, versionLine())
		return nil
	}
	if doUpdate {
		logger.Debugf("starting self-update, current version=%s", version.Version)
		update := cfg.SelfUpdate
		if update == nil {
			update = selfupdate.Update
		}
		updated, latest, err := update(context.Background(), version.Version)
		if err != nil {
			return fmt.Errorf("self-update failed: %w", err)
		}
		if updated {
			logger.Debugf("self-update applied: %s -> %s", version.Version, latest)
			fmt.Fprintf(cfg.Stdout, "ageout updated %s -> %s\n", version.Version, latest)
		} else {
			logger.Debugf("self-update: already at latest version %s", latest)
			fmt.Fprintf(cfg.Stdout, "ageout %s is already the latest version\n", version.Version)
		}
		return nil
	}

	if d.IsZero() {
		return errors.New("at least one of -y/-year, -m/-month, -d/-day, -H/-hour, -M/-min must be specified")
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args()[1:])
	}

	target := "."
	if fs.NArg() == 1 {
		target = fs.Arg(0)
	}

	root, pattern, err := resolveTarget(target, fs.NArg() > 0)
	if err != nil {
		return err
	}
	logger.Debugf("resolved target: root=%s pattern=%v recursive=%v", root, pattern, recursive)

	files, err := scan.Files(root, scan.Options{Recursive: recursive, Pattern: pattern})
	if err != nil {
		return err
	}
	logger.Debugf("scan found %d candidate file(s)", len(files))

	now := cfg.Now
	if now.IsZero() {
		now = time.Now()
	}
	cutoff := d.Cutoff(now)
	logger.Debugf("cutoff=%s now=%s dryrun=%v", cutoff.Format(time.RFC3339), now.Format(time.RFC3339), dryRun)

	out := io.Writer(cfg.Stdout)
	if silent {
		out = io.Discard
	}

	getFileTime := cfg.GetFileTime
	if getFileTime == nil {
		if created {
			logger.Debugf("using file creation time as the age criterion")
			getFileTime = birthtime.Get
		} else {
			logger.Debugf("using last modified time as the age criterion")
			getFileTime = purge.ModTime
		}
	}

	p := &purge.Purger{GetFileTime: getFileTime, Debugf: logger.Debugf}
	_, err = p.Run(out, files, now, cutoff, dryRun)
	return err
}

// resolveTarget decides whether target names a directory to scan or a
// regular expression to match file names against (searched from the
// current directory). explicit indicates whether the user actually passed
// a target argument, as opposed to it defaulting to ".".
func resolveTarget(target string, explicit bool) (root string, pattern *regexp.Regexp, err error) {
	if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
		return target, nil, nil
	}
	if !explicit {
		return ".", nil, nil
	}

	re, err := regexp.Compile(target)
	if err != nil {
		return "", nil, fmt.Errorf("%q is not a directory and is not a valid regexp: %w", target, err)
	}
	return ".", re, nil
}

// versionLine formats the tool name, released version, and the VCS commit
// it was built from, e.g. "ageout v1.2.3 (commit abc1234def0)".
func versionLine() string {
	return fmt.Sprintf("ageout %s (commit %s)", version.Version, version.Commit())
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, versionLine())
	fmt.Fprintln(w)
	fmt.Fprintln(w, "usage: ageout [flags] [directory|regexp]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "With no target, the current directory is scanned. If the target is an")
	fmt.Fprintln(w, "existing directory, its files are scanned instead. Otherwise the target")
	fmt.Fprintln(w, "is treated as a regular expression matched against file names in the")
	fmt.Fprintln(w, "current directory.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Age threshold (at least one required):")
	fmt.Fprintln(w, "  -y, -year int      years component of the age threshold")
	fmt.Fprintln(w, "  -m, -month int     months component of the age threshold")
	fmt.Fprintln(w, "  -d, -day int       days component of the age threshold")
	fmt.Fprintln(w, "  -H, -hour int      hours component of the age threshold")
	fmt.Fprintln(w, "  -M, -min int       minutes component of the age threshold")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Other flags:")
	fmt.Fprintln(w, "  -r, -recursive     search directories recursively")
	fmt.Fprintln(w, "  -c, -created       use file creation time instead of last modified time")
	fmt.Fprintln(w, "      -dryrun        report target files without deleting them")
	fmt.Fprintln(w, "      -silent        suppress printing of every scanned file's status and elapsed age")
	fmt.Fprintln(w, "      -debug         print timestamped debug tracing of ageout's internal steps to stdout")
	fmt.Fprintln(w, "      -update        update ageout to the latest release and exit")
	fmt.Fprintln(w, "  -v, -version       print the version number and exit")
	fmt.Fprintln(w, "  -h, -help          show this help message and exit")
}
