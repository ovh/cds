package auth

import (
	"context"
	"testing"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/sessionstore"
)

func TestLocalAuth(t *testing.T, db *gorp.DbMap) Driver {
	authDriver, err := GetDriver(context.Background(), "local", nil, sessionstore.Options{Mode: "local"}, func() *gorp.DbMap { return db })
	if err != nil {
		panic(err)
	}
	return authDriver
}
