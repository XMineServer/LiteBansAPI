package httpapi

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"xmine/litebans-api/internal/domain"
)

var (
	uuid32Re = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
	uuid36Re = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	nameRe   = regexp.MustCompile(`^[A-Za-z0-9_]{1,16}$`)
)

// NormalizeUUID validates a uuid (32 or 36 char form) and returns the canonical 36-char dashed,
// lowercase form, per TOR 2.1.
func NormalizeUUID(raw string) (string, error) {
	switch {
	case uuid36Re.MatchString(raw):
		return strings.ToLower(raw), nil
	case uuid32Re.MatchString(raw):
		lower := strings.ToLower(raw)
		return lower[0:8] + "-" + lower[8:12] + "-" + lower[12:16] + "-" + lower[16:20] + "-" + lower[20:32], nil
	default:
		return "", domain.NewInvalidUUID("uuid must be 32 hex characters or 36 with dashes")
	}
}

// ValidatePlayerName checks a player/moderator name against the allowed alphabet and length (TOR 4.5).
func ValidatePlayerName(name string) error {
	if !nameRe.MatchString(name) {
		return domain.NewInvalidParameter("name must be 1-16 alphanumeric/underscore characters")
	}
	return nil
}

// ParseBoolParam parses an optional boolean query parameter, returning nil if absent.
func ParseBoolParam(r *http.Request, key string) (*bool, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil, domain.NewInvalidParameter("invalid boolean value for " + key)
	}
	return &b, nil
}

// ParseIntParam parses an optional integer query parameter, returning nil if absent.
func ParseIntParam(r *http.Request, key string) (*int, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil, domain.NewInvalidParameter("invalid integer value for " + key)
	}
	return &n, nil
}

// ParseInt64Param parses an optional int64 (millis) query parameter, returning nil if absent.
func ParseInt64Param(r *http.Request, key string) (*int64, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, domain.NewInvalidParameter("invalid integer value for " + key)
	}
	return &n, nil
}

// ParseUUIDParam parses and normalizes an optional uuid query parameter, returning nil if absent.
func ParseUUIDParam(r *http.Request, key string) (*string, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil, nil
	}
	normalized, err := NormalizeUUID(v)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}
