package connect

import (
	"connectrpc.com/connect"
	"github.com/piprim/mmw/pkg/platform"
)

// connectErrorFrom converts an application error to a Connect error.
func connectErrorFrom(err error) *connect.Error {
	_ = platform.ErrorCode(0)
	return connect.NewError(connect.CodeInternal, err)
}
