package corpsso

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"
	"github.com/ovh/cds/sdk/slug"
	"github.com/rockbears/log"
	"gopkg.in/square/go-jose.v2"
)

const (
	TTL        = 15 * time.Minute
	TTLSeconds = int(TTL / time.Second)
)

type CorpSSODriver struct {
	Config SSOConfig
}

type SSOConfig struct {
	Request struct {
		RedirectMethod string
		RedirectURL    string
		Keys           struct {
			RequestSigningKey string
		}
	}
	Token struct {
		SigningKey    string
		KeySigningKey struct {
			KeySigningKey   string
			SigningKeyClaim string
		}
	}
	MFASupportEnabled    bool
	AllowedOrganizations sdk.StringSlice
}

func NewCorpSSODriver(cfg SSOConfig) sdk.Driver {
	var d = CorpSSODriver{cfg}
	return d
}

func (c CorpSSODriver) GetSigninURI(state sdk.AuthSigninConsumerToken) (sdk.AuthDriverSigningRedirect, error) {
	var result sdk.AuthDriverSigningRedirect
	var cfg = c.Config.Request

	state.IssuedAt = time.Now().Unix()

	jws, err := prepareSSORequest(cfg.Keys.RequestSigningKey, state)
	if err != nil {
		return result, sdk.WithStack(err)
	}

	u, err := url.Parse(cfg.RedirectURL)
	if err != nil {
		return result, sdk.WithStack(fmt.Errorf("unable to parse url: %v", err))
	}

	result = sdk.AuthDriverSigningRedirect{
		Method:      cfg.RedirectMethod,
		URL:         u.String(),
		ContentType: "application/x-www-form-urlencoded",
		Body: map[string]string{
			"request": jws,
		},
	}

	return result, nil
}

func (c CorpSSODriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	token := req.String("token")
	if token == "" {
		token = req.String("code")
	}
	if token == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid token for corporate sso signin")
	}
	return nil
}

func (c CorpSSODriver) CheckSigninStateToken(req sdk.AuthConsumerSigninRequest) error {
	// Check if state is given and if its valid
	state, err := req.StringE("state")
	if err != nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid state")
	}

	pk, err := gpg.NewPrivateKeyFromPem(c.Config.Request.Keys.RequestSigningKey, "")
	if err != nil {
		return sdk.WrapError(err, "unable to load private key")
	}

	jwsRequest, err := jose.ParseSigned(state)
	if err != nil {
		return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("unable to parse state: %v", err))
	}

	rawRequest, err := jwsRequest.Verify(pk.GetPubKey())
	if err != nil {
		return sdk.NewError(sdk.ErrUnauthorized, fmt.Errorf("state verification failed: %v", err))
	}

	var signinStateToken sdk.AuthSigninConsumerToken
	if err := sdk.JSONUnmarshal(rawRequest, &signinStateToken); err != nil {
		return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("unable to parse state: %v", err))
	}

	return nil
}

func (c CorpSSODriver) GetUserInfoFromDriver(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var u sdk.AuthDriverUserInfo
	var cfg = c.Config.Token

	// Check if token is given and if its valid
	token := req.String("token")
	if token == "" {
		token = req.String("code")
	}
	if token == "" {
		return u, sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing token value")
	}

	// Parse token
	jws, err := jose.ParseSigned(token)
	if err != nil {
		return u, sdk.NewError(sdk.ErrUnauthorized, fmt.Errorf("token verification failed: %v", err))
	}

	// Define wich public Key we have to take care about to verify the token
	var publicKey interface{}
	switch {
	case cfg.KeySigningKey.KeySigningKey != "":
		ksk, err := gpg.NewPublicKeyFromPem(cfg.KeySigningKey.KeySigningKey)
		if err != nil {
			return u, sdk.WrapError(err, "unable to load public key")
		}

		if len(jws.Signatures) == 0 {
			return u, sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing jws signature in jwt")
		}

		rawSsoKeyBase64, ok := jws.Signatures[0].Protected.ExtraHeaders[jose.HeaderKey(cfg.KeySigningKey.SigningKeyClaim)].(string)
		if !ok {
			return u, sdk.NewErrorFrom(sdk.ErrUnauthorized, "missing signing key claim in jwt")
		}

		rawSsoKey, err := base64.StdEncoding.DecodeString(rawSsoKeyBase64)
		if err != nil {
			return u, sdk.NewErrorFrom(sdk.ErrUnauthorized, "unable to base64 decode raw sso key in jwt: %s", err)
		}

		ssoKey, err := gpg.NewPublicKeyFromData(bytes.NewReader(rawSsoKey))
		if err != nil {
			return u, sdk.NewError(sdk.ErrUnauthorized, err)
		}

		ssoKeySig := ssoKey.IsSignedBy(ksk)
		if ssoKeySig == nil {
			return u, sdk.NewErrorFrom(sdk.ErrUnauthorized, "ssoKey is not signed by KSK")
		}

		if ssoKeySig.SigLifetimeSecs != nil {
			if time.Now().After(ssoKeySig.CreationTime.Add(time.Duration(*ssoKeySig.SigLifetimeSecs) * time.Second)) {
				return u, sdk.NewErrorFrom(sdk.ErrUnauthorized, "ssoKey is expired")
			}
		}

		publicKey = ssoKey.GetKey()

	case cfg.SigningKey != "":
		publicKey, err = gpg.NewPublicKeyFromData(strings.NewReader(cfg.SigningKey))
		if err != nil {
			return u, sdk.WithStack(err)
		}

	default:
		return u, sdk.NewErrorFrom(sdk.ErrUnauthorized, "unable to check token signature")
	}

	rawIssuedToken, err := jws.Verify(publicKey)
	if err != nil {
		return u, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	var itk IssuedToken
	if err := sdk.JSONUnmarshal(rawIssuedToken, &itk); err != nil {
		return u, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	// We consider that the token provided by the Corporate SSO provider is valid for 15 minutes
	if itk.IAT+int64(TTLSeconds) < time.Now().Unix() {
		return u, sdk.NewErrorFrom(sdk.ErrWrongRequest, "expired JWT %s/%s", itk.RemoteUser, itk.TokenID)
	}

	// User organization should be in the list of allowed values
	if !c.Config.AllowedOrganizations.Contains(itk.Organization) {
		return u, sdk.NewErrorFrom(sdk.ErrWrongRequest, "organization not allowed %q", itk.Organization)
	}

	log.Info(ctx, "new session created for remote_user: %v, iat: %v, token_id: %v, mfa: %v", itk.RemoteUser, itk.IAT, itk.TokenID, itk.MFA)

	u.Username = itk.RemoteUser
	if len(u.Username) < 3 && itk.RemoteUsername != "" {
		u.Username = slug.Convert(itk.RemoteUsername)
	}
	u.Fullname = itk.RemoteUsername
	u.ExternalID = itk.RemoteUser
	u.MFA = itk.MFA && c.Config.MFASupportEnabled
	u.Email = itk.Email
	u.ExternalTokenID = itk.TokenID
	u.Organization = itk.Organization

	return u, nil
}

type IssuedToken struct {
	Audience       string   `json:"Audience"`
	RemoteUser     string   `json:"RemoteUser"`
	RemoteUsername string   `json:"RemoteUsername"`
	Email          string   `json:"email"`
	TokenID        string   `json:"TokenId"`
	MFA            bool     `json:"MFA"`
	IAT            int64    `json:"iat"`
	Organization   string   `json:"org"`
	Groups         []string `json:"Groups,omitempty"`
}
