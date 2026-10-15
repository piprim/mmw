package connect

import (
	"errors"

	"connectrpc.com/connect"
	"[[.PlatformPath]]/pkg/platform"
	commonv1 "[[.ContractsPath]]/go/network/common/v1"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"

	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/inbound/mapper"
)

// domainConnectCodeMap maps the contract's error codes to Connect status codes.
var domainConnectCodeMap = map[platform.ErrorCode]connect.Code{
[[- if .Database]]
	platform.ErrorCode([[.Name]]def.ErrorCodeNotFound):     connect.CodeNotFound,
	platform.ErrorCode([[.Name]]def.ErrorCodeInvalidID):    connect.CodeInvalidArgument,
	platform.ErrorCode([[.Name]]def.ErrorCodeInvalidLabel): connect.CodeInvalidArgument,
[[- else]]
	platform.ErrorCode([[.Name]]def.ErrorCodeGatewayUnavailable): connect.CodeUnavailable,
	platform.ErrorCode([[.Name]]def.ErrorCodeInvalidLabel):       connect.CodeInvalidArgument,
[[- end]]
}

// connectErrorFrom converts an application error into a *connect.Error.
// Domain errors get their Connect code and a typed commonv1.DomainError
// detail; everything else becomes CodeInternal.
func connectErrorFrom(err error) *connect.Error {
	domainErr, ok := errors.AsType[*platform.DomainError](mapper.DomainErrorFor(err))
	if !ok {
		return connect.NewError(connect.CodeInternal, err)
	}

	code, ok := domainConnectCodeMap[domainErr.Code]
	if !ok {
		code = connect.CodeInternal
	}

	cerr := connect.NewError(code, errors.New(domainErr.Message))

	detail, detailErr := connect.NewErrorDetail(&commonv1.DomainError{
		Code:    int32(domainErr.Code),
		Message: domainErr.Message,
	})
	if detailErr == nil {
		cerr.AddDetail(detail)
	}

	return cerr
}
