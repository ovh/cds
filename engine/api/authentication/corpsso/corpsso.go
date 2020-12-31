package corpsso

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	jose "gopkg.in/square/go-jose.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"
)

const (
	TTL        = 15 * time.Minute
	TTLSeconds = int(TTL / time.Second)
)

var (
	_ sdk.AuthDriverWithRedirect         = new(authDriver)
	_ sdk.AuthDriverWithSigninStateToken = new(authDriver)
)

type authDriver struct {
	Config Config
}

type Config struct {
	Request struct {
		RedirectMethod string `json:"redirect_method"`
		RedirectURL    string `json:"redirect_url"`
		Keys           struct {
			RequestSigningKey string `json:"request_signing_key"`
		} `json:"keys"`
	} `json:"request"`
	Token struct {
		SigningKey    string `json:"token_signing_key"`
		KeySigningKey struct {
			KeySigningKey   string `json:"public_signing_key"`
			SigningKeyClaim string `json:"signing_key_claim"`
		} `json:"key_signing_key,omitempty"`
	} `json:"token"`
	MailDomain        string `json:"mail_domain"`
	MFASupportEnabled bool   `json:"mfa_support"`
}

func NewDriver(cfg Config) sdk.AuthDriver {
	var d = authDriver{cfg}
	return d
}

func (d authDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerCorporateSSO,
		SignupDisabled: false,
	}
}

func (d authDriver) GetSigninURI(state sdk.AuthSigninConsumerToken) (sdk.AuthDriverSigningRedirect, error) {
	var result sdk.AuthDriverSigningRedirect
	var cfg = d.Config.Request

	state.IssuedAt = time.Now().Unix()

	jws, err := prepareRequest(cfg.Keys.RequestSigningKey, state)
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

func (d authDriver) GetSessionDuration(req sdk.AuthDriverUserInfo, c sdk.AuthConsumer) time.Duration {
	if d.Config.MFASupportEnabled && req.MFA {
		return time.Hour
	}
	return 24 * time.Hour
}

func (d authDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	token, ok := req["token"]
	if !ok {
		token = req["code"]
	}
	if token == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid token for corporate sso signin")
	}
	return nil
}

func (d authDriver) CheckSigninStateToken(req sdk.AuthConsumerSigninRequest) error {
	// Check if state is given and if its valid
	state, okState := req["state"]
	if !okState {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing state value")
	}

	pk, err := gpg.NewPrivateKeyFromPem(d.Config.Request.Keys.RequestSigningKey, "")
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
	if err := json.Unmarshal(rawRequest, &signinStateToken); err != nil {
		return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("unable to parse state: %v", err))
	}

	return nil
}

func (d authDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var u sdk.AuthDriverUserInfo
	var cfg = d.Config.Token

	// Check if token is given and if its valid
	token, okToken := req["token"]
	if !okToken {
		token = req["code"]
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

	var itk issuedToken
	if err := json.Unmarshal(rawIssuedToken, &itk); err != nil {
		return u, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	// We consider that the token provided by the Corporate SSO provider is valid for 15 minutes
	if itk.IAT+int64(TTLSeconds) < time.Now().Unix() {
		return u, sdk.NewErrorFrom(sdk.ErrWrongRequest, "expired JWT %s/%s", itk.RemoteUser, itk.TokenID)
	}

	u.Username = itk.RemoteUser
	u.ExternalID = itk.RemoteUser
	u.MFA = itk.MFA
	u.Email = itk.RemoteUser + "@" + d.Config.MailDomain

	return u, nil
}

type issuedToken struct {
	Audience   string `json:"Audience"`
	RemoteUser string `json:"RemoteUser"`
	TokenID    string `json:"TokenId"`
	MFA        bool   `json:"MFA"`
	IAT        int64  `json:"iat"`
}
