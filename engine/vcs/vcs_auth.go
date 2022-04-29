package vcs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// Context
type contextKey string

var (
	contextKeyVCSProjectConf     contextKey = "vcs-project-conf"
	contextKeyAccessToken        contextKey = "access-token"
	contextKeyAccessTokenCreated contextKey = "access-token-created"
	contextKeyAccessTokenSecret  contextKey = "access-token-secret"
)

func (s *Service) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	encodedVCSProjectConf := req.Header.Get(sdk.HeaderXVCSProjectConf)
	if encodedVCSProjectConf != "" {
		vcsProjectConf, err := base64.StdEncoding.DecodeString(encodedVCSProjectConf)
		if err != nil {
			return ctx, fmt.Errorf("bad header syntax: %s", err)
		}
		if len(vcsProjectConf) != 0 {
			var vcsProject sdk.VCSProject
			if err := json.Unmarshal(vcsProjectConf, &vcsProject); err != nil {
				return nil, sdk.WrapError(sdk.ErrUnauthorized, "invalid vcs project configuration err:%v", err)
			}
			ctx = context.WithValue(ctx, contextKeyVCSProjectConf, vcsProject)
		}
		return ctx, nil
	}

	encodedAccessToken := req.Header.Get(sdk.HeaderXAccessToken)
	accessToken, err := base64.StdEncoding.DecodeString(encodedAccessToken)
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax: %s", err)
	}

	encodedAccessTokenSecret := req.Header.Get(sdk.HeaderXAccessTokenSecret)
	accessTokenSecret, err := base64.StdEncoding.DecodeString(encodedAccessTokenSecret)
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax: %s", err)
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
	vcsProject, ok := ctx.Value(contextKeyVCSProjectConf).(sdk.VCSProject)
	if ok {
		vcsAuth.VCSProject = &vcsProject
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
		vcsAuth.AccessTokenSecret != "" &&
		vcsAuth.AccessTokenCreated > 0 {
		return vcsAuth, nil
	}

	return sdk.VCSAuth{}, sdk.WrapError(sdk.ErrUnauthorized, "invalid access token headers")
}
