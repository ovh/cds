package api

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestSearchFreeTextQuery(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)

	// Every project shares a unique token so that the assertions cannot be disturbed by the
	// projects inserted by the other tests, an admin being allowed to search all of them.
	token := sdk.RandomString(10)
	projAlpha := assets.InsertTestProject(t, db, api.Cache, "aaa"+token, "alpha"+token)
	projBeta := assets.InsertTestProject(t, db, api.Cache, "bbb"+token, "beta"+token)

	uri := api.Router.GetRouteV2("GET", api.getSearchHandler, map[string]string{})
	test.NotEmpty(t, uri)

	search := func(t *testing.T, query string) []string {
		req := assets.NewAuthentifiedRequest(t, admin, pwd, "GET", uri+"?query="+url.QueryEscape(query), nil)
		w := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code)

		var results sdk.SearchResults
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results))
		require.Equal(t, strconv.Itoa(len(results)), w.Header().Get("X-Total-Count"))

		ids := make([]string, 0, len(results))
		for i := range results {
			ids = append(ids, results[i].ID)
		}
		return ids
	}

	for _, tc := range []struct {
		name     string
		query    string
		expected []string
	}{
		{"single word", token, []string{projAlpha.Key, projBeta.Key}},
		{"single word on the label", "alpha" + token, []string{projAlpha.Key}},
		{"all words must match", token + " alpha", []string{projAlpha.Key}},
		{"words can be given in any order", "alpha " + token, []string{projAlpha.Key}},
		{"words can match different fields", "aaa" + token + " alpha", []string{projAlpha.Key}},
		{"extra spaces are ignored", "  alpha   " + token + " ", []string{projAlpha.Key}},
		{"words are case insensitive", "ALPHA" + token, []string{projAlpha.Key}},
		{"unmatched word excludes the result", token + " gamma", []string{}},
		{"like wildcards are escaped", token + "%", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ElementsMatch(t, tc.expected, search(t, tc.query))
		})
	}
}
