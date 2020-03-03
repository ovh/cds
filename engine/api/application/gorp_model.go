package application

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type dbApplication sdk.Application
type dbApplicationVariableAudit sdk.ApplicationVariableAudit
type dbApplicationKey struct {
	gorpmapping.SignedEntity
	sdk.ApplicationKey
}

func (e dbApplicationKey) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.ApplicationID, e.ID, e.Name}
	return gorpmapping.CanonicalForms{
		"{{print .ApplicationID}}{{print .ID}}{{.Name}}",
	}
}

type dbApplicationVulnerability sdk.Vulnerability

type dbApplicationVariable struct {
	gorpmapping.SignedEntity
	ID            int64  `db:"id"`
	ApplicationID int64  `db:"application_id"`
	Name          string `db:"var_name"`
	ClearValue    string `db:"var_value"`
	CipherValue   string `db:"cipher_value" gorpmapping:"encrypted,ID,Name"`
	Type          string `db:"var_type"`
}

func (e dbApplicationVariable) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.ApplicationID, e.ID, e.Name, e.Type}
	return gorpmapping.CanonicalForms{
		"{{print .ApplicationID}}{{print .ID}}{{.Name}}{{.Type}}",
	}
}

func newDBApplicationVariable(v sdk.Variable, appID int64) dbApplicationVariable {
	if sdk.NeedPlaceholder(v.Type) {
		return dbApplicationVariable{
			ID:            v.ID,
			Name:          v.Name,
			CipherValue:   v.Value,
			Type:          v.Type,
			ApplicationID: appID,
		}
	}
	return dbApplicationVariable{
		ID:            v.ID,
		Name:          v.Name,
		ClearValue:    v.Value,
		Type:          v.Type,
		ApplicationID: appID,
	}
}

func (e dbApplicationVariable) Variable() sdk.Variable {
	if sdk.NeedPlaceholder(e.Type) {
		return sdk.Variable{
			ID:    e.ID,
			Name:  e.Name,
			Value: e.CipherValue,
			Type:  e.Type,
		}
	}

	return sdk.Variable{
		ID:    e.ID,
		Name:  e.Name,
		Value: e.ClearValue,
		Type:  e.Type,
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbApplication{}, "application", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationVariableAudit{}, "application_variable_audit", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationKey{}, "application_key", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationVulnerability{}, "application_vulnerability", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationVariable{}, "application_variable", true, "id"))
}

type sqlApplicationJSON struct {
	Metadata    sql.NullString `db:"metadata"`
	VCSStrategy sql.NullString `db:"vcs_strategy"`
}

// PostGet is a db hook
func (a *dbApplication) PostGet(db gorp.SqlExecutor) error {
	var appContext = sqlApplicationJSON{}
	if err := db.SelectOne(&appContext, "select metadata, vcs_strategy from application where id = $1", a.ID); err != nil {
		return sdk.WrapError(err, "Cannot load metadata and vcs strategy")
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
		ava.VariableAfter = *vAfter
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

	v, err := json.Marshal(ava.VariableAfter)
	if err != nil {
		return err
	}
	vA.Valid = true
	vA.String = string(v)

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
			ava.VariableBefore.Value = sdk.PasswordPlaceholder
		}
	}
	if sdk.NeedPlaceholder(ava.VariableAfter.Type) {
		ava.VariableAfter.Value = sdk.PasswordPlaceholder
	}

	return nil
}
