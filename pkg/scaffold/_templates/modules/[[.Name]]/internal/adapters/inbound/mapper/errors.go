package mapper

import (
	"errors"

	"[[.PlatformPath]]/pkg/platform"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// DomainErrorFor translates a domain sentinel error into a *platform.DomainError
// carrying the error code of the shared contract, so every caller (Connect,
// in-process) receives a typed error with no knowledge of the domain.
// Non-domain errors are returned unchanged.
//
// A new sentinel needs three registrations: the proto [[.Name | pascal]]ErrorCode
// enum, this switch, and domainConnectCodeMap in connect/errors.go.
func DomainErrorFor(err error) error {
	switch {
[[- if .Database]]
	case errors.Is(err, [[.Entity]].ErrNotFound):
		return &platform.DomainError{
			Code:    platform.ErrorCode([[.Name]]def.ErrorCodeNotFound),
			Message: err.Error(),
		}

	case errors.Is(err, [[.Entity]].ErrInvalidID):
		return &platform.DomainError{
			Code:    platform.ErrorCode([[.Name]]def.ErrorCodeInvalidID),
			Message: err.Error(),
		}
[[- else]]
	case errors.Is(err, [[.Entity]].ErrGatewayUnavailable):
		return &platform.DomainError{
			Code:    platform.ErrorCode([[.Name]]def.ErrorCodeGatewayUnavailable),
			Message: err.Error(),
		}
[[- end]]

	case errors.Is(err, [[.Entity]].ErrInvalidLabel):
		return &platform.DomainError{
			Code:    platform.ErrorCode([[.Name]]def.ErrorCodeInvalidLabel),
			Message: err.Error(),
		}
	}

	return err
}
