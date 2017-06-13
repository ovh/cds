package keychain

import (
	"testing"

	"os"

	"github.com/stretchr/testify/assert"
)

func Test_checkLibSecretAvailable(t *testing.T) {
	checkLibSecretAvailable()
}

func Test_StoreSecret(t *testing.T) {
	//If we are runnig inside a CDS worker. Skip the test
	if os.Getenv("CDS_KEY") != "" {
		t.SkipNow()
	}

	err := StoreSecret("http://test.url.local", "username", "password")
	assert.NoError(t, err)

	username, secret, err := GetSecret("http://test.url.local")
	assert.NoError(t, err)
	assert.Equal(t, "username", username)
	assert.Equal(t, "password", secret)
}

func Test_EmptyStoreSecret(t *testing.T) {
	//If we are runnig inside a CDS worker. Skip the test
	if os.Getenv("CDS_KEY") != "" {
		t.SkipNow()
	}

	_, _, err := GetSecret("http://test.url.local.empty")
	assert.Error(t, err)
}
