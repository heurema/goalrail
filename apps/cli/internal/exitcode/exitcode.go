package exitcode

import "errors"

const (
	Success    = 0
	Runtime    = 1
	Usage      = 2
	Validation = 3
)

// Error wraps an error with the CLI exit code that should be used by main.
type Error struct {
	code int
	err  error
}

func (e *Error) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *Error) Code() int {
	if e == nil {
		return Runtime
	}
	return e.code
}

func With(code int, err error) error {
	if err == nil {
		return nil
	}
	return &Error{code: code, err: err}
}

func UsageError(err error) error {
	return With(Usage, err)
}

func RuntimeError(err error) error {
	return With(Runtime, err)
}

func ValidationError(err error) error {
	return With(Validation, err)
}

func ForError(err error) int {
	if err == nil {
		return Success
	}

	var coded *Error
	if errors.As(err, &coded) {
		return coded.Code()
	}

	return Runtime
}
