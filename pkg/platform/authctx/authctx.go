// Package authctx provides context helpers for propagating the authenticated
// user identity across application-layer boundaries.
//
// It is written by BearerAuthMiddleware (platform/middleware) after a
// successful token validation, and read by any application-layer code that
// needs the current user's identity.
package authctx

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type platformContextKey struct{}

type applicationsContextKey struct{}

// ErrUnauthenticated is returned when the request context carries no userID.
var ErrUnauthenticated = errors.New("unauthenticated")

// WithUserID stores the authenticated userID in the context.
// Called by BearerAuthMiddleware before passing the request to the next handler.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, platformContextKey{}, userID)
}

// UserIDFromContext extracts the authenticated userID from the context.
// Returns ErrUnauthenticated if absent.
func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	v := ctx.Value(platformContextKey{})
	if v == nil {
		return uuid.Nil, ErrUnauthenticated
	}

	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrUnauthenticated
	}

	return id, nil
}

// WithApplications stores the applications the authenticated user may access
// in the context. Called by BearerAuthMiddleware after a successful token
// validation.
func WithApplications(ctx context.Context, applications []string) context.Context {
	return context.WithValue(ctx, applicationsContextKey{}, applications)
}

// ApplicationsFromContext extracts the applications the authenticated user may
// access from the context. It returns nil if absent.
func ApplicationsFromContext(ctx context.Context) []string {
	apps, _ := ctx.Value(applicationsContextKey{}).([]string)

	return apps
}
