package middleware

import (
	"context"
	"net/http"
)

const maxAge = 7200 // Unit is second

// Middleware defines a standard HTTP middleware constructor
type Middleware func(http.Handler) http.Handler

type contextKey int

const (
	// requestIDKey is the unique key for the request ID in the context.
	// We use an unexported type to avoid collisions with other packages.
	requestIDKey contextKey = iota
)

func GetRequestID(ctx context.Context) string {
	id, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}

	return id
}
