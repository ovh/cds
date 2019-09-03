package gorpmapping

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
)

// Insert value in given db.
func Insert(db gorp.SqlExecutor, i interface{}) error {
	err := db.Insert(i)
	if e, ok := err.(*pq.Error); ok {
		switch e.Code {
		case ViolateUniqueKeyPGCode:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		case StringDataRightTruncation:
			err = sdk.NewError(sdk.ErrConflict, e)
		}
	}

	if err != nil {
		return sdk.WithStack(err)
	}

	if err := updateEncryptedData(db, i); err != nil {
		return err
	}

	if err := resetEncryptedData(db, i); err != nil {
		return err
	}

	return nil
}

// Update value in given db.
func Update(db gorp.SqlExecutor, i interface{}) error {
	n, err := db.Update(i)
	if e, ok := err.(*pq.Error); ok {
		switch e.Code {
		case ViolateUniqueKeyPGCode:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		case StringDataRightTruncation:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		}
	}
	if err != nil {
		return sdk.WithStack(err)
	}
	if n < 1 {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	if err := updateEncryptedData(db, i); err != nil {
		return err
	}

	if err := resetEncryptedData(db, i); err != nil {
		return err
	}

	return nil
}

// Delete value in given db.
func Delete(db gorp.SqlExecutor, i interface{}) error {
	_, err := db.Delete(i)
	return sdk.WithStack(err)
}

type GetOptionFunc func(db gorp.SqlExecutor, i interface{}) error

var GetOptions = struct {
	WithDecryption GetOptionFunc
}{
	WithDecryption: getEncryptedData,
}

// GetAll values from database.
func GetAll(ctx context.Context, db gorp.SqlExecutor, q Query, i interface{}, opts ...GetOptionFunc) error {
	_, end := observability.Span(ctx, fmt.Sprintf("database.GetAll(%T)", i), observability.Tag("query", q.String()))
	defer end()

	if _, err := db.Select(i, q.query, q.arguments...); err != nil {
		sdk.WithStack(err)
	}

	for _, f := range opts {
		if err := f(db, i); err != nil {
			return err
		}
	}
	return nil
}

// Get a value from database.
func Get(ctx context.Context, db gorp.SqlExecutor, q Query, i interface{}, opts ...GetOptionFunc) (bool, error) {
	_, end := observability.Span(ctx, fmt.Sprintf("database.Get(%T)", i), observability.Tag("query", q.String()))
	defer end()

	if err := db.SelectOne(i, q.query, q.arguments...); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sdk.WithStack(err)
	}

	if err := resetEncryptedData(db, i); err != nil {
		return false, err
	}

	for _, f := range opts {
		if err := f(db, i); err != nil {
			return false, err
		}
	}
	return true, nil
}

// GetInt a value from database.
func GetInt(db gorp.SqlExecutor, q Query) (int64, error) {
	res, err := db.SelectNullInt(q.query, q.arguments...)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, sdk.WithStack(err)
	}
	if !res.Valid {
		return 0, nil
	}

	return res.Int64, nil
}
