package main

import (
	"errors"
	"fmt"
)

type ExitError struct {
	Code   int
	Err    error
	Silent bool
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee ExitError
	if errors.As(err, &ee) {
		if ee.Code == 0 {
			return 1
		}
		return ee.Code
	}
	return 1
}

func isSilent(err error) bool {
	var ee ExitError
	if errors.As(err, &ee) {
		return ee.Silent
	}
	return false
}

func exitError(code int, err error) error {
	return ExitError{Code: code, Err: err}
}

func exitErrorf(code int, format string, args ...any) error {
	return ExitError{Code: code, Err: fmt.Errorf(format, args...)}
}

func usageError() error {
	return ExitError{Code: 2, Silent: true}
}
