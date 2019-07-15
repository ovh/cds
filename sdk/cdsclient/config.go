package cdsclient

import (
	"github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/sdk"
)

//Config is the configuration data used by the cdsclient interface implementation
type Config struct {
	Host                              string
	User                              string
	SessionToken                      string
	BuitinConsumerAuthenticationToken string
	Verbose                           bool
	Retry                             int
	InsecureSkipVerifyTLS             bool
}

//ProviderConfig is the configuration data used by the cdsclient ProviderClient interface implementation
type ProviderConfig struct {
	Host                  string
	Token                 string
	RequestSecondsTimeout int
	InsecureSkipVerifyTLS bool
}

//ServiceConfig is the configuration data used by the cdsclient interface implementation
type ServiceConfig struct {
	Host                  string
	Token                 string
	RequestSecondsTimeout int
	InsecureSkipVerifyTLS bool
	Hook                  func(Interface) error // This hook is used by unit tests
	Verbose               bool
}

func (c Config) HasValidSessionToken() bool {
	if c.SessionToken == "" {
		return false
	}
	unsafeToken, _, err := new(jwt.Parser).ParseUnverified(c.SessionToken, &sdk.AuthSessionJWTClaims{})
	if err != nil {
		return false
	}

	_, ok := unsafeToken.Claims.(*sdk.AuthSessionJWTClaims)
	if !ok {
		return false
	}

	if err := unsafeToken.Claims.Valid(); err != nil {
		return false
	}

	return true
}
