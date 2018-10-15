package auth

import (
	"context"
	"testing"

	"github.com/go-gorp/gorp"
)

func TestLocalAuth(t *testing.T, db *gorp.DbMap) Driver {
	authDriver, err := GetDriver(context.Background(), "local", nil, func() *gorp.DbMap { return db })
	if err != nil {
		panic(err)
	}
	return authDriver
}
