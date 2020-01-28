package migrate

import (
	"context"

	"github.com/ovh/cds/engine/api/cache"

	"github.com/go-gorp/gorp"
)

// RefactorAuthenticationUser migrates the old user table to the new user tables.
func RefactorAuthenticationUser(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	// TODO remove
	return nil
}

func refactorAuthenticationUser(ctx context.Context, db *gorp.DbMap, store cache.Store, u interface{}) error {
	// TODO remove
	return nil
}
