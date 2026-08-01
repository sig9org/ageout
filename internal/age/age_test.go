package age

import (
	"testing"
	"time"
)

func TestDuration_IsZero(t *testing.T) {
	cases := []struct {
		name string
		d    Duration
		want bool
	}{
		{"all zero", Duration{}, true},
		{"years set", Duration{Years: 1}, false},
		{"months set", Duration{Months: 1}, false},
		{"days set", Duration{Days: 1}, false},
		{"hours set", Duration{Hours: 1}, false},
		{"minutes set", Duration{Minutes: 1}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.d.IsZero(); got != c.want {
				t.Errorf("IsZero() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestDuration_Cutoff(t *testing.T) {
	now := time.Date(2026, time.March, 15, 12, 30, 0, 0, time.UTC)

	cases := []struct {
		name string
		d    Duration
		want time.Time
	}{
		{
			name: "one year",
			d:    Duration{Years: 1},
			want: time.Date(2025, time.March, 15, 12, 30, 0, 0, time.UTC),
		},
		{
			name: "one month crosses shorter month",
			d:    Duration{Months: 1},
			want: time.Date(2026, time.February, 15, 12, 30, 0, 0, time.UTC),
		},
		{
			name: "days",
			d:    Duration{Days: 10},
			want: time.Date(2026, time.March, 5, 12, 30, 0, 0, time.UTC),
		},
		{
			name: "hours and minutes",
			d:    Duration{Hours: 2, Minutes: 45},
			want: time.Date(2026, time.March, 15, 9, 45, 0, 0, time.UTC),
		},
		{
			name: "combined components",
			d:    Duration{Years: 1, Months: 2, Days: 3, Hours: 4, Minutes: 5},
			want: time.Date(2025, time.January, 12, 8, 25, 0, 0, time.UTC),
		},
		{
			name: "zero duration returns now",
			d:    Duration{},
			want: now,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.d.Cutoff(now)
			if !got.Equal(c.want) {
				t.Errorf("Cutoff() = %v, want %v", got, c.want)
			}
		})
	}
}
