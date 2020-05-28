package migrate_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/test"
	"github.com/stretchr/testify/require"
)

func TestRefactorGroupMembership(t *testing.T) {
	db, _ := test.SetupPG(t)

	require.NoError(t, migrate.RefactorGroupMembership(context.TODO(), db))
}
