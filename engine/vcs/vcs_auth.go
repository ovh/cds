package vcs

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
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
	projectKey, err := base64.StdEncoding.DecodeString(req.Header.Get(sdk.HeaderXVCSProjectKey))
	if err != nil {
		return nil, sdk.WrapError(err, "bad header syntax for HeaderXVCSProjectKey")
	}
	if string(vcsType) != "" {
		ctx = context.WithValue(ctx, contextKeyVCSURL, string(vcsURL))
		ctx = context.WithValue(ctx, contextKeyVCSURLApi, string(vcsURLApi))
		ctx = context.WithValue(ctx, contextKeyVCSType, string(vcsType))
		ctx = context.WithValue(ctx, contextKeyVCSUsername, string(vcsUsername))
		ctx = context.WithValue(ctx, contextKeyVCSToken, string(vcsToken))

		// identify which credentials served the call: without them a vcs error cannot be told apart
		// from a permission issue. The token is deliberately never logged.
		ctx = context.WithValue(ctx, cdslog.VCSType, string(vcsType))
		ctx = context.WithValue(ctx, cdslog.VCSURL, string(vcsURL))
		ctx = context.WithValue(ctx, cdslog.VCSUsername, string(vcsUsername))
		ctx = context.WithValue(ctx, cdslog.Project, string(projectKey))
		if name := muxVar(req, "name"); name != "" {
			ctx = context.WithValue(ctx, cdslog.VCSServer, name)
		}
		if repo := muxVar(req, "repo"); repo != "" {
			ctx = context.WithValue(ctx, cdslog.Repository, muxVar(req, "owner")+"/"+repo)
		}
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
