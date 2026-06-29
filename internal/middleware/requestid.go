package middleware

import (
	"context"
	"net/http"
	"xmine/litebans-api/internal/requestid"
)

// NewRequestID generates a short random id for a new request chain.
func NewRequestID() string {
	return requestid.New()
}

// WithRequestID stores id in ctx for later retrieval (e.g. by outgoing
// HTTP clients via PropagatingTransport) and structured logging.
func WithRequestID(ctx context.Context, id string) context.Context {
	return requestid.IntoContext(ctx, id)
}

// RequestIDFromContext returns the request id stored in ctx, if any.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	return requestid.FromContext(ctx)
}

// PropagatingTransport copies the request id carried in a request's context
// (set by Observability) onto the X-Request-Id header of every outgoing
// request, so a downstream service's Observability middleware picks up the
// same id and the whole call chain shares one request id in its logs.
//
// This is an alias of requestid.PropagatingTransport: the implementation
// lives in the dependency-free internal/requestid package so that clients/*
// packages (which internal/auth, and therefore internal/middleware, depend
// on) can use it without an import cycle.
type PropagatingTransport = requestid.PropagatingTransport

var _ http.RoundTripper = PropagatingTransport{}
