package vcs

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// Context
type contextKey string

var (
	contextKeyVCSURL             contextKey = "vcs-url"
	contextKeyVCSURLApi          contextKey = "vcs-url-api"
	contextKeyVCSType            contextKey = "vcs-type"
	contextKeyVCSUsername        contextKey = "vcs-username"
	contextKeyVCSToken           contextKey = "vcs-token"
	contextKeyAccessToken        contextKey = "access-token"         // DEPRECATED VCS
	contextKeyAccessTokenCreated contextKey = "access-token-created" // DEPRECATED VCS
	contextKeyAccessTokenSecret  contextKey = "access-token-secret"  // DEPRECATED VCS
)

func (s *Service) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	vcsURL, err := base64.StdEncoding.DecodeString(req.Header.Get(sdk.HeaderXVCSURL))
	if err != nil {
		return nil, sdk.WrapError(err, "bad header syntax for HeaderXVCSURL")
	}
	vcsURLApi, err := base64.StdEncoding.DecodeString(req.Header.Get(sdk.HeaderXVCSURLApi))
	if err != nil {
		return nil, sdk.WrapError(err, "bad header syntax for HeaderXVCSURLApi")
	}
	vcsType, err := base64.StdEncoding.DecodeString(req.Header.Get(sdk.HeaderXVCSType))
	if err != nil {
		return nil, sdk.WrapError(err, "bad header syntax for HeaderXVCSType")
	}
	vcsUsername, err := base64.StdEncoding.DecodeString(req.Header.Get(sdk.HeaderXVCSUsername))
	if err != nil {
		return nil, sdk.WrapError(err, "bad header syntax for HeaderXVCSUsername")
	}
	vcsToken, err := base64.StdEncoding.DecodeString(req.Header.Get(sdk.HeaderXVCSToken))
	if err != nil {
		return nil, sdk.WrapError(err, "bad header syntax for HeaderXVCSToken")
	}
	if string(vcsURL) != "" {
		ctx = context.WithValue(ctx, contextKeyVCSURL, string(vcsURL))
		ctx = context.WithValue(ctx, contextKeyVCSURLApi, string(vcsURLApi))
		ctx = context.WithValue(ctx, contextKeyVCSType, string(vcsType))
		ctx = context.WithValue(ctx, contextKeyVCSUsername, string(vcsUsername))
		ctx = context.WithValue(ctx, contextKeyVCSToken, string(vcsToken))
		return ctx, nil
	}

	encodedAccessToken := req.Header.Get(sdk.HeaderXAccessToken)
	accessToken, err := base64.StdEncoding.DecodeString(encodedAccessToken)
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax for access token: %s", err)
	}

	encodedAccessTokenSecret := req.Header.Get(sdk.HeaderXAccessTokenSecret)
	accessTokenSecret, err := base64.StdEncoding.DecodeString(encodedAccessTokenSecret)
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax for access token secret: %s", err)
	}

	encodedAccessTokenCreated := req.Header.Get(sdk.HeaderXAccessTokenCreated)
	accessTokenCreated, err := base64.StdEncoding.DecodeString(encodedAccessTokenCreated)
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax for access token created: %s", err)
	}

	if len(accessToken) != 0 {
		ctx = context.WithValue(ctx, contextKeyAccessToken, string(accessToken))
	}
	if len(accessTokenSecret) != 0 {
		ctx = context.WithValue(ctx, contextKeyAccessTokenSecret, string(accessTokenSecret))
	}
	if len(accessTokenCreated) != 0 {
		ctx = context.WithValue(ctx, contextKeyAccessTokenCreated, string(accessTokenCreated))
	}
	return ctx, nil
}

func getVCSAuth(ctx context.Context) (sdk.VCSAuth, error) {
	var vcsAuth sdk.VCSAuth
	vcsURL, ok := ctx.Value(contextKeyVCSURL).(string)

	if ok {
		vcsAuth.URL = vcsURL

		username, _ := ctx.Value(contextKeyVCSUsername).(string)
		vcsAuth.Username = username

		vcsURLApi, _ := ctx.Value(contextKeyVCSURLApi).(string)
		vcsAuth.URLApi = vcsURLApi

		vcsType, _ := ctx.Value(contextKeyVCSType).(string)
		vcsAuth.Type = vcsType

		token, _ := ctx.Value(contextKeyVCSToken).(string)
		vcsAuth.Token = token

		return vcsAuth, nil
	}

	// DEPRECATED VCS
	accessToken, _ := ctx.Value(contextKeyAccessToken).(string)
	vcsAuth.AccessToken = accessToken

	accessTokenSecret, _ := ctx.Value(contextKeyAccessTokenSecret).(string)
	vcsAuth.AccessTokenSecret = accessTokenSecret

	accessTokenCreated, _ := ctx.Value(contextKeyAccessTokenCreated).(string)
	if accessTokenCreated != "" {
		created, err := strconv.ParseInt(accessTokenCreated, 10, 64)
		if err != nil {
			return sdk.VCSAuth{}, sdk.WrapError(sdk.ErrUnauthorized, "invalid token created header: %v err:%v", accessTokenCreated, err)
		}
		vcsAuth.AccessTokenCreated = created
	}

	if vcsAuth.AccessToken != "" &&
		vcsAuth.AccessTokenSecret != "" {
		return vcsAuth, nil
	}

	return sdk.VCSAuth{}, sdk.WrapError(sdk.ErrUnauthorized, "invalid access token headers")
}
