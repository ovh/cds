package keychain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StoreSecret(t *testing.T) {
	err := StoreSecret("http://test.url.local", "username", "password")
	assert.NoError(t, err)

	username, secret, err := GetSecret("http://test.url.local")
	assert.NoError(t, err)
	assert.Equal(t, "username", username)
	assert.Equal(t, "password", secret)
}

func Test_EmptyStoreSecret(t *testing.T) {
	_, _, err := GetSecret("http://test.url.local.empty")
	assert.Error(t, err)
}
