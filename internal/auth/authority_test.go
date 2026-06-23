package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestAuthorityClientHasPermission(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Internal-Token") != "secret" {
			t.Errorf("missing/wrong X-Internal-Token header")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"permissions": []string{"web.litebans.view.all"},
		})
	}))
	defer server.Close()

	c := NewAuthorityClient(server.URL, "secret", time.Minute)

	ok, err := c.HasPermission(t.Context(), "uuid-1", "web.litebans.view.all")
	if err != nil || !ok {
		t.Fatalf("HasPermission: ok=%v err=%v", ok, err)
	}
	ok, err = c.HasPermission(t.Context(), "uuid-1", "other.permission")
	if err != nil || ok {
		t.Fatalf("HasPermission for missing perm: ok=%v err=%v", ok, err)
	}

	// Second call within TTL should be served from cache, not hit the server again.
	if calls.Load() != 1 {
		t.Fatalf("expected 1 upstream call, got %d", calls.Load())
	}
}

func TestAuthorityClientUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := NewAuthorityClient(server.URL, "secret", time.Minute)
	if _, err := c.HasPermission(t.Context(), "uuid-1", "web.litebans.view.all"); err == nil {
		t.Fatalf("expected error when authority is unavailable")
	}
}
