package api

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/sdk"
)

/**
 * apiClaims is a global struct to store all forms of JWT Claims
 * It is either a user, a worker or an hatchery
 * It contains all the permission and groups of a user, worker and hatchery
 */

type AuthenticationConfig struct {
	SigningKey              []byte  `toml:"signingKey" default:"AllYourBase" comment:"JWT signing key" json:"-"`
	UserExpirationDelay     float64 `toml:"userExpirationDelay" default:"-1" comment:"JWT Expiration Delay. -1 for unlimited tokens" json:"userExpirationDelay"`
	WorkerExpirationDelay   float64 `toml:"workerExpirationDelay" default:"-1" comment:"JWT Expiration Delay. -1 for unlimited tokens" json:"workerExpirationDelay"`
	HatcheryExpirationDelay float64 `toml:"hatcheryExpirationDelay" default:"-1" comment:"JWT Expiration Delay. -1 for unlimited tokens" json:"hatcheryExpirationDelay"`
	ServiceExpirationDelay  float64 `toml:"serviceExpirationDelay" default:"-1" comment:"JWT Expiration Delay. -1 for unlimited tokens" json:"serviceExpirationDelay"`
}

type apiClaims struct {
	User     *sdk.User     `json:"user,omitempty"`
	Worker   *sdk.Worker   `json:"worker,omitempty"`
	Hatchery *sdk.Hatchery `json:"hatchery,omitempty"`
	Service  *sdk.Service  `json:"service,omitempty"`
	jwt.StandardClaims
}

func (api *API) claimsForUser(u *sdk.User) jwt.Claims {
	return apiClaims{
		u,
		nil,
		nil,
		nil,
		jwt.StandardClaims{
			Issuer: api.Name,
		},
	}
}

func (api *API) claimsForWorker(w *sdk.Worker) jwt.Claims {
	return apiClaims{
		nil,
		w,
		nil,
		nil,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(12 * time.Hour).Unix(),
			Issuer:    api.Name,
		},
	}
}

func (api *API) claimsForHatchery(h *sdk.Hatchery) jwt.Claims {
	return apiClaims{
		nil,
		nil,
		h,
		nil,
		jwt.StandardClaims{
			Issuer: api.Name,
		},
	}
}

func (api *API) claimsForService(s *sdk.Service) jwt.Claims {
	return apiClaims{
		nil,
		nil,
		nil,
		s,
		jwt.StandardClaims{
			Issuer: api.Name,
		},
	}
}

func (api *API) newToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(api.Config.Auth.AuthenticationConfig.SigningKey)
}

func (api *API) parseToken(tokenString string) (jwt.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &apiClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return api.Config.Auth.AuthenticationConfig.SigningKey, nil
		})
	if err != nil {
		return nil, err

	}
	if !token.Valid {
		return nil, sdk.ErrInvalidToken
	}
	return token.Claims, nil
}

func claimsUser(c jwt.Claims) *sdk.User {
	claims, ok := c.(*apiClaims)
	if !ok {
		return nil
	}
	return claims.User
}

func claimsWorker(c jwt.Claims) *sdk.Worker {
	claims, ok := c.(*apiClaims)
	if !ok {
		return nil
	}
	return claims.Worker
}

func claimsHatchery(c jwt.Claims) *sdk.Hatchery {
	claims, ok := c.(*apiClaims)
	if !ok {
		return nil
	}
	return claims.Hatchery
}

func claimsService(c jwt.Claims) *sdk.Service {
	claims, ok := c.(*apiClaims)
	if !ok {
		return nil
	}
	return claims.Service
}
