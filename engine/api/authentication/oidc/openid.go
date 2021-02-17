package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/authentication"

	oidc "github.com/coreos/go-oidc"
	"github.com/ovh/cds/sdk"
	"golang.org/x/oauth2"
)

var _ sdk.AuthDriverWithRedirect = (*authDriver)(nil)
var _ sdk.AuthDriverWithSigninStateToken = (*authDriver)(nil)

// NewDriver returns a new OIDC auth driver for given config.
func NewDriver(signupDisabled bool, cdsURL, url, clientID, clientSecret string) (sdk.AuthDriver, error) {
	provider, err := oidc.NewProvider(context.Background(), url)
	if err != nil {
		return nil, sdk.WrapError(err, "failed to initialize OIDC driver")
	}
	// Configure an OpenID Connect aware OAuth2 client.
	oauth2Config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("%s/auth/callback/%s", cdsURL, sdk.ConsumerOIDC),
		// Discovery returns the OAuth2 endpoints.
		Endpoint: provider.Endpoint(),
		// "openid" is a required scope for OpenID Connect flows.
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}
	oidcConfig := &oidc.Config{
		ClientID: clientID,
	}
	verifier := provider.Verifier(oidcConfig)

	return &authDriver{
		signupDisabled: signupDisabled,
		cdsURL:         cdsURL,
		OAuth2Config:   oauth2Config,
		Verifier:       verifier,
	}, nil
}

type authDriver struct {
	signupDisabled bool
	cdsURL         string
	OAuth2Config   oauth2.Config
	Verifier       *oidc.IDTokenVerifier
}

func (d authDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerOIDC,
		SignupDisabled: d.signupDisabled,
	}
}

func (d authDriver) GetSigninURI(signinState sdk.AuthSigninConsumerToken) (sdk.AuthDriverSigningRedirect, error) {
	// Generate a new state value for the auth signin request
	jws, err := authentication.NewDefaultSigninStateToken(signinState.Origin,
		signinState.RedirectURI, signinState.IsFirstConnection)
	if err != nil {
		return sdk.AuthDriverSigningRedirect{}, err
	}

	var result = sdk.AuthDriverSigningRedirect{
		Method: http.MethodGet,
		URL:    d.OAuth2Config.AuthCodeURL(jws),
	}

	return result, nil
}

func (d authDriver) GetSessionDuration() time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

func (d authDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	if code, ok := req["code"]; !ok || code == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid code")
	}
	return nil
}

func (d authDriver) CheckSigninStateToken(req sdk.AuthConsumerSigninRequest) error {
	// Check if state is given and if its valid
	state, okState := req["state"]
	if !okState {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing state value")
	}

	return authentication.CheckDefaultSigninStateToken(state)
}

func (d authDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var info sdk.AuthDriverUserInfo

	ctx2 := context.WithValue(context.Background(), oauth2.HTTPClient, http.DefaultClient)
	oauth2Token, err := d.OAuth2Config.Exchange(ctx2, req["code"])
	if err != nil {
		return info, sdk.WrapError(err, "failed to exchange token")
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return info, sdk.WithStack(fmt.Errorf("no id_token field in oauth2 token"))
	}
	idToken, err := d.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return info, sdk.WrapError(err, "failed to verify ID Token")
	}
	tokenClaim := make(map[string]interface{})
	if err := idToken.Claims(&tokenClaim); err != nil {
		return info, sdk.WrapError(err, "cannot unmarshal OIDC claim")
	}

	// Check if email is verified.
	// See standard claims at https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
	if verified, ok := tokenClaim["email_verified"].(bool); !ok || !verified {
		return info, sdk.NewErrorFrom(sdk.ErrInvalidUser, "OIDC user's email not verified")
	}
	if info.ExternalID, ok = tokenClaim["sub"].(string); !ok {
		return info, sdk.WithStack(errors.New("missing OIDC user ID in token claim"))
	}

	if info.Username, ok = tokenClaim["preferred_username"].(string); !ok {
		return info, sdk.WithStack(errors.New("missing username in OIDC token claim"))
	}

	info.Fullname, _ = tokenClaim["name"].(string)
	if info.Email, ok = tokenClaim["email"].(string); !ok {
		return info, sdk.WithStack(errors.New("missing user's email in OIDC token claim"))
	}

	return info, nil
}
