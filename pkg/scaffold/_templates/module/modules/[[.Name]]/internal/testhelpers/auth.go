package testhelpers

import (
	"context"

	"github.com/google/uuid"

	"[[.PlatformPath]]/pkg/platform/authctx"
)

// TestUser is the user the slice tests act as.
var TestUser = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// AuthedCtx returns a context authenticated as TestUser.
func AuthedCtx() context.Context {
	return authctx.WithUserID(context.Background(), TestUser)
}
