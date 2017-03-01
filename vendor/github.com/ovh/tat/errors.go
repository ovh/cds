package tat

import (
	"fmt"
	"net/http"
)

//APIError is the Error wrapper for api error management
type APIError struct {
	Code  int    `json:"-"`
	Cause string `json:"error"`
}

func (e *APIError) Error() string {
	return e.Cause
}

func (e *APIError) String() string {
	return fmt.Sprintf("(%d) %s", e.Code, e.Cause)
}

//NewError returns a new APIError
func NewError(code int, format string, a ...interface{}) error {
	return &APIError{code, fmt.Sprintf(format, a...)}
}

//Error returns the error in the proper way
func Error(err error) (int, interface{}) {
	switch err.(type) {
	case *APIError:
		err1 := err.(*APIError)
		return err1.Code, err1
	default:
		e := map[string]string{"error": err.Error()}
		return http.StatusInternalServerError, e
	}
}
