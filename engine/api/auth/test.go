package auth

import (
	"testing"

	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/test"
)

func TestLocalAuth(t *testing.T) Driver {
	authDriver, err := GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	test.NoError(t, err)
	return authDriver
}
