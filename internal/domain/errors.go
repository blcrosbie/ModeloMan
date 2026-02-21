package domain

import "fmt"

type ErrorCode string

const (
	CodeInvalidArgument    ErrorCode = "invalid_argument"
	CodeNotFound           ErrorCode = "not_found"
	CodeConflict           ErrorCode = "conflict"
	CodeUnauthenticated    ErrorCode = "unauthenticated"
	CodeFailedPrecondition ErrorCode = "failed_precondition"
	CodeResourceExhausted  ErrorCode = "resource_exhausted"
	CodeInternal           ErrorCode = "internal"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func InvalidArgument(message string) *AppError {
	return &AppError{Code: CodeInvalidArgument, Message: message}
}

func NotFound(message string) *AppError {
	return &AppError{Code: CodeNotFound, Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Code: CodeConflict, Message: message}
}

func Unauthenticated(message string) *AppError {
	return &AppError{Code: CodeUnauthenticated, Message: message}
}

func FailedPrecondition(message string) *AppError {
	return &AppError{Code: CodeFailedPrecondition, Message: message}
}

func ResourceExhausted(message string) *AppError {
	return &AppError{Code: CodeResourceExhausted, Message: message}
}

func Internal(message string, cause error) *AppError {
	return &AppError{Code: CodeInternal, Message: message, Cause: cause}
}

func AsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	typed, ok := err.(*AppError)
	return typed, ok
}
