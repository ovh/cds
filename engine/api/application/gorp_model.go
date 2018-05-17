package application

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

type dbApplication sdk.Application
type dbApplicationVariableAudit sdk.ApplicationVariableAudit
type dbApplicationKey sdk.ApplicationKey

func init() {
	gorpmapping.Register(gorpmapping.New(dbApplication{}, "application", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationVariableAudit{}, "application_variable_audit", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationKey{}, "application_key", false))
}

type sqlApplicationJSON struct {
	Metadata    sql.NullString `db:"metadata"`
	VCSStrategy sql.NullString `db:"vcs_strategy"`
}

// PostGet is a db hook
func (a *dbApplication) PostGet(db gorp.SqlExecutor) error {
	var appContext = sqlApplicationJSON{}
	if err := db.SelectOne(&appContext, "select metadata, vcs_strategy from application where id = $1", a.ID); err != nil {
		return sdk.WrapError(err, "dbApplication>PostGet Cannot load metadata and vcs strategy")
	}

	if appContext.Metadata.Valid {
		metadata := sdk.Metadata{}
		if err := json.Unmarshal([]byte(appContext.Metadata.String), &metadata); err != nil {
			return err
		}
		a.Metadata = metadata
	}
	if appContext.VCSStrategy.Valid {
		vcs := sdk.RepositoryStrategy{}
		if err := json.Unmarshal([]byte(appContext.VCSStrategy.String), &vcs); err != nil {
			return err
		}
		a.RepositoryStrategy = vcs
	}
	return nil
}

// PostUpdate is a db hook
func (a *dbApplication) PostUpdate(db gorp.SqlExecutor) error {
	b, err := json.Marshal(a.Metadata)
	if err != nil {
		return err
	}

	v, err := json.Marshal(a.RepositoryStrategy)
	if err != nil {
		return err
	}

	if _, err := db.Exec("update application set metadata = $2, vcs_strategy = $3 where id = $1", a.ID, b, v); err != nil {
		return err
	}
	return nil
}

// PostInsert is a db hook
func (a *dbApplication) PostInsert(db gorp.SqlExecutor) error {
	return a.PostUpdate(db)
}

// PostGet is a db hook
func (ava *dbApplicationVariableAudit) PostGet(db gorp.SqlExecutor) error {
	var before, after sql.NullString
	query := "SELECT variable_before, variable_after from application_variable_audit WHERE id = $1"
	if err := db.QueryRow(query, ava.ID).Scan(&before, &after); err != nil {
		return err
	}

	if before.Valid {
		vBefore := &sdk.Variable{}
		if err := json.Unmarshal([]byte(before.String), vBefore); err != nil {
			return err
		}
		if sdk.NeedPlaceholder(vBefore.Type) {
			vBefore.Value = sdk.PasswordPlaceholder
		}
		ava.VariableBefore = vBefore

	}

	if after.Valid {
		vAfter := &sdk.Variable{}
		if err := json.Unmarshal([]byte(after.String), vAfter); err != nil {
			return err
		}
		if sdk.NeedPlaceholder(vAfter.Type) {
			vAfter.Value = sdk.PasswordPlaceholder
		}
		ava.VariableAfter = vAfter
	}

	return nil
}

// PostUpdate is a db hook
func (ava *dbApplicationVariableAudit) PostUpdate(db gorp.SqlExecutor) error {
	var vB, vA sql.NullString

	if ava.VariableBefore != nil {
		v, err := json.Marshal(ava.VariableBefore)
		if err != nil {
			return err
		}
		vB.Valid = true
		vB.String = string(v)
	}

	if ava.VariableAfter != nil {
		v, err := json.Marshal(ava.VariableAfter)
		if err != nil {
			return err
		}
		vA.Valid = true
		vA.String = string(v)
	}

	query := "update application_variable_audit set variable_before = $2, variable_after = $3 where id = $1"
	if _, err := db.Exec(query, ava.ID, vB, vA); err != nil {
		return err
	}
	return nil
}

// PostInsert is a db hook
func (ava *dbApplicationVariableAudit) PostInsert(db gorp.SqlExecutor) error {
	return ava.PostUpdate(db)
}

// PreInsert
func (ava *dbApplicationVariableAudit) PreInsert(s gorp.SqlExecutor) error {
	if ava.VariableBefore != nil {
		if sdk.NeedPlaceholder(ava.VariableBefore.Type) {
			secret, err := secret.Encrypt([]byte(ava.VariableBefore.Value))
			if err != nil {
				return err
			}
			ava.VariableBefore.Value = base64.StdEncoding.EncodeToString(secret)
		}
	}
	if ava.VariableAfter != nil {
		if sdk.NeedPlaceholder(ava.VariableAfter.Type) {
			var err error
			secret, err := secret.Encrypt([]byte(ava.VariableAfter.Value))
			if err != nil {
				return err
			}
			ava.VariableAfter.Value = base64.StdEncoding.EncodeToString(secret)
		}
	}
	return nil
}
