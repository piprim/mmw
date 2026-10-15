package [[.Entity]]

import "errors"

var (
[[- if .Database]]
	ErrInvalidID    = errors.New("invalid [[.Entity]] ID")
	ErrNotFound     = errors.New("[[.Entity]] not found")
	ErrInvalidLabel = errors.New("label must be between 1 and 200 characters")
[[- else]]
	ErrGatewayUnavailable = errors.New("[[.Entity]] gateway unavailable")
	ErrInvalidLabel       = errors.New("label must be between 1 and 200 characters")
[[- end]]
)
