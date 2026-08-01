package logx

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWarnf_WrapsInOrange(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{W: &buf}
	l.Warnf("careful: %s", "thing")

	got := buf.String()
	if !strings.Contains(got, colorOrange) || !strings.Contains(got, colorReset) {
		t.Errorf("Warnf output = %q, want it wrapped in orange/reset codes", got)
	}
	if !strings.Contains(got, "careful: thing") {
		t.Errorf("Warnf output = %q, want it to contain the formatted message", got)
	}
}

func TestErrorf_WrapsInRed(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{W: &buf}
	l.Errorf("boom: %s", "oops")

	got := buf.String()
	if !strings.Contains(got, colorRed) || !strings.Contains(got, colorReset) {
		t.Errorf("Errorf output = %q, want it wrapped in red/reset codes", got)
	}
	if !strings.Contains(got, "boom: oops") {
		t.Errorf("Errorf output = %q, want it to contain the formatted message", got)
	}
}

func TestNoColor_SuppressesEscapeCodes(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{W: &buf, NoColor: true}
	l.Warnf("plain warning")

	got := buf.String()
	if strings.Contains(got, colorOrange) || strings.Contains(got, colorReset) {
		t.Errorf("Warnf output = %q, want no escape codes when NoColor is set", got)
	}
	want := "plain warning\n"
	if got != want {
		t.Errorf("Warnf output = %q, want %q", got, want)
	}
}

func TestDebugf_DisabledByDefault(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{W: &buf}
	l.Debugf("should not appear")

	if buf.Len() != 0 {
		t.Errorf("Debugf output = %q, want empty when Debug is false", buf.String())
	}
}

func TestDebugf_IncludesTimestampWhenEnabled(t *testing.T) {
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var buf bytes.Buffer
	l := &Logger{W: &buf, Debug: true, Now: func() time.Time { return fixed }}
	l.Debugf("hello %s", "world")

	got := buf.String()
	if !strings.Contains(got, "2026-01-02 03:04:05.000 [debug] ") {
		t.Errorf("Debugf output = %q, want it to contain the timestamp and [debug] tag", got)
	}
	if !strings.Contains(got, "hello world") {
		t.Errorf("Debugf output = %q, want it to contain the formatted message", got)
	}
}

func TestDebugf_WrapsInGray(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{W: &buf, Debug: true}
	l.Debugf("tracing detail")

	got := buf.String()
	if !strings.Contains(got, colorGray) || !strings.Contains(got, colorReset) {
		t.Errorf("Debugf output = %q, want it wrapped in gray/reset codes", got)
	}
}

func TestDebugf_NoColorSuppressesEscapeCodes(t *testing.T) {
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var buf bytes.Buffer
	l := &Logger{W: &buf, Debug: true, NoColor: true, Now: func() time.Time { return fixed }}
	l.Debugf("hello %s", "world")

	got := buf.String()
	if strings.Contains(got, colorGray) || strings.Contains(got, colorReset) {
		t.Errorf("Debugf output = %q, want no escape codes when NoColor is set", got)
	}
	want := "2026-01-02 03:04:05.000 [debug] hello world\n"
	if got != want {
		t.Errorf("Debugf output = %q, want %q", got, want)
	}
}

func TestNew_RespectsNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	l := New(&bytes.Buffer{}, false)
	if !l.NoColor {
		t.Error("New() with NO_COLOR set should produce a Logger with NoColor = true")
	}
}

func TestNew_ColorEnabledByDefault(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	l := New(&bytes.Buffer{}, true)
	if l.NoColor {
		t.Error("New() without NO_COLOR should produce a Logger with NoColor = false")
	}
	if !l.Debug {
		t.Error("New(..., true) should set Debug = true")
	}
}
