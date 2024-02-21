package xray

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	InternalServerError = errors.New("internal server error")
	ErrNotFound         = errors.New("not found")
	ErrConfict          = errors.New("conflict")
	ErrBadRequest       = errors.New("bad request")
)

func ErrorIs(expected, actual error) bool {
	return expected.Error() == actual.Error()
}

func CheckError(code int) error {
	switch code {
	case 0:
		return nil
	case 409:
		return ErrConfict
	case 404:
		return ErrNotFound
	case 400:
		return ErrBadRequest
	default:
		if code < 400 {
			return nil
		}
		return errors.WithStack(fmt.Errorf("HTTP Error code %d", code))
	}
}
