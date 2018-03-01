package api

import (
	"net/http"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// WriteError is a helper function to return error in a language the called understand
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	al := r.Header.Get("Accept-Language")
	msg, errProcessed := sdk.ProcessError(err, al)
	sdkErr := sdk.Error{Message: msg}

	// ErrAlreadyTaken and ErrWorkerModelAlreadyBooked are not useful to log in warning
	if sdk.ErrorIs(errProcessed, sdk.ErrAlreadyTaken) || sdk.ErrorIs(errProcessed, sdk.ErrWorkerModelAlreadyBooked) {
		log.Debug("%-7s | %-4d | %s \t %s", r.Method, errProcessed.Status, r.RequestURI, err)
	} else {
		log.Warning("%-7s | %-4d | %s \t %s", r.Method, errProcessed.Status, r.RequestURI, err)
	}

	WriteJSON(w, sdkErr, errProcessed.Status)
}
