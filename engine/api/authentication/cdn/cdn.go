package worker

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/authentication"

	"github.com/ovh/cds/sdk"
)

// SessionDuration the life time of a CDN token request.
var SessionDuration = 1 * time.Hour

// VerifyToken checks token technical validity
func VerifyToken(publicKey *rsa.PublicKey, tokenStr string) (*sdk.CDNRequest, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("cannot verify token with a nil public key")
	}

	var cdnjws sdk.CDNRequest
	if err := authentication.VerifyJWSWithSpecificKey(publicKey, tokenStr, &cdnjws); err != nil {
		return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "Cannot verify jws")
	}

	return &cdnjws, nil
}
