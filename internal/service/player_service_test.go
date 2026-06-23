package service

import "testing"

func TestIsOfflineMode(t *testing.T) {
	cases := []struct {
		uuid string
		want bool
	}{
		{"dc1be393-0640-47b4-9bad-5b11482e44e6", false}, // version 4 -> online
		{"069a79f4-44e9-3000-a000-000000000000", true},  // version 3 -> offline
		{"069a79f4", false},                             // too short to determine
	}
	for _, c := range cases {
		if got := IsOfflineMode(c.uuid); got != c.want {
			t.Errorf("IsOfflineMode(%q) = %v, want %v", c.uuid, got, c.want)
		}
	}
}
