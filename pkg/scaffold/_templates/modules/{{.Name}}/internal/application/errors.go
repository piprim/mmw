package application

// ErrorCode represents application-level error codes for the {{.Name}} module.
type ErrorCode int

const (
	ErrorCodeUnknown ErrorCode = iota
	ErrorCodeNotFound
	ErrorCodeAlreadyExists
	ErrorCodeInvalidInput
)

// DomainErrorFor wraps domain errors with application error codes.
func DomainErrorFor(err error) error {
	return err
}
