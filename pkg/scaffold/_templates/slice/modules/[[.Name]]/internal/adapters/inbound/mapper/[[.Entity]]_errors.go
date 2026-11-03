package mapper

import (
	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.ModulePath]]/internal/domain/[[.Entity]]"
)

// [[.Entity]]ErrorCodes lists the error codes of the [[.Entity]] domain, read by DomainErrorFor.
var [[.Entity]]ErrorCodes = map[error][[.Name]]v1.[[.Name | pascal]]ErrorCode{
[[- if ne .Shape "gateway"]]
	[[.Entity]].ErrNotFound: [[.Name]]def.ErrorCodeNotFound,

	[[.Entity]].ErrInvalidID: [[.Name]]def.ErrorCodeInvalidID,
[[- else]]
	[[.Entity]].ErrGatewayUnavailable: [[.Name]]def.ErrorCodeGatewayUnavailable,
[[- end]]

	[[.Entity]].ErrInvalidLabel: [[.Name]]def.ErrorCodeInvalidLabel,
}
