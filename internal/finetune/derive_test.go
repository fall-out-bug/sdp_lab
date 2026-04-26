package finetune

import "testing"

func TestDeriveComplexity(t *testing.T) {
	cases := map[string]string{
		"S": "low", "s": "low", "XS": "low",
		"M": "medium",
		"L": "high", "XL": "high",
		"":  "",
		"?": "",
	}
	for in, want := range cases {
		if got := DeriveComplexity(in); got != want {
			t.Errorf("DeriveComplexity(%q)=%q want %q", in, got, want)
		}
	}
}

func TestDeriveRiskFromPriority(t *testing.T) {
	cases := map[string]string{
		"P0": "high", "P1": "high", "0": "high", "1": "high",
		"P2": "low", "P4": "low", "3": "low",
		"": "", "P9": "",
	}
	for in, want := range cases {
		if got := DeriveRiskFromPriority(in); got != want {
			t.Errorf("DeriveRiskFromPriority(%q)=%q want %q", in, got, want)
		}
	}
}

func TestDeriveTaskType(t *testing.T) {
	cases := []struct {
		title, body string
		want        string
	}{
		{"Fix broken regression test", "", "bugfix"},
		{"Add documentation", "readme update", "docs"},
		{"Refactor parser", "extract helper", "refactor"},
		{"Test coverage harness", "", "test"},
		{"Implement new feature flag", "introduce gate", "feature"},
		{"Random title", "no signals", ""},
		// Word-boundary regressions: substrings must NOT match.
		{"Latest changes", "no other signals", ""},                       // "latest" must not hit "test"
		{"Improve debug logging", "structured fields", ""},               // "debug" must not hit "bug"
		{"Manifest schema cleanup", "", "refactor"},                      // "manifest" not "test"; cleanup wins
		{"Fastest path lookup", "", ""},                                  // "fastest" not "test"
		{"Fixture for ratelimiter", "", "test"},                          // "fixture" still hits
	}
	for _, c := range cases {
		if got := DeriveTaskType(c.title, c.body); got != c.want {
			t.Errorf("DeriveTaskType(%q,%q)=%q want %q", c.title, c.body, got, c.want)
		}
	}
}
