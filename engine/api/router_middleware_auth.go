package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	var err error
	var shouldContinue bool

	// Providers (like arsenal)
	ctx, shouldContinue, err = api.authAllowProviderMiddleware(ctx, w, req, rc)
	if err != nil {
		return ctx, sdk.WithStack(err)
	}
	if !shouldContinue {
		return ctx, nil
	}

	// Tokens (like izanamy)
	ctx, shouldContinue, err = api.authStatusTokenMiddleware(ctx, w, req, rc)
	if err != nil {
		return ctx, sdk.WithStack(err)
	}
	if !shouldContinue {
		return ctx, nil
	}

	// JWT base authentification
	token := JWT(ctx)
	if token == nil {
		return ctx, nil
	}

	// Put the granted user in the context
	var APIConsumer = sdk.APIConsumer{
		Fullname:   token.Description, // TODO
		OnBehalfOf: token.AuthentifiedUser,
		Groups:     token.Groups,
	}
	ctx = context.WithValue(ctx, auth.ContextAPIConsumer, &APIConsumer)

	return ctx, nil
}

// Check Provider
func (api *API) authAllowProviderMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
	if rc.Options["allowProvider"] == "true" {
		providerName := req.Header.Get("X-Provider-Name")
		providerToken := req.Header.Get("X-Provider-Token")
		var providerOK bool
		for _, p := range api.Config.Providers {
			if p.Name == providerName && p.Token == providerToken {
				providerOK = true
				break
			}
		}
		if providerOK {
			ctx = context.WithValue(ctx, auth.ContextProvider, providerName)
			return ctx, false, nil
		}
	}
	return ctx, true, nil
}

// Checks static tokens
func (api *API) authStatusTokenMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
	if h, ok := rc.Options["token"]; ok {
		headerSplitted := strings.Split(h, ":")
		receivedValue := req.Header.Get(headerSplitted[0])
		if receivedValue != headerSplitted[1] {
			return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied token on %s %s for %s", req.Method, req.URL, req.RemoteAddr)
		}
		return ctx, false, nil
	}
	return ctx, true, nil
}
