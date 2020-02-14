package builtin

import (
	"net/http"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/assert"
)

func Test_checkNewConsumerScopes(t *testing.T) {
	cases := []struct {
		Name         string
		ParentScopes sdk.AuthConsumerScopeDetails
		Scopes       sdk.AuthConsumerScopeDetails
		Error        bool
	}{
		{
			Name: "Parent has no scopes (ex: local consumer)",
			Scopes: sdk.AuthConsumerScopeDetails{
				{Scope: sdk.AuthConsumerScopeAccessToken},
			},
			Error: false,
		},
		{
			Name: "Scope are not in parent scopes",
			ParentScopes: sdk.AuthConsumerScopeDetails{
				{Scope: sdk.AuthConsumerScopeAccessToken},
			},
			Scopes: sdk.AuthConsumerScopeDetails{
				{Scope: sdk.AuthConsumerScopeAction},
			},
			Error: true,
		},
		{
			Name: "Scope all routes are not in parent",
			ParentScopes: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{Route: "/handler"},
					},
				},
			},
			Scopes: sdk.AuthConsumerScopeDetails{
				{Scope: sdk.AuthConsumerScopeAccessToken},
			},
			Error: true,
		}, {
			Name: "Scope all routes are in parent",
			ParentScopes: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
				},
				{
					Scope: sdk.AuthConsumerScopeAction,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{Route: "/handler"},
					},
				},
			},
			Scopes: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{Route: "/handler"},
					},
				},
				{
					Scope: sdk.AuthConsumerScopeAction,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{Route: "/handler"},
					},
				},
			},
			Error: false,
		},
		{
			Name: "Scope all route methods are not in parent",
			ParentScopes: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{
							Route:   "/handler",
							Methods: []string{http.MethodGet},
						},
					},
				},
			},
			Scopes: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{
							Route:   "/handler",
							Methods: []string{http.MethodPost},
						},
					},
				},
			},
			Error: true,
		},
		{
			Name: "Scope all route methods are in parent",
			ParentScopes: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{
							Route: "/handler1",
						},
						{
							Route:   "/handler2",
							Methods: []string{http.MethodGet},
						},
					},
				},
			},
			Scopes: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{
							Route:   "/handler1",
							Methods: []string{http.MethodGet},
						},
						{
							Route:   "/handler2",
							Methods: []string{http.MethodGet},
						},
					},
				},
			},
			Error: false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			err := checkNewConsumerScopes(c.ParentScopes, c.Scopes)
			if c.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
