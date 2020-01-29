package migrate_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/test"

	"github.com/stretchr/testify/require"
)

func TestRefactorAuthenticationAuth(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	mail.Init("", "", "", "", "", false, true)
	require.NoError(t, migrate.RefactorAuthenticationAuth(context.TODO(), db, cache,
		"http://localhost:8081", "http://localhost:8080"))
}

func TestRefactorGroupMembership(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()
	require.NoError(t, migrate.RefactorGroupMembership(context.TODO(), db))
}
