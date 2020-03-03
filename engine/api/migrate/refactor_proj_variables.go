package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"
)

// RefactorProjectVariables .
func RefactorProjectVariables(ctx context.Context, db *gorp.DbMap) error {
	query := "SELECT id FROM project_variable WHERE sig IS NULL"
	rows, err := db.Query(query)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close() // nolint
			return sdk.WithStack(err)
		}
		ids = append(ids, id)
	}

	if err := rows.Close(); err != nil {
		return sdk.WithStack(err)
	}

	var mError = new(sdk.MultiError)
	for _, id := range ids {
		if err := refactorProjectVariables(ctx, db, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.RefactorProjectVariables> unable to migrate project_variable %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func refactorProjectVariables(ctx context.Context, db *gorp.DbMap, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	query := "SELECT project_id, var_name, var_value, cipher_value, var_type FROM project_variable WHERE id = $1 AND sig IS NULL FOR UPDATE SKIP LOCKED"
	var (
		projID      sql.NullInt64
		name        sql.NullString
		typ         sql.NullString
		clearValue  sql.NullString
		cipherValue sql.NullString
	)
	if err := tx.QueryRow(query, id).Scan(&projID, &name, &clearValue, &cipherValue, &typ); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WithStack(err)
	}

	var stringIfValid = func(name string, v sql.NullString) (string, error) {
		if !v.Valid {
			return "", sdk.WithStack(fmt.Errorf("invalid %s data", name))
		}
		return v.String, nil
	}

	var int64IfValid = func(name string, v sql.NullInt64) (int64, error) {
		if !v.Valid {
			return 0, sdk.WithStack(fmt.Errorf("invalid %s data", name))
		}
		return v.Int64, nil
	}

	var v = sdk.Variable{
		ID: id,
	}

	i, err := int64IfValid("projID", projID)
	if err != nil {
		return err
	}
	pID := i

	s, err := stringIfValid("name", name)
	if err != nil {
		return err
	}
	v.Name = s

	s, err = stringIfValid("type", typ)
	if err != nil {
		return err
	}
	v.Type = s

	if !sdk.NeedPlaceholder(v.Type) {
		s, err = stringIfValid("var_value", clearValue)
		if err != nil {
			return err
		}
		v.Value = s
	} else {
		s, err = stringIfValid("cipher_value", cipherValue)
		if err != nil {
			return err
		}

		btes, err := secret.Decrypt([]byte(s))
		if err != nil {
			return err
		}
		v.Value = string(btes)
	}

	if err := project.UpdateVariable(tx, pID, &v, nil, nil); err != nil {
		return err
	}

	log.Info(ctx, "migrate.refactorProjectVariables> variable %s (%d) migrated", v.Name, v.ID)

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
