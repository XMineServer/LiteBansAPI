package service

import "testing"

func TestResolveOffsetPage(t *testing.T) {
	page, pageSize, err := ResolveOffsetPage(nil, nil, 10, 100)
	if err != nil || page != 1 || pageSize != 10 {
		t.Fatalf("defaults: got page=%d pageSize=%d err=%v", page, pageSize, err)
	}

	big := 500
	_, pageSize, err = ResolveOffsetPage(nil, &big, 10, 100)
	if err != nil || pageSize != 100 {
		t.Fatalf("clamp: got pageSize=%d err=%v", pageSize, err)
	}

	zero := 0
	if _, _, err := ResolveOffsetPage(&zero, nil, 10, 100); err == nil {
		t.Fatalf("expected error for page=0")
	}
}

func TestTotalPages(t *testing.T) {
	cases := []struct {
		total    int64
		pageSize int
		want     int
	}{
		{0, 10, 0},
		{1, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{532, 10, 54},
	}
	for _, c := range cases {
		if got := TotalPages(c.total, c.pageSize); got != c.want {
			t.Errorf("TotalPages(%d, %d) = %d, want %d", c.total, c.pageSize, got, c.want)
		}
	}
}
