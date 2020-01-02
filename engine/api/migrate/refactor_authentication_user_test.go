package migrate_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/test"

	"github.com/stretchr/testify/require"
)

func TestRefactorAuthenticationUser(t *testing.T) {
	db, cache, _ := test.SetupPG(t)
	mail.Init("", "", "", "", "", false, true)
	require.NoError(t, migrate.RefactorAuthenticationUser(context.TODO(), db, cache))
}
