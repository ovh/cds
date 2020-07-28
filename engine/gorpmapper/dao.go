package gorpmapper

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func checkDatabase(db gorp.SqlExecutor) error {
	if db == nil {
		return sdk.NewErrorFrom(sdk.ErrServiceUnavailable, "database unavailabe")
	}
	return nil
}

// Insert value in given db.
func (m *Mapper) Insert(db gorp.SqlExecutor, i interface{}) error {
	if err := checkDatabase(db); err != nil {
		return err
	}

	if _, has := m.GetTableMapping(i); !has {
		return sdk.WithStack(fmt.Errorf("unkown entity %T", i))
	}

	err := db.Insert(i)
	if e, ok := err.(*pq.Error); ok {
		switch e.Code {
		case ViolateUniqueKeyPGCode:
			err = sdk.NewError(sdk.ErrConflictData, e)
		case StringDataRightTruncation:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		}
	}

	if err != nil {
		return sdk.WithStack(err)
	}

	if err := m.updateEncryptedData(db, i); err != nil {
		return err
	}

	if err := m.resetEncryptedData(db, i); err != nil {
		return err
	}

	return nil
}

func (m *Mapper) UpdateColumns(db gorp.SqlExecutor, i interface{}, columnFilter gorp.ColumnFilter) error {
	if err := checkDatabase(db); err != nil {
		return err
	}
	mapping, has := m.GetTableMapping(i)
	if !has {
		return sdk.WithStack(fmt.Errorf("unkown entity %T", i))
	}

	val := reflect.ValueOf(i)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var hasPlaceHolder bool
	for _, f := range mapping.EncryptedFields {
		// Reset the field to the decrypted value if the value is set to the placeholder
		field := val.FieldByName(f.Name)
		if field.Interface() == sdk.PasswordPlaceholder {
			hasPlaceHolder = true
			break
		}
	}

	// If the data has encrypted data
	if mapping.EncryptedEntity && hasPlaceHolder {
		id := reflectFindValueByTag(i, "db", mapping.Keys[0])
		entityName := fmt.Sprintf("%T", reflect.ValueOf(i).Elem().Interface())

		// Reload and decrypt the old tuple from the database
		tuple, err := m.LoadTupleByPrimaryKey(db, entityName, id, GetOptions.WithDecryption)
		if err != nil {
			return err
		}

		valTuple := reflect.ValueOf(tuple)
		if valTuple.Kind() == reflect.Ptr {
			valTuple = valTuple.Elem()
		}

		for _, f := range mapping.EncryptedFields {
			// Reset the field to the decrypted value if the value is set to the placeholder
			field := val.FieldByName(f.Name)
			if field.Interface() == sdk.PasswordPlaceholder {
				oldVal := valTuple.FieldByName(f.Name)
				field.Set(oldVal)
			}
		}
	}

	n, err := db.UpdateColumns(columnFilter, i)
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

	if err := m.updateEncryptedData(db, i); err != nil {
		return err
	}

	if err := m.resetEncryptedData(db, i); err != nil {
		return err
	}

	return nil
}

// Update value in given db.
func (m *Mapper) Update(db gorp.SqlExecutor, i interface{}) error {
	if err := checkDatabase(db); err != nil {
		return err
	}
	return m.UpdateColumns(db, i, acceptAllFilter)
}

// Delete value in given db.
func (m *Mapper) Delete(db gorp.SqlExecutor, i interface{}) error {
	if err := checkDatabase(db); err != nil {
		return err
	}

	_, err := db.Delete(i)
	return sdk.WithStack(err)
}

func acceptAllFilter(col *gorp.ColumnMap) bool {
	return true
}

type GetOptionFunc func(m *Mapper, db gorp.SqlExecutor, i interface{}) error

var GetOptions = struct {
	WithDecryption GetOptionFunc
}{
	WithDecryption: getEncryptedData,
}

// GetAll values from database.
func (m *Mapper) GetAll(ctx context.Context, db gorp.SqlExecutor, q Query, i interface{}, opts ...GetOptionFunc) error {
	if err := checkDatabase(db); err != nil {
		return err
	}

	_, end := telemetry.Span(ctx, fmt.Sprintf("database.GetAll(%T)", i), telemetry.Tag("query", q.String()))
	defer end()

	if _, err := db.Select(i, q.Query, q.Arguments...); err != nil {
		return sdk.WithStack(err)
	}

	v := sdk.ValueFromInterface(i)

	switch reflect.TypeOf(v.Interface()).Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			var dest reflect.Value
			if v.Index(i).Kind() == reflect.Ptr {
				dest = v.Index(i)
			} else {
				dest = reflect.NewAt(reflect.TypeOf(v.Index(i).Interface()), unsafe.Pointer(v.Index(i).UnsafeAddr()))
			}
			if err := m.resetEncryptedData(db, dest.Interface()); err != nil {
				return err
			}
		}
	default:
		if err := m.resetEncryptedData(db, i); err != nil {
			return err
		}
	}

	for _, f := range opts {
		if err := f(m, db, i); err != nil {
			return err
		}
	}
	return nil
}

// Get a value from database.
func (m *Mapper) Get(ctx context.Context, db gorp.SqlExecutor, q Query, i interface{}, opts ...GetOptionFunc) (bool, error) {
	if err := checkDatabase(db); err != nil {
		return false, err
	}

	_, end := telemetry.Span(ctx, fmt.Sprintf("database.Get(%T)", i), telemetry.Tag("query", q.String()))
	defer end()

	if err := db.SelectOne(i, q.Query, q.Arguments...); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sdk.WithStack(err)
	}

	if err := m.resetEncryptedData(db, i); err != nil {
		return false, err
	}

	for _, f := range opts {
		if err := f(m, db, i); err != nil {
			return false, err
		}
	}
	return true, nil
}

// GetInt a value from database.
func (m *Mapper) GetInt(db gorp.SqlExecutor, q Query) (int64, error) {
	if err := checkDatabase(db); err != nil {
		return 0, err
	}

	res, err := db.SelectNullInt(q.Query, q.Arguments...)
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

func (m *Mapper) dbMappingPKey(i interface{}) (string, string, interface{}, error) {
	mapping, has := m.GetTableMapping(i)
	if !has {
		return "", "", nil, sdk.WithStack(fmt.Errorf("unkown entity %T", i))
	}

	if len(mapping.Keys) > 1 {
		return "", "", nil, sdk.WithStack(errors.New("multiple primary key not supported"))
	}

	id := reflectFindValueByTag(i, "db", mapping.Keys[0])

	return mapping.Name, mapping.Keys[0], id, nil
}
