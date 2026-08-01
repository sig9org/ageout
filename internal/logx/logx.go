// Package logx implements ageout's console output conventions: plain
// status/info lines, orange warnings, red errors, and (when enabled) gray,
// timestamped debug tracing of internal steps.
package logx

import (
	"fmt"
	"io"
	"os"
	"time"
)

const (
	colorOrange = "\x1b[38;5;208m"
	colorRed    = "\x1b[31m"
	colorGray   = "\x1b[90m"
	colorReset  = "\x1b[0m"
)

// Logger writes ageout's warning, error, and debug output. Now exists as a
// test seam for deterministic debug timestamps; production callers should
// leave it nil, which falls back to time.Now.
type Logger struct {
	W       io.Writer
	Debug   bool
	NoColor bool
	Now     func() time.Time
}

// New returns a Logger writing to w. Color is disabled when the NO_COLOR
// environment variable is set (see https://no-color.org).
func New(w io.Writer, debug bool) *Logger {
	return &Logger{W: w, Debug: debug, NoColor: os.Getenv("NO_COLOR") != ""}
}

// Warnf writes an orange, non-fatal warning line.
func (l *Logger) Warnf(format string, args ...any) {
	fmt.Fprintln(l.W, l.colorize(colorOrange, fmt.Sprintf(format, args...)))
}

// Errorf writes a red error line.
func (l *Logger) Errorf(format string, args ...any) {
	fmt.Fprintln(l.W, l.colorize(colorRed, fmt.Sprintf(format, args...)))
}

// Debugf writes a timestamped, gray debug line. It is a no-op unless Debug
// is true.
func (l *Logger) Debugf(format string, args ...any) {
	if !l.Debug {
		return
	}
	now := time.Now()
	if l.Now != nil {
		now = l.Now()
	}
	line := fmt.Sprintf("%s [debug] %s", now.Format("2006-01-02 15:04:05.000"), fmt.Sprintf(format, args...))
	fmt.Fprintln(l.W, l.colorize(colorGray, line))
}

func (l *Logger) colorize(code, s string) string {
	if l.NoColor {
		return s
	}
	return code + s + colorReset
}
