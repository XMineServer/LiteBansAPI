package service

import "testing"

func TestCleanReason(t *testing.T) {
	cases := map[string]string{
		"§cCheating §l(X-Ray)": "Cheating (X-Ray)",
		"&aGood &kbye":         "Good bye",
		"Color #FF00AA here":   "Color  here",
		"plain text":           "plain text",
		"line1\nline2":         "line1\nline2",
	}
	for input, want := range cases {
		if got := CleanReason(input); got != want {
			t.Errorf("CleanReason(%q) = %q, want %q", input, got, want)
		}
	}
}
