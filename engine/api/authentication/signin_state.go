package authentication

import (
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// Duration for authentication tokens.
const (
	signinStateDuration    time.Duration = time.Minute * 5
	verifyConsumerDuration time.Duration = time.Hour * 24
	resetConsumerDuration  time.Duration = time.Hour * 1
)

type signaturePayloadType string

const (
	signinStatePayload    signaturePayloadType = "signin-state"
	verifyConsumerPayload signaturePayloadType = "verify-consumer"
	resetConsumerPayload  signaturePayloadType = "reset-consumer"
)

// defaultSignaturePayload contains minimal fields for a jws signature payload.
type defaultSignaturePayload struct {
	Type   signaturePayloadType `json:"type"`
	Nonce  string               `json:"nonce"`
	Expire int64                `json:"expire"`
}

type signinState struct {
	defaultSignaturePayload
	Origin string `json:"origin"`
}

type verifyConsumer struct {
	defaultSignaturePayload
	ConsumerID string `json:"consumer_id"`
}

type resetConsumer struct {
	defaultSignaturePayload
	ConsumerID string `json:"consumer_id"`
}

// NewSigninStateToken returns a jws used for signin request.
func NewSigninStateToken(origin string) (string, error) {
	state := signinState{
		defaultSignaturePayload: defaultSignaturePayload{
			Type:   signinStatePayload,
			Expire: time.Now().Add(signinStateDuration).Unix(),
		},
		Origin: origin,
	}
	return SignJWS(state)
}

// CheckSigninStateToken checks if a given signature is a valid signin state.
func CheckSigninStateToken(signature string) error {
	var state signinState
	if err := VerifyJWS(signature, &state); err != nil {
		return err
	}
	if state.Type != signinStatePayload || state.Expire < time.Now().Unix() {
		return sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given signin state")
	}
	return nil
}

// NewVerifyConsumerToken returns a new verify consumer token for given consumer id.
func NewVerifyConsumerToken(store cache.Store, consumerID string) (string, error) {
	state := verifyConsumer{
		defaultSignaturePayload: defaultSignaturePayload{
			Type:   verifyConsumerPayload,
			Nonce:  sdk.UUID(),
			Expire: time.Now().Add(verifyConsumerDuration).Unix(),
		},
		ConsumerID: consumerID,
	}

	cacheKey := cache.Key("authentication:consumer:verify", state.ConsumerID)
	if err := store.SetWithDuration(cacheKey, state.Nonce, verifyConsumerDuration); err != nil {
		return "", err
	}

	return SignJWS(state)
}

// CheckVerifyConsumerToken checks that the given signature is a valid verify consumer token.
func CheckVerifyConsumerToken(store cache.Store, signature string) (string, error) {
	var state verifyConsumer
	if err := VerifyJWS(signature, &state); err != nil {
		return "", err
	}
	if state.Type != verifyConsumerPayload || state.Expire < time.Now().Unix() {
		return "", sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given verify consumer token")
	}

	cacheKey := cache.Key("authentication:consumer:verify", state.ConsumerID)

	var nonce string
	if ok := store.Get(cacheKey, &nonce); !ok || nonce != state.Nonce {
		return "", sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given verify consumer token")
	}

	return state.ConsumerID, nil
}

// CleanVerifyConsumerToken deletes a consumer verify token from cache if exists.
func CleanVerifyConsumerToken(store cache.Store, consumerID string) {
	cacheKey := cache.Key("authentication:consumer:verify", consumerID)
	store.Delete(cacheKey)
}

// NewResetConsumerToken returns a new reset consumer token for given consumer id.
func NewResetConsumerToken(store cache.Store, consumerID string) (string, error) {
	state := resetConsumer{
		defaultSignaturePayload: defaultSignaturePayload{
			Type:   resetConsumerPayload,
			Nonce:  sdk.UUID(),
			Expire: time.Now().Add(verifyConsumerDuration).Unix(),
		},
		ConsumerID: consumerID,
	}

	cacheKey := cache.Key("authentication:consumer:reset", consumerID)
	if err := store.SetWithDuration(cacheKey, state.Nonce, verifyConsumerDuration); err != nil {
		return "", err
	}

	return SignJWS(state)
}

// CheckResetConsumerToken checks that the given signature is a valid reset consumer token.
func CheckResetConsumerToken(store cache.Store, signature string) (string, error) {
	var state resetConsumer
	if err := VerifyJWS(signature, &state); err != nil {
		return "", err
	}
	if state.Type != resetConsumerPayload || state.Expire < time.Now().Unix() {
		return "", sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given reset consumer token")
	}

	cacheKey := cache.Key("authentication:consumer:reset", state.ConsumerID)

	var nonce string
	if ok := store.Get(cacheKey, &nonce); !ok || nonce != state.Nonce {
		return "", sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given reset consumer token")
	}

	return state.ConsumerID, nil
}

// CleanResetConsumerToken deletes a consumer reset token from cache if exists.
func CleanResetConsumerToken(store cache.Store, consumerID string) {
	cacheKey := cache.Key("authentication:consumer:reset", consumerID)
	store.Delete(cacheKey)
}
