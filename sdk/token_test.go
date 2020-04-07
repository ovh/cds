package sdk_test

import (
	"net/http"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/assert"
)

func TestAuthConsumerScopeDetailsIsValid(t *testing.T) {
	cases := []struct {
		Name    string
		Details sdk.AuthConsumerScopeDetails
		Error   bool
	}{
		{
			Name: "Some uniques scopes with unique routes and methods should be valid",
			Details: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{
							Route:   "/handler1",
							Methods: []string{http.MethodGet, http.MethodPost},
						},
						{
							Route:   "/handler2",
							Methods: []string{http.MethodGet, http.MethodPut},
						},
					},
				},
				{
					Scope: sdk.AuthConsumerScopeAction,
				},
			},
			Error: false,
		},
		{
			Name: "Duplicate scopes should generate an error",
			Details: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
				},
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
				},
			},
			Error: true,
		},
		{
			Name: "Duplicate routes should generate an error",
			Details: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{
							Route: "/handler",
						},
						{
							Route: "/handler",
						},
					},
				},
			},
			Error: true,
		},
		{
			Name: "Duplicate methods should generate an error",
			Details: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{
							Route:   "/handler",
							Methods: []string{http.MethodGet, http.MethodGet},
						},
					},
				},
			},
			Error: true,
		},
		{
			Name: "Invalid method should generate an error",
			Details: sdk.AuthConsumerScopeDetails{
				{
					Scope: sdk.AuthConsumerScopeAccessToken,
					Endpoints: sdk.AuthConsumerScopeEndpoints{
						{
							Route:   "/handler",
							Methods: []string{"UNKNOWN"},
						},
					},
				},
			},
			Error: true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			err := c.Details.IsValid()
			if c.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
