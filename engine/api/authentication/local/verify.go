package local

import (
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

const verifyLocalConsumerTokenDuration time.Duration = time.Hour * 24

// verifyLocalConsumerToken contains data for verify signature.
type verifyLocalConsumerToken struct {
	ConsumerID string `json:"consumer_id"`
	Nonce      string `json:"nonce"`
}

// NewVerifyConsumerToken returns a new verify consumer token for given consumer id.
func NewVerifyConsumerToken(store cache.Store, consumerID string) (string, error) {
	payload := verifyLocalConsumerToken{
		ConsumerID: consumerID,
		Nonce:      sdk.UUID(),
	}

	cacheKey := cache.Key("authentication:consumer:verify", payload.ConsumerID)
	if err := store.SetWithDuration(cacheKey, payload.Nonce, verifyLocalConsumerTokenDuration); err != nil {
		return "", err
	}

	return authentication.SignJWS(payload, verifyLocalConsumerTokenDuration)
}

// CheckVerifyConsumerToken checks that the given signature is a valid verify consumer token.
func CheckVerifyConsumerToken(store cache.Store, signature string) (string, error) {
	var payload verifyLocalConsumerToken
	if err := authentication.VerifyJWS(signature, &payload); err != nil {
		return "", err
	}

	cacheKey := cache.Key("authentication:consumer:verify", payload.ConsumerID)
	var nonce string
	if ok := store.Get(cacheKey, &nonce); !ok || nonce != payload.Nonce {
		return "", sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given verify consumer token")
	}

	return payload.ConsumerID, nil
}

// CleanVerifyConsumerToken deletes a consumer verify token from cache if exists.
func CleanVerifyConsumerToken(store cache.Store, consumerID string) {
	cacheKey := cache.Key("authentication:consumer:verify", consumerID)
	store.Delete(cacheKey)
}
