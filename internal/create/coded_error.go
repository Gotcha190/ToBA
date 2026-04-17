package create

import "fmt"

type codeCarrier interface {
	Code() string
}

type CodedError struct {
	code string
	msg  string
	err  error
}

// NewCodedError creates a coded error value that carries a human-readable
// message together with a machine-readable code.
//
// Parameters:
// - code: symbolic error code used by the pipeline logger
// - msg: human-readable error message prefix
// - err: underlying error that caused the failure
//
// Returns:
// - an error implementing Error, Unwrap, and Code
func NewCodedError(code string, msg string, err error) error {
	return CodedError{
		code: code,
		msg:  msg,
		err:  err,
	}
}

// Error formats the coded message together with the wrapped error, when one is
// present.
//
// Parameters:
// - none
//
// Returns:
// - the formatted error string
func (e CodedError) Error() string {
	if e.err == nil {
		return e.msg
	}
	if e.msg == "" {
		return e.err.Error()
	}
	return fmt.Sprintf("%s: %v", e.msg, e.err)
}

// Unwrap returns the wrapped error for errors.Is and errors.As support.
//
// Parameters:
// - none
//
// Returns:
// - the wrapped error, or nil when no underlying error is present
func (e CodedError) Unwrap() error {
	return e.err
}

// Code exposes the machine-readable error code associated with the failure.
//
// Parameters:
// - none
//
// Returns:
// - the coded error identifier used by the pipeline logger
func (e CodedError) Code() string {
	return e.code
}
