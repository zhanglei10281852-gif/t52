package ticket

import (
	"errors"
	"fmt"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrStorage    = errors.New("storage error")

	ErrRecordNotFound = errors.New("repository record not found")
)

type Error struct {
	Kind    error
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) Is(target error) bool {
	return target == e.Kind || errors.Is(e.Cause, target)
}

func newError(kind error, message string, cause error) error {
	return &Error{Kind: kind, Message: message, Cause: cause}
}

func PublicMessage(err error) string {
	var serviceErr *Error
	if errors.As(err, &serviceErr) && serviceErr.Message != "" {
		return serviceErr.Message
	}
	return "服务暂时不可用"
}
