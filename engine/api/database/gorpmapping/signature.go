package gorpmapping

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/gorpmapper"
)

// InsertAndSign a data in database, given data should implement canonicaller interface.
func InsertAndSign(ctx context.Context, db gorpmapper.SqlExecutorWithTx, i gorpmapper.Canonicaller) error {
	return Mapper.InsertAndSign(ctx, db, i)
}

// UpdateAndSign a data in database, given data should implement canonicaller interface.
func UpdateAndSign(ctx context.Context, db gorpmapper.SqlExecutorWithTx, i gorpmapper.Canonicaller) error {
	return Mapper.UpdateAndSign(ctx, db, i)
}

// UpdateColumnsAndSign a data in database, given data should implement canonicaller interface.
func UpdateColumnsAndSign(ctx context.Context, db gorpmapper.SqlExecutorWithTx, i gorpmapper.Canonicaller, colFilter gorp.ColumnFilter) error {
	return Mapper.UpdateColumnsAndSign(ctx, db, i, colFilter)
}

func CheckSignature(i gorpmapper.Canonicaller, sig []byte) (bool, error) {
	return Mapper.CheckSignature(i, sig)
}
