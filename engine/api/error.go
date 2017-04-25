package main

import (
	"net/http"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// WriteError is a helper function to return error in a language the called understand
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	al := r.Header.Get("Accept-Language")
	msg, code := sdk.ProcessError(err, al)
	sdkErr := sdk.Error{Message: msg}
	log.Warning("%-7s | %-4d | %s \t %s", r.Method, code, r.RequestURI, err)
	WriteJSON(w, r, sdkErr, code)
}
