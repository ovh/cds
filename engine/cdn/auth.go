package cdn

import (
	"context"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/engine/authentication"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/sdk"
)

var (
	jwtCookieName = "jwt_token"
	keyPermission = cache.Key("cdn", "permission")
)

func (s *Service) checkAuthJWT(req *http.Request) (*sdk.AuthSessionJWTClaims, error) {
	var jwtRaw string

	// Try to get the jwt from the cookie firstly then from the authorization bearer header, a XSRF token with cookie
	jwtCookie, _ := req.Cookie(jwtCookieName)
	if jwtCookie != nil {
		jwtRaw = jwtCookie.Value
	} else if strings.HasPrefix(req.Header.Get("Authorization"), "Bearer ") {
		jwtRaw = strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	}
	if jwtRaw == "" {
		return nil, sdk.WithStack(sdk.ErrUnauthorized)
	}

	v := authentication.NewVerifier(s.ParsedAPIPublicKey)
	token, err := jwt.ParseWithClaims(jwtRaw, &sdk.AuthSessionJWTClaims{}, v.VerifyJWT)
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}
	claims, ok := token.Claims.(*sdk.AuthSessionJWTClaims)
	if !ok || !token.Valid {
		return nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	return claims, nil
}

func (s *Service) checkItemAccess(ctx context.Context, itemType sdk.CDNItemType, apiRef string, sessionID string) error {
	keyWorkflowPermissionForSession := cache.Key(keyPermission, apiRef, sessionID)

	exists, err := s.Cache.Exist(keyWorkflowPermissionForSession)
	if err != nil {
		return sdk.WrapError(err, "unable to check if permission %s exists", keyWorkflowPermissionForSession)
	}
	if exists {
		return nil
	}

	item, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRef, itemType)
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.ErrNotFound)
	}

	if err := s.Client.WorkflowLogAccess(item.APIRef.ProjectKey, item.APIRef.WorkflowName, sessionID); err != nil {
		return sdk.NewErrorWithStack(err, sdk.ErrNotFound)
	}

	if err := s.Cache.SetWithTTL(keyWorkflowPermissionForSession, true, 3600); err != nil {
		return sdk.WrapError(err, "unable to store permission %s", keyWorkflowPermissionForSession)
	}

	return nil
}
