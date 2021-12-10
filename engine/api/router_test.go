package api

import (
	"context"
	"net/http"
	"sort"
	"testing"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_computeScopeDetails(t *testing.T) {
	r := &Router{
		Mux:        mux.NewRouter(),
		Background: context.TODO(),
	}

	myHandler := func() service.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			return nil
		}
	}

	r.Handle("/handler1", Scope(sdk.AuthConsumerScopeUser), r.GET(myHandler))
	r.Handle("/handler2", Scope(sdk.AuthConsumerScopeAction), r.GET(myHandler), r.PUT(myHandler))
	r.Handle("/handler3", Scopes(sdk.AuthConsumerScopeAccessToken, sdk.AuthConsumerScopeAction), r.GET(myHandler), r.POST(myHandler), r.DELETE(myHandler))

	r.computeScopeDetails()

	for i := range r.scopeDetails {
		switch r.scopeDetails[i].Scope {
		case sdk.AuthConsumerScopeUser:
			require.Len(t, r.scopeDetails[i].Endpoints, 1)
			assert.Equal(t, r.scopeDetails[i].Endpoints[0].Route, "/handler1")
			require.Len(t, r.scopeDetails[i].Endpoints[0].Methods, 1)
		case sdk.AuthConsumerScopeAction:
			sort.Slice(r.scopeDetails[i].Endpoints, func(j, k int) bool {
				return r.scopeDetails[i].Endpoints[j].Route < r.scopeDetails[i].Endpoints[k].Route
			})
			require.Len(t, r.scopeDetails[i].Endpoints, 2)
			assert.Equal(t, r.scopeDetails[i].Endpoints[0].Route, "/handler2")
			require.Len(t, r.scopeDetails[i].Endpoints[0].Methods, 2)
			assert.Equal(t, r.scopeDetails[i].Endpoints[1].Route, "/handler3")
			require.Len(t, r.scopeDetails[i].Endpoints[1].Methods, 3)
		case sdk.AuthConsumerScopeAccessToken:
			require.Len(t, r.scopeDetails[i].Endpoints, 1)
			assert.Equal(t, r.scopeDetails[i].Endpoints[0].Route, "/handler3")
			require.Len(t, r.scopeDetails[i].Endpoints[0].Methods, 3)
		}
	}
}
