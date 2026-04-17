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

func NewCodedError(code string, msg string, err error) error {
	return CodedError{
		code: code,
		msg:  msg,
		err:  err,
	}
}

func (e CodedError) Error() string {
	if e.err == nil {
		return e.msg
	}
	if e.msg == "" {
		return e.err.Error()
	}
	return fmt.Sprintf("%s: %v", e.msg, e.err)
}

func (e CodedError) Unwrap() error {
	return e.err
}

func (e CodedError) Code() string {
	return e.code
}
