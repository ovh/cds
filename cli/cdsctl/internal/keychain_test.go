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
		Session:               "the-token-test",
		Token:                 "the-token-consumer-test",
	}
	err := storeTokens(cdsContext.Context, ContextTokens{Session: cdsContext.Session, Token: cdsContext.Token})
	assert.NoError(t, err)

	tokens, err := cdsContext.getTokens(cdsContext.Context)
	assert.NoError(t, err)
	assert.Equal(t, cdsContext.Session, tokens.Session)
	assert.Equal(t, cdsContext.Token, tokens.Token)

	// store another user for the same context -> we rewrite the entry in keychain
	cdsContext.Session = "the-token-test2"
	cdsContext.Token = "the-token-consumer-test2"
	err = storeTokens(cdsContext.Context, ContextTokens{Session: cdsContext.Session, Token: cdsContext.Token})
	assert.NoError(t, err)

	tokens, err = cdsContext.getTokens(cdsContext.Context)
	assert.NoError(t, err)
	assert.Equal(t, cdsContext.Session, tokens.Session)
	assert.Equal(t, cdsContext.Token, tokens.Token)

}
