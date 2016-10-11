package main

import (
	"net/http"

	"github.com/ovh/cds/sdk"
)

// WriteError is a helper function to return error in a language the called understand
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	al := r.Header.Get("Accept-Language")
	msg, code := sdk.ProcessError(err, al)
	sdkErr := sdk.Error{Message: msg}
	WriteJSON(w, r, sdkErr, code)
}
