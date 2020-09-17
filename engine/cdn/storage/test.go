package storage

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"testing"
)

func DeleteUnit(t *testing.T, m *gorpmapper.Mapper, db gorp.SqlExecutor, u *sdk.CDNUnit) {
	unitDB := toUnitDB(*u)
	require.NoError(t, m.Delete(db, unitDB))
}
