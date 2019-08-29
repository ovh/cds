package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoreToken(t *testing.T) {
	cdsContext := CDSContext{
		Context:               "My Context Name2",
		Host:                  "http://localhost:8080/test",
		InsecureSkipVerifyTLS: false,
		SessionToken:          "the-token-test",
		User:                  "the-username-test",
	}
	err := storeToken(cdsContext.Context, cdsContext.SessionToken)
	assert.NoError(t, err)

	token, err := cdsContext.getToken(cdsContext.Context)
	assert.NoError(t, err)
	assert.Equal(t, cdsContext.SessionToken, token)

	// store another user for the same context -> we rewrite the entry in keychain
	cdsContext.SessionToken = "the-token-test2"
	cdsContext.User = "the-username-test2"
	err = storeToken(cdsContext.Context, cdsContext.SessionToken)
	assert.NoError(t, err)

	token, err = cdsContext.getToken(cdsContext.Context)
	assert.NoError(t, err)
	assert.Equal(t, cdsContext.SessionToken, token)

}
