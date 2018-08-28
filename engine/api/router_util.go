package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) deleteUserPermissionCache(ctx context.Context, store cache.Store) {
	if getUser(ctx) != nil {
		username := getUser(ctx).Username
		kp := cache.Key("users", username, "perms")
		kg := cache.Key("users", username, "groups")
		store.Delete(kp)
		store.Delete(kg)
	}
}

// writeNoContentPostMiddleware writes StatusNoContent (204) for each response with No Header Content-Type
// this is a PostMiddlewaare, launch if there no error in handler.
// If there is no Content-Type, it's because there is no body return. In CDS, we
// always use service.WriteJSON to send body or explicitly write Content-TYpe as application/octet-stream
// So, if there is No Content-Type, we return 204 with content type to json
func writeNoContentPostMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	for headerName := range w.Header() {
		if headerName == "Content-Type" {
			return ctx, nil
		}
		if headerName == "Location" {
			return ctx, nil
		}
	}
	service.WriteProcessTime(w)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	return ctx, nil
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

// GetRoute returns the routes given a handler
func (r *Router) GetRoute(method string, handler service.HandlerFunc, vars map[string]string) string {
	p1 := reflect.ValueOf(handler()).Pointer()
	var url string
	for uri, routerConfig := range r.mapRouterConfigs {
		rc := routerConfig.Config[method]
		if rc == nil {
			continue
		}

		if strings.HasPrefix(uri, r.Prefix) {
			sf2 := reflect.ValueOf(rc.Handler)
			if p1 == sf2.Pointer() {
				url = uri
				break
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

// FormString return a string
func FormString(r *http.Request, s string) string {
	return r.FormValue(s)
}

// FormInt return a int from query params
func FormInt(r *http.Request, s string) (int, error) {
	stringValue := FormString(r, s)
	if stringValue == "" {
		return 0, nil
	}
	i, err := strconv.Atoi(stringValue)
	if err != nil {
		return i, sdk.WrapError(sdk.ErrInvalidNumber, "FormInt> %s is not a integer", stringValue)
	}
	return i, nil
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

func translate(r *http.Request, msgList []sdk.Message) []string {
	al := r.Header.Get("Accept-Language")
	msgListString := []string{}
	for _, m := range msgList {
		s := m.String(al)
		if s != "" {
			msgListString = append(msgListString, s)
		}
	}
	return msgListString
}
