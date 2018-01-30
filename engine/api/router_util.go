package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

func deleteUserPermissionCache(ctx context.Context, store cache.Store) {
	if getUser(ctx) != nil {
		kp := cache.Key("users", getUser(ctx).Username, "perms")
		kg := cache.Key("users", getUser(ctx).Username, "groups")
		store.Delete(kp)
		store.Delete(kg)
	}
}

// Accepted is a helper function used by asynchronous handlers
func Accepted(w http.ResponseWriter, r *http.Request) error {
	const msg = "request accepted"
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(msg)))
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(msg))
	return nil
}

// WriteJSON is a helper function to marshal json, handle errors and set Content-Type for the best
func WriteJSON(w http.ResponseWriter, r *http.Request, data interface{}, status int) error {
	b, e := json.Marshal(data)
	if e != nil {
		return sdk.WrapError(e, "WriteJSON> unable to marshal : %s", e)
	}

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(b)))
	writeProcessTime(w)
	w.WriteHeader(status)
	w.Write(b)
	return nil
}

// writeNoContentPostMiddleware writes StatusNoContent (204) for each response with No Header Content-Type
// this is a PostMiddlewaare, launch if there no error in handler.
// If there is no Content-Type, it's because there is no body return. In CDS, we
// always use WriteJSON to send body or explicitly write Content-TYpe as application/octet-stream
// So, if there is No Content-Type, we return 204
func writeNoContentPostMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
	for headerName := range w.Header() {
		if headerName == "Content-Type" {
			return ctx, nil
		}
		if headerName == "Location" {
			return ctx, nil
		}
	}
	writeProcessTime(w)
	w.WriteHeader(http.StatusNoContent)
	return ctx, nil
}

func writeProcessTime(w http.ResponseWriter) {
	for k, v := range w.Header() {
		if k == cdsclient.ResponseAPINanosecondsTimeHeader && len(v) == 1 {
			start, err := strconv.ParseInt(v[0], 10, 64)
			if err != nil {
				log.Error("writeProcessTime> error on ParseInt header ResponseAPINanosecondsTimeHeader: %s", err)
				break
			}
			w.Header().Add(cdsclient.ResponseProcessTimeHeader, fmt.Sprintf("%d", time.Now().UnixNano()-start))
		}
	}
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
func (r *Router) GetRoute(method string, handler HandlerFunc, vars map[string]string) string {
	p1 := reflect.ValueOf(handler()).Pointer()
	var url string
	for uri, routerConfig := range r.mapRouterConfigs {
		rc := routerConfig.config[method]
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
