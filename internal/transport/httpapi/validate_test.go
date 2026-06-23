package httpapi

import "testing"

func TestNormalizeUUID(t *testing.T) {
	want := "dc1be393-0640-47b4-9bad-5b11482e44e6"
	got, err := NormalizeUUID("DC1BE393064047B49BAD5B11482E44E6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	got, err = NormalizeUUID(want)
	if err != nil || got != want {
		t.Errorf("dashed form: got %q, err %v", got, err)
	}

	if _, err := NormalizeUUID("not-a-uuid"); err == nil {
		t.Errorf("expected error for invalid uuid")
	}
}

func TestValidatePlayerName(t *testing.T) {
	if err := ValidatePlayerName("Some_Player1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := ValidatePlayerName("this_name_is_way_too_long"); err == nil {
		t.Errorf("expected error for too-long name")
	}
	if err := ValidatePlayerName("bad name"); err == nil {
		t.Errorf("expected error for name with space")
	}
}
