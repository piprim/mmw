// Package contract tests that each slice's inproc contract adapter implements
// its <module>def service contract: mapping and error propagation, with no
// infrastructure.
package contract

import (
	"context"
	"errors"

	"[[.PlatformPath]]/pkg/platform"

	"[[.ModulePath]]/internal/testhelpers"
)

func authedCtx() context.Context { return testhelpers.AuthedCtx() }

// hasDomainCode reports whether err carries the given contract error code.
// The adapter wraps a *platform.DomainError, so match by code.
func hasDomainCode(err error, want platform.ErrorCode) bool {
	de, ok := errors.AsType[*platform.DomainError](err)

	return ok && de.Code == want
}
