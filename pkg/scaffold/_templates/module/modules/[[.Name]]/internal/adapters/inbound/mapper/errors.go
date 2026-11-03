package mapper

import (
	"errors"

	"[[.PlatformPath]]/pkg/platform"

	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"
)

// DomainErrorFor translates a domain sentinel error into a *platform.DomainError
// carrying the error code of the shared contract, so every caller (Connect,
// in-process) receives a typed error with no knowledge of the domain.
// Non-domain errors are returned unchanged.
//
// Each slice declares its sentinels in <entity>_errors.go as a table onto the
// codes of contracts/proto/<module>/v1/errors.proto, which connect/errors.go
// maps to Connect codes.
func DomainErrorFor(err error) error {
	// mmw:slice-error
	return err
}

// domainErrorFrom returns the DomainError of the first sentinel of codes that
// err wraps; nil for any other error. Sentinels are distinct errors, so the
// map iteration order does not matter.
func domainErrorFrom(err error, codes map[error][[.Name]]v1.[[.Name | pascal]]ErrorCode) *platform.DomainError {
	for sentinel, code := range codes {
		if errors.Is(err, sentinel) {
			return &platform.DomainError{Code: platform.ErrorCode(code), Message: err.Error()}
		}
	}

	return nil
}
