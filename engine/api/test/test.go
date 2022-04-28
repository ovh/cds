package test

import (
	"context"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

// SetupPG setup PG DB for test and use gorpmapping singleton's mapper.
func SetupPG(t *testing.T, bootstrapFunc ...test.Bootstrapf) (*test.FakeTransaction, cache.Store) {
	db, _, cache := SetupPGWithFactory(t, bootstrapFunc...)
	return db, cache
}

func SetupPGWithFactory(t *testing.T, bootstrapFunc ...test.Bootstrapf) (*test.FakeTransaction, *database.DBConnectionFactory, cache.Store) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	db, factory, cache, cancel := test.SetupPGToCancel(t, gorpmapping.Mapper, sdk.TypeAPI, bootstrapFunc...)
	t.Cleanup(cancel)

	err := authentication.Init(context.TODO(), "cds-api-test", []authentication.KeyConfig{{Key: string(test.SigningKey)}})
	require.NoError(t, err, "unable to init authentication layer")

	return db, factory, cache
}
