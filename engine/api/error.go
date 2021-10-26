package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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
		id := vars["request_id"]

		if api.Config.Graylog.URL == "" || api.Config.Graylog.AccessToken == "" {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/search/universal/absolute", api.Config.Graylog.URL), nil)
		if err != nil {
			return sdk.WrapError(err, "invalid given Graylog url")
		}

		q := req.URL.Query()
		q.Add("query", fmt.Sprintf("request_id:%s", id))
		q.Add("from", "1970-01-01 00:00:00.000")
		q.Add("to", time.Now().Format("2006-01-02 15:04:05"))
		q.Add("filter", fmt.Sprintf("streams:%s", api.Config.Graylog.Stream))
		req.URL.RawQuery = q.Encode()

		req.SetBasicAuth(api.Config.Graylog.AccessToken, "token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return sdk.WrapError(err, "cannot send query to Graylog")
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return sdk.WrapError(err, "cannot read response from Graylog")
		}

		var res graylogResponse
		if err := sdk.JSONUnmarshal(body, &res); err != nil {
			return sdk.WrapError(err, "cannot unmarshal response from Graylog")
		}

		if res.TotalResult < 1 {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		logs := make([]sdk.Error, res.TotalResult)
		for i := range res.Messages {
			logs[i].RequestID = res.Messages[i].Message["request_id"].(string)
			logs[i].Message = res.Messages[i].Message["message"].(string)
			if st, ok := res.Messages[i].Message["stack_trace"]; ok {
				logs[i].StackTrace = st.(string)
			}
		}

		return service.WriteJSON(w, logs, http.StatusOK)
	}
}
