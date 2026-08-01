// Package age computes the cutoff time used to decide whether a file has
// aged past a configured threshold.
package age

import "time"

// Duration is a calendar-based age threshold expressed as the individual
// components a user can pass on the command line.
type Duration struct {
	Years   int
	Months  int
	Days    int
	Hours   int
	Minutes int
}

// IsZero reports whether every component is zero, meaning no threshold was
// configured.
func (d Duration) IsZero() bool {
	return d.Years == 0 && d.Months == 0 && d.Days == 0 && d.Hours == 0 && d.Minutes == 0
}

// Cutoff returns the point in time before which a file is considered
// expired, computed relative to now. Years/months/days are applied via
// calendar arithmetic (so e.g. "1 month" correctly crosses variable-length
// months) and hours/minutes are applied as a fixed duration on top.
func (d Duration) Cutoff(now time.Time) time.Time {
	t := now.AddDate(-d.Years, -d.Months, -d.Days)
	t = t.Add(-time.Duration(d.Hours) * time.Hour)
	t = t.Add(-time.Duration(d.Minutes) * time.Minute)
	return t
}
