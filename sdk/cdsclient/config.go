package cdsclient

import (
	"sync"

	jwt "github.com/golang-jwt/jwt"

	"github.com/ovh/cds/sdk"
)

// Config is the configuration data used by the cdsclient interface implementation
type Config struct {
	Host                               string
	CDNHost                            string
	User                               string
	SessionToken                       string
	BuiltinConsumerAuthenticationToken string
	Verbose                            bool
	Retry                              int
	InsecureSkipVerifyTLS              bool
	Mutex                              *sync.Mutex
}

// ProviderConfig is the configuration data used by the cdsclient ProviderClient interface implementation
type ProviderConfig struct {
	Host                  string
	Token                 string
	RequestSecondsTimeout int
	InsecureSkipVerifyTLS bool
}

// ServiceConfig is the configuration data used by the cdsclient interface implementation
type ServiceConfig struct {
	Host                  string
	Token                 string
	RequestSecondsTimeout int
	InsecureSkipVerifyTLS bool
	Hook                  func(Interface) error // This hook is used by unit tests
	Verbose               bool
}

func (c *Config) HasValidSessionToken() bool {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

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
