package vcs

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
)

// HTTP Headers
const (
	HeaderXAccessToken       = "X-CDS-ACCESS-TOKEN"
	HeaderXAccessTokenSecret = "X-CDS-ACCESS-TOKEN-SECRET"
)

// Context
type contextKey string

var (
	contextKeyAccessToken       contextKey = "access-token"
	contextKeyAccessTokenSecret contextKey = "access-token-secret"
)

func (s *Service) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *api.HandlerConfig) (context.Context, error) {
	if rc.Options["auth"] != "true" {
		return ctx, nil
	}

	hash, err := base64.StdEncoding.DecodeString(req.Header.Get(sdk.AuthHeader))
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax: %s", err)
	}

	encodedAccessToken := req.Header.Get(HeaderXAccessToken)
	accessToken, err := base64.StdEncoding.DecodeString(encodedAccessToken)
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax: %s", err)
	}

	encodedAccessTokenSecret := req.Header.Get(HeaderXAccessTokenSecret)
	accessTokenSecret, err := base64.StdEncoding.DecodeString(encodedAccessTokenSecret)
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax: %s", err)
	}

	if len(accessToken) != 0 {
		ctx = context.WithValue(ctx, contextKeyAccessToken, string(accessToken))
	}
	if len(accessTokenSecret) != 0 {
		ctx = context.WithValue(ctx, contextKeyAccessTokenSecret, string(accessTokenSecret))
	}

	if s.Hash != string(hash) {
		return ctx, sdk.ErrUnauthorized
	}

	return ctx, nil
}

func getAccessTokens(ctx context.Context) (string, string, bool) {
	accessToken, ok := ctx.Value(contextKeyAccessToken).(string)
	if !ok {
		return "", "", false
	}
	accessTokenSecret, ok := ctx.Value(contextKeyAccessTokenSecret).(string)
	if !ok {
		return "", "", false
	}
	return string(accessToken), string(accessTokenSecret), len(accessToken) > 0
}
