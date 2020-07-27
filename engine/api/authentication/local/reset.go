package local

import (
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

const resetLocalConsumerTokenDuration time.Duration = time.Hour * 1

type resetLocalConsumerToken struct {
	ConsumerID string `json:"consumer_id"`
	Nonce      int64  `json:"nonce"`
}

// NewResetConsumerToken returns a new reset consumer token for given consumer id.
func NewResetConsumerToken(store cache.Store, consumerID string) (string, error) {
	payload := resetLocalConsumerToken{
		ConsumerID: consumerID,
		Nonce:      time.Now().Unix(),
	}

	cacheKey := cache.Key("authentication:consumer:reset", consumerID)
	if err := store.SetWithDuration(cacheKey, payload.Nonce, resetLocalConsumerTokenDuration); err != nil {
		return "", err
	}

	return authentication.SignJWS(payload, resetLocalConsumerTokenDuration)
}

// CheckResetConsumerToken checks that the given signature is a valid reset consumer token.
func CheckResetConsumerToken(store cache.Store, signature string) (string, error) {
	var payload resetLocalConsumerToken
	if err := authentication.VerifyJWS(signature, &payload); err != nil {
		return "", err
	}

	cacheKey := cache.Key("authentication:consumer:reset", payload.ConsumerID)
	var nonce int64
	if ok, _ := store.Get(cacheKey, &nonce); !ok || nonce != payload.Nonce {
		return "", sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given reset consumer token")
	}

	return payload.ConsumerID, nil
}

// CleanResetConsumerToken deletes a consumer reset token from cache if exists.
func CleanResetConsumerToken(store cache.Store, consumerID string) {
	cacheKey := cache.Key("authentication:consumer:reset", consumerID)
	_ = store.Delete(cacheKey)
}
