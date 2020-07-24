package gorpmapping

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/gorpmapper"
)

// Insert value in given db.
func Insert(db gorp.SqlExecutor, i interface{}) error {
	return Mapper.Insert(db, i)
}

func UpdateColumns(db gorp.SqlExecutor, i interface{}, columnFilter gorp.ColumnFilter) error {
	return Mapper.UpdateColumns(db, i, columnFilter)
}

// Update value in given db.
func Update(db gorp.SqlExecutor, i interface{}) error {
	return Mapper.Update(db, i)
}

// Delete value in given db.
func Delete(db gorp.SqlExecutor, i interface{}) error {
	return Mapper.Delete(db, i)
}

// GetAll values from database.
func GetAll(ctx context.Context, db gorp.SqlExecutor, q Query, i interface{}, opts ...GetOptionFunc) error {
	return Mapper.GetAll(ctx, db, q.Query, i, opts...)
}

// Get a value from database.
func Get(ctx context.Context, db gorp.SqlExecutor, q Query, i interface{}, opts ...GetOptionFunc) (bool, error) {
	return Mapper.Get(ctx, db, q.Query, i, opts...)
}

// GetInt a value from database.
func GetInt(db gorp.SqlExecutor, q Query) (int64, error) {
	return Mapper.GetInt(db, q.Query)
}

type GetOptionFunc = gorpmapper.GetOptionFunc

var GetOptions = gorpmapper.GetOptions
