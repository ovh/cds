package local

import (
	"context"
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
func NewResetConsumerToken(ctx context.Context, store cache.Store, consumerID string) (string, error) {
	var now = time.Now()
	payload := resetLocalConsumerToken{
		ConsumerID: consumerID,
		Nonce:      now.Unix(),
	}

	cacheKey := cache.Key("authentication:consumer:reset", consumerID)
	if err := store.SetWithDuration(ctx, cacheKey, payload.Nonce, resetLocalConsumerTokenDuration); err != nil {
		return "", err
	}

	return authentication.SignJWS(payload, now, resetLocalConsumerTokenDuration)
}

// CheckResetConsumerToken checks that the given signature is a valid reset consumer token.
func CheckResetConsumerToken(ctx context.Context, store cache.Store, signature string) (string, error) {
	var payload resetLocalConsumerToken
	if err := authentication.VerifyJWS(signature, &payload); err != nil {
		return "", err
	}

	cacheKey := cache.Key("authentication:consumer:reset", payload.ConsumerID)
	var nonce int64
	if ok, _ := store.Get(ctx, cacheKey, &nonce); !ok || nonce != payload.Nonce {
		return "", sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given reset consumer token")
	}

	return payload.ConsumerID, nil
}

// CleanResetConsumerToken deletes a consumer reset token from cache if exists.
func CleanResetConsumerToken(ctx context.Context, store cache.Store, consumerID string) {
	cacheKey := cache.Key("authentication:consumer:reset", consumerID)
	_ = store.Delete(ctx, cacheKey)
}
