package authentication

import (
	"time"

	"github.com/ovh/cds/sdk"
)

const (
	signinStateDuration    time.Duration = time.Minute * 5
	verifyConsumerDuration time.Duration = time.Hour * 24
)

type signaturePayloadType string

const (
	signinStatePayload    signaturePayloadType = "signin-state"
	verifyConsumerPayload signaturePayloadType = "verify-consumer"
)

// defaultSignaturePayload contains minimal fields for a jws signature payload.
type defaultSignaturePayload struct {
	Type   signaturePayloadType `json:"type"`
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
func NewVerifyConsumerToken(consumerID string) (string, error) {
	state := verifyConsumer{
		defaultSignaturePayload: defaultSignaturePayload{
			Type:   signinStatePayload,
			Expire: time.Now().Add(verifyConsumerDuration).Unix(),
		},
		ConsumerID: consumerID,
	}
	return SignJWS(state)
}

// CheckVerifyConsumerToken checks that the given signature is a valid verify consumer token.
func CheckVerifyConsumerToken(signature string) (string, error) {
	var state verifyConsumer
	if err := VerifyJWS(signature, &state); err != nil {
		return "", err
	}
	if state.Type != verifyConsumerPayload || state.Expire < time.Now().Unix() {
		return "", sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given signin state")
	}
	return state.ConsumerID, nil
}
