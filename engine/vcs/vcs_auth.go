package vcs

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// Context
type contextKey string

var (
	contextKeyVCSURL      contextKey = "vcs-url"
	contextKeyVCSURLApi   contextKey = "vcs-url-api"
	contextKeyVCSType     contextKey = "vcs-type"
	contextKeyVCSUsername contextKey = "vcs-username"
	contextKeyVCSToken    contextKey = "vcs-token"
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
	if string(vcsType) != "" {
		ctx = context.WithValue(ctx, contextKeyVCSURL, string(vcsURL))
		ctx = context.WithValue(ctx, contextKeyVCSURLApi, string(vcsURLApi))
		ctx = context.WithValue(ctx, contextKeyVCSType, string(vcsType))
		ctx = context.WithValue(ctx, contextKeyVCSUsername, string(vcsUsername))
		ctx = context.WithValue(ctx, contextKeyVCSToken, string(vcsToken))
		return ctx, nil
	}

	return ctx, nil
}

func getVCSAuth(ctx context.Context) (sdk.VCSAuth, error) {
	var vcsAuth sdk.VCSAuth
	vcsType, ok := ctx.Value(contextKeyVCSType).(string)
	if !ok {
		return sdk.VCSAuth{}, sdk.WrapError(sdk.ErrUnauthorized, "invalid access token headers")
	}

	vcsURL, _ := ctx.Value(contextKeyVCSURL).(string)
	vcsAuth.URL = vcsURL

	username, _ := ctx.Value(contextKeyVCSUsername).(string)
	vcsAuth.Username = username

	vcsURLApi, _ := ctx.Value(contextKeyVCSURLApi).(string)
	vcsAuth.URLApi = vcsURLApi

	vcsAuth.Type = vcsType

	token, _ := ctx.Value(contextKeyVCSToken).(string)
	vcsAuth.Token = token

	return vcsAuth, nil
}
