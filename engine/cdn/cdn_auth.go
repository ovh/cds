package cdn

import (
	"crypto/rsa"
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
)

// VerifyToken checks token technical validity
func VerifyToken(publicKey *rsa.PublicKey, tokenStr string) (*sdk.CDNRequest, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("cannot verify token with a nil public key")
	}

	var cdnjws sdk.CDNRequest
	if err := jws.VerifyJWSWithSpecificKey(publicKey, tokenStr, &cdnjws); err != nil {
		return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "Cannot verify jws")
	}

	return &cdnjws, nil
}
