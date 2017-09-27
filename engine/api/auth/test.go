package auth

import (
	"context"
	"testing"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/sessionstore"
)

func TestLocalAuth(t *testing.T, db *gorp.DbMap, o sessionstore.Options) Driver {
	authDriver, err := GetDriver(context.Background(), "local", nil, o, func() *gorp.DbMap { return db })
	if err != nil {
		panic(err)
	}
	return authDriver
}
