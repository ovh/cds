package local

import (
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

const verifyLocalRegistrationTokenDuration time.Duration = time.Hour * 24

// verifyLocalRegistrationToken contains data for verify signature.
type verifyLocalRegistrationToken struct {
	RegistrationID    string `json:"registration_id"`
	Nonce             string `json:"nonce"`
	IsFirstConnection bool   `json:"is_first_connection,omitempty"`
}

// NewRegistrationToken returns a new token for given registration id.
func NewRegistrationToken(store cache.Store, regID string, isFirstConnection bool) (string, error) {
	payload := verifyLocalRegistrationToken{
		RegistrationID:    regID,
		Nonce:             sdk.UUID(),
		IsFirstConnection: isFirstConnection,
	}

	cacheKey := cache.Key("authentication:registration:verify", payload.RegistrationID)
	if err := store.SetWithDuration(cacheKey, payload.Nonce, verifyLocalRegistrationTokenDuration); err != nil {
		return "", err
	}

	return authentication.SignJWS(payload, verifyLocalRegistrationTokenDuration)
}

// CheckRegistrationToken checks that the given signature is a valid registration token.
func CheckRegistrationToken(store cache.Store, signature string) (string, error) {
	var payload verifyLocalRegistrationToken
	if err := authentication.VerifyJWS(signature, &payload); err != nil {
		return "", err
	}

	cacheKey := cache.Key("authentication:registration:verify", payload.RegistrationID)
	var nonce string
	if ok, _ := store.Get(cacheKey, &nonce); !ok || nonce != payload.Nonce {
		return "", sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given registration token")
	}

	return payload.RegistrationID, nil
}

// CleanVerifyConsumerToken deletes a consumer verify token from cache if exists.
func CleanVerifyConsumerToken(store cache.Store, consumerID string) {
	cacheKey := cache.Key("authentication:registration:verify", consumerID)
	_ = store.Delete(cacheKey)
}
