package jujerr

import (
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	"github.com/loopfz/gadgeto/tonic"
)

func ErrHook(c *gin.Context, e error) (int, interface{}) {

	errcode, errpl := 500, e.Error()
	if _, ok := e.(tonic.InputError); ok {
		errcode, errpl = 400, e.Error()
	} else {
		switch {
		case errors.IsBadRequest(e) || errors.IsNotValid(e) || errors.IsAlreadyExists(e) || errors.IsNotSupported(e) || errors.IsNotAssigned(e) || errors.IsNotProvisioned(e):
			errcode, errpl = 400, e.Error()
		case errors.IsForbidden(e):
			errcode, errpl = 403, e.Error()
		case errors.IsMethodNotAllowed(e):
			errcode, errpl = 405, e.Error()
		case errors.IsNotFound(e) || errors.IsUserNotFound(e):
			errcode, errpl = 404, e.Error()
		case errors.IsUnauthorized(e):
			errcode, errpl = 401, e.Error()
		case errors.IsNotImplemented(e):
			errcode, errpl = 501, e.Error()
		}
	}

	return errcode, gin.H{`error`: errpl}
}
