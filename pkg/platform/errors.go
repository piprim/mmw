package platform

// ErrorCode is the numeric code identifying a domain error.
// Values come from proto-generated enums (e.g. todov1.TodoErrorCode).
type ErrorCode int32

// DomainError is the boundary error type returned by the application layer.
// It implements the error interface and carries a machine-readable Code
// and a human-readable Message.
type DomainError struct {
	Code    ErrorCode
	Message string
}

func (e *DomainError) Error() string { return e.Message }
