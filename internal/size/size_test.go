package size

import "testing"

func TestParse(t *testing.T) {
	cases := map[string]int64{
		"10": 10, "10b": 10, "10B": 10,
		"10k": 10 * 1024, "10K": 10 * 1024,
		"10m": 10 * 1024 * 1024, "10M": 10 * 1024 * 1024,
		"10g": 10 * 1024 * 1024 * 1024, "10G": 10 * 1024 * 1024 * 1024,
	}
	for input, want := range cases {
		got, err := Parse(input)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
}

func TestParseRejectsInvalidSizes(t *testing.T) {
	for _, input := range []string{"", "k", "1.5K", "-1", "1T", "999999999999999999999G"} {
		if _, err := Parse(input); err == nil {
			t.Errorf("Parse(%q) succeeded; want error", input)
		}
	}
}
