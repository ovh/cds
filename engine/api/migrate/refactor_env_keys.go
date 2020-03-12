package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"
)

// RefactorEnvironmentKeys .
func RefactorEnvironmentKeys(ctx context.Context, db *gorp.DbMap) error {
	query := "SELECT id FROM environment_key WHERE sig IS NULL"
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
		if err := refactorEnvironmentKeys(ctx, db, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.RefactorEnvironmentKeys> unable to migrate application_key %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func refactorEnvironmentKeys(ctx context.Context, db *gorp.DbMap, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	query := "SELECT environment_id, name, type, public, key_id, private FROM environment_key WHERE id = $1 AND sig IS NULL FOR UPDATE SKIP LOCKED"
	var (
		environmentID sql.NullInt64
		name          sql.NullString
		typ           sql.NullString
		public        sql.NullString
		keyID         sql.NullString
		private       sql.NullString
	)
	if err := tx.QueryRow(query, id).Scan(&environmentID, &name, &typ, &public, &keyID, &private); err != nil {
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

	var k = sdk.EnvironmentKey{
		ID: id,
	}

	i, err := int64IfValid("environmentID", environmentID)
	if err != nil {
		return err
	}
	k.EnvironmentID = i

	s, err := stringIfValid("name", name)
	if err != nil {
		return err
	}
	k.Name = s

	s, err = stringIfValid("type", typ)
	if err != nil {
		return err
	}
	k.Type = s

	s, err = stringIfValid("public", public)
	if err != nil {
		return err
	}
	k.Public = s

	s, err = stringIfValid("private", private)
	if err != nil {
		return err
	}

	btes, err := secret.Decrypt([]byte(s))
	if err != nil {
		return err
	}
	k.Private = string(btes)

	s, _ = stringIfValid("keyID", keyID)
	k.KeyID = s

	if err := environment.UpdateKey(tx, &k); err != nil {
		return err
	}

	log.Info(ctx, "migrate.RefactorEnvironmentKeys> key %s (%d) migrated", k.Name, k.ID)

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
