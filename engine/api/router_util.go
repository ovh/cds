package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// WriteJSON is a helper function to marshal json, handle errors and set Content-Type for the best
func WriteJSON(w http.ResponseWriter, r *http.Request, data interface{}, status int) error {
	b, e := json.Marshal(data)
	if e != nil {
		log.Warning("WriteJSON> unable to marshal : %s", e)
		return sdk.ErrUnknownError
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(b)
	return nil
}

// UnmarshalBody read the request body and tries to json.unmarshal it. It returns sdk.ErrWrongRequest in case of error.
func UnmarshalBody(r *http.Request, i interface{}) error {
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		return sdk.ErrWrongRequest
	}
	if err := json.Unmarshal(data, i); err != nil {
		err = sdk.NewError(sdk.ErrWrongRequest, err)
		return sdk.WrapError(err, "UnmarshalBody> unable to unmarshal %s", string(data))
	}
	return nil
}

func (r *Router) getRoute(method string, handler Handler, vars map[string]string) string {
	sf1 := reflect.ValueOf(handler)
	var url string
	for uri, routerConfig := range mapRouterConfigs {
		if strings.HasPrefix(uri, r.prefix) {
			switch method {
			case "GET":
				sf2 := reflect.ValueOf(routerConfig.get)
				if sf1.Pointer() == sf2.Pointer() {
					url = uri
					break
				}
			case "POST":
				sf2 := reflect.ValueOf(routerConfig.post)
				if sf1.Pointer() == sf2.Pointer() {
					url = uri
					break
				}
			case "PUT":
				sf2 := reflect.ValueOf(routerConfig.put)
				if sf1.Pointer() == sf2.Pointer() {
					url = uri
					break
				}
			case "DELETE":
				sf2 := reflect.ValueOf(routerConfig.deleteHandler)
				if sf1.Pointer() == sf2.Pointer() {
					url = uri
					break
				}
			}
		}
	}

	for k, v := range vars {
		url = strings.Replace(url, "{"+k+"}", v, -1)
	}

	if url == "" {
		log.Debug("Cant find route for Handler %s %v", method, handler)
	}

	return url
}

// FormBool return true if the form value is set to true|TRUE|yes|YES|1
func FormBool(r *http.Request, s string) bool {
	v := r.FormValue(s)
	switch v {
	case "true", "TRUE", "yes", "YES", "1":
		return true
	default:
		return false
	}
}

// requestVarInt return int value for a var in Request
func requestVarInt(r *http.Request, s string) (int64, error) {
	vars := mux.Vars(r)
	idString := vars[s]

	// Check ID Job
	id, erri := strconv.ParseInt(idString, 10, 64)
	if erri != nil {
		if s == "id" {
			return id, sdk.WrapError(sdk.ErrInvalidID, "requestVarInt> id not an integer: %s", idString)
		}
		return id, sdk.WrapError(sdk.ErrWrongRequest, "requestVarInt> %s is not an integer: %s", s, idString)
	}
	return id, nil
}
