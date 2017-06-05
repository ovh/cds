package auth

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/sessionstore"
)

func TestLocalAuth(t *testing.T) Driver {
	authDriver, err := GetDriver(context.Background(), "local", nil, sessionstore.Options{Mode: "local"})
	if err != nil {
		panic(err)
	}
	return authDriver
}
