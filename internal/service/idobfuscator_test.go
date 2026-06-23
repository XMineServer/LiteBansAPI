package service

import (
	"testing"
	"xmine/litebans-api/internal/domain"
)

func TestIDObfuscatorRoundTrip(t *testing.T) {
	o := NewIDObfuscator(true, "test-secret")
	for _, typ := range []domain.PunishmentType{domain.TypeBan, domain.TypeMute, domain.TypeWarning, domain.TypeKick} {
		for _, id := range []int64{0, 1, 42, 1234, 999999} {
			token := o.Encode(typ, id)
			got, err := o.Decode(typ, token)
			if err != nil {
				t.Fatalf("Decode(%q) error: %v", token, err)
			}
			if got != id {
				t.Errorf("round trip for type=%s id=%d: got %d via token %q", typ, id, got, token)
			}
		}
	}
}

func TestIDObfuscatorRejectsWrongType(t *testing.T) {
	o := NewIDObfuscator(true, "test-secret")
	token := o.Encode(domain.TypeBan, 42)
	if _, err := o.Decode(domain.TypeMute, token); err == nil {
		t.Errorf("expected error decoding a ban token as a mute, got nil")
	}
}

func TestIDObfuscatorNoCollisionsWithinType(t *testing.T) {
	o := NewIDObfuscator(true, "test-secret")
	seen := make(map[string]int64)
	for id := int64(0); id < 2000; id++ {
		token := o.Encode(domain.TypeBan, id)
		if other, exists := seen[token]; exists {
			t.Fatalf("collision: id=%d and id=%d both encode to %q", id, other, token)
		}
		seen[token] = id
	}
}
