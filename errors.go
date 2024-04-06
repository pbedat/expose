package expose

import "errors"

type ErrWithCode struct {
	code string
	err  error
}

func (e ErrWithCode) Code() string {
	return e.code
}

func (e ErrWithCode) Error() string {
	return e.err.Error()
}

type WithCode interface {
	error
	Code() string
}

func SetErrCode(err error, code string) error {
	return &ErrWithCode{code: code, err: err}
}

func GetErrCode(err error) (string, bool) {
	var withCode WithCode
	if errors.As(err, &withCode) {
		return withCode.Code(), true
	}

	return "", false
}
