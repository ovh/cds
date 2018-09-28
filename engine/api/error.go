package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/service"
)

type graylogResponse struct {
	TotalResult int `json:"total_results"`
	Messages    []struct {
		Message map[string]interface{} `json:"message"`
	} `json:"messages"`
}

func (api *API) getErrorHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		if api.Config.Graylog.URL == "" || api.Config.Graylog.AccessToken == "" {
			return sdk.ErrNotImplemented
		}

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/search/universal/absolute", api.Config.Graylog.URL), nil)
		if err != nil {
			return sdk.WrapError(err, "invalid given Graylog url")
		}

		q := req.URL.Query()
		q.Add("query", fmt.Sprintf("error_uuid:%s", uuid))
		q.Add("from", "1970-01-01 00:00:00.000")
		q.Add("to", time.Now().Format("2006-01-02 15:04:05"))
		q.Add("limit", "1")
		req.URL.RawQuery = q.Encode()

		req.SetBasicAuth(api.Config.Graylog.AccessToken, "token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return sdk.WrapError(err, "cannot send query to Graylog")
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return sdk.WrapError(err, "cannot read response from Graylog")
		}

		var res graylogResponse
		if err := json.Unmarshal(body, &res); err != nil {
			return sdk.WrapError(err, "cannot unmarshal response from Graylog")
		}

		if res.TotalResult < 1 {
			return sdk.ErrNotFound
		}

		e := sdk.Error{
			UUID:    res.Messages[0].Message["error_uuid"].(string),
			Message: res.Messages[0].Message["message"].(string),
		}
		if st, ok := res.Messages[0].Message["stack_trace"]; ok {
			e.StackTrace = st.(string)
		}

		return service.WriteJSON(w, e, http.StatusOK)
	}
}
