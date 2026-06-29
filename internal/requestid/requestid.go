// Package requestid provides the cross-service request id helpers: generation,
// context storage and an http.RoundTripper that propagates the id to outgoing
// requests. It is deliberately dependency-free (no other internal packages) so
// that both internal/middleware (which sets the id on inbound requests) and
// clients/* (which need to attach it to outgoing requests) can depend on it
// without creating an import cycle, since internal/middleware also depends on
// internal/auth, which in turn depends on clients/authority.
package requestid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDKey struct{}

// New generates a short random id for a new request chain.
func New() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// IntoContext stores id in ctx for later retrieval (e.g. by outgoing
// HTTP clients via PropagatingTransport) and structured logging.
func IntoContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// FromContext returns the request id stored in ctx, if any.
func FromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey{}).(string)
	return id, ok
}

// PropagatingTransport copies the request id carried in a request's context
// (set by middleware.Observability) onto the X-Request-Id header of every
// outgoing request, so a downstream service's Observability middleware picks
// up the same id and the whole call chain shares one request id in its logs.
type PropagatingTransport struct {
	Base http.RoundTripper
}

func (t PropagatingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if id, ok := FromContext(req.Context()); ok {
		req = req.Clone(req.Context())
		req.Header.Set("X-Request-Id", id)
	}
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
