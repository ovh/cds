package api

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

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
	service.WriteProcessTime(ctx, w)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	return ctx, nil
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

// FormString return a string
func FormString(r *http.Request, s string) string {
	return r.FormValue(s)
}

// QueryString return a string from a query parameter
func QueryString(r *http.Request, s string) string {
	return r.FormValue(s)
}

// QueryBool return a boolean from a query parameter
func QueryBool(r *http.Request, s string) bool {
	return service.FormBool(r, s)
}

// QueryStrings returns the list of values for given query param key or nil if key no values.
func QueryStrings(r *http.Request, key string) ([]string, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	if v, ok := r.Form[key]; ok {
		return v, nil
	}
	return nil, nil
}

// SortOrder constant.
type SortOrder string

// SortOrders.
const (
	ASC  SortOrder = "asc"
	DESC SortOrder = "desc"
)

func validateSortOrder(s string) bool {
	switch SortOrder(s) {
	case ASC, DESC:
		return true
	}
	return false
}

// SortCompareInt returns the result of the right compare equation depending of given sort order.
func SortCompareInt(i, j int, o SortOrder) bool {
	if o == ASC {
		return i < j
	}
	return i > j
}

// QuerySort returns the a of key found in sort query param or nil if sort param not found.
func QuerySort(r *http.Request) (map[string]SortOrder, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	v, ok := r.Form["sort"]
	if !ok {
		return nil, nil
	}

	res := map[string]SortOrder{}
	for _, item := range strings.Split(v[0], ",") {
		if item == "" {
			return nil, sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("invalid given sort key"))
		}
		s := strings.Split(item, ":")
		if len(s) > 1 {
			if !validateSortOrder(s[1]) {
				return nil, sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("invalid given sort param"))
			}
			res[s[0]] = SortOrder(s[1])
		} else {
			res[s[0]] = ASC
		}
	}

	return res, nil
}

// requestVarInt return int value for a var in Request
func requestVarInt(r *http.Request, s string) (int64, error) {
	vars := mux.Vars(r)

	// Check ID Job
	id, err := strconv.ParseInt(vars[s], 10, 64)
	if err != nil {
		err = sdk.WrapError(err, "%s is not an integer: %s", s, vars[s])
		if s == "id" {
			return 0, sdk.NewErrorWithStack(err, sdk.ErrInvalidID)
		}
		return 0, sdk.NewErrorWithStack(err, sdk.ErrWrongRequest)
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
