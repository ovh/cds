package auth

import (
	"testing"

	"github.com/ovh/cds/engine/api/sessionstore"
)

func TestLocalAuth(t *testing.T) Driver {
	authDriver, err := GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	NoError(t, err)
	return authDriver
}
