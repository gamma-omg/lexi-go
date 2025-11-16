package service

import (
	"fmt"
	"runtime/debug"
)

type ServiceError struct {
	Err        error
	Msg        string
	StackTrace string
	StatusCode int
	Env        map[string]string
}

func NewServiceError(err error, statusCode int, msg string, args ...any) *ServiceError {
	return &ServiceError{
		Err:        err,
		Msg:        fmt.Sprintf(msg, args...),
		StatusCode: statusCode,
		StackTrace: string(debug.Stack()),
		Env:        make(map[string]string),
	}
}

func (e *ServiceError) Error() string {
	return e.Msg
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}
