package platform

// ErrorCode is the numeric code identifying a domain error.
type ErrorCode int32

// DomainError is a typed error carrying a machine-readable Code and a
// human-readable Message. It implements the error interface.
type DomainError struct {
	Code    ErrorCode
	Message string
}

func (e *DomainError) Error() string { return e.Message }
