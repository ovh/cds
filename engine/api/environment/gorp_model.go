package environment

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

type dbEnvironmentVariableAudit sdk.EnvironmentVariableAudit

type dbEnvironmentKey struct {
	gorpmapping.SignedEntity
	sdk.EnvironmentKey
}

func (e dbEnvironmentKey) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.EnvironmentID, e.ID, e.Name}
	return gorpmapping.CanonicalForms{
		"{{print .EnvironmentID}}{{print .ID}}{{.Name}}",
	}
}

type dbEnvironmentVariable struct {
	gorpmapping.SignedEntity
	ID            int64  `db:"id"`
	EnvironmentID int64  `db:"environment_id"`
	Name          string `db:"name"`
	ClearValue    string `db:"value"`
	CipherValue   string `db:"cipher_value" gorpmapping:"encrypted,ID,Name"`
	Type          string `db:"type"`
}

func (e dbEnvironmentVariable) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.EnvironmentID, e.ID, e.Name, e.Type}
	return gorpmapping.CanonicalForms{
		"{{print .EnvironmentID}}{{print .ID}}{{.Name}}{{.Type}}",
	}
}

func newdbEnvironmentVariable(v sdk.EnvironmentVariable, projID int64) dbEnvironmentVariable {
	if sdk.NeedPlaceholder(v.Type) {
		return dbEnvironmentVariable{
			ID:            v.ID,
			Name:          v.Name,
			CipherValue:   v.Value,
			Type:          v.Type,
			EnvironmentID: projID,
		}
	}
	return dbEnvironmentVariable{
		ID:            v.ID,
		Name:          v.Name,
		ClearValue:    v.Value,
		Type:          v.Type,
		EnvironmentID: projID,
	}
}

func (e dbEnvironmentVariable) Variable() sdk.EnvironmentVariable {
	if sdk.NeedPlaceholder(e.Type) {
		return sdk.EnvironmentVariable{
			ID:            e.ID,
			Name:          e.Name,
			Value:         e.CipherValue,
			Type:          e.Type,
			EnvironmentID: e.EnvironmentID,
		}
	}

	return sdk.EnvironmentVariable{
		ID:            e.ID,
		Name:          e.Name,
		Value:         e.ClearValue,
		Type:          e.Type,
		EnvironmentID: e.EnvironmentID,
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbEnvironmentVariableAudit{}, "environment_variable_audit", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbEnvironmentKey{}, "environment_key", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbEnvironmentVariable{}, "environment_variable", true, "id"))
}

// PostGet is a db hook
func (eva *dbEnvironmentVariableAudit) PostGet(db gorp.SqlExecutor) error {
	var before, after sql.NullString
	query := "SELECT variable_before, variable_after from environment_variable_audit WHERE id = $1"
	if err := db.QueryRow(query, eva.ID).Scan(&before, &after); err != nil {
		return sdk.WithStack(err)
	}

	if before.Valid {
		vBefore := &sdk.EnvironmentVariable{}
		if err := json.Unmarshal([]byte(before.String), vBefore); err != nil {
			return sdk.WithStack(err)
		}
		if sdk.NeedPlaceholder(vBefore.Type) {
			vBefore.Value = sdk.PasswordPlaceholder
		}
		eva.VariableBefore = vBefore

	}

	if after.Valid {
		vAfter := &sdk.EnvironmentVariable{}
		if err := json.Unmarshal([]byte(after.String), vAfter); err != nil {
			return sdk.WithStack(err)
		}
		if sdk.NeedPlaceholder(vAfter.Type) {
			vAfter.Value = sdk.PasswordPlaceholder
		}
		eva.VariableAfter = *vAfter
	}

	return nil
}

// PostUpdate is a db hook
func (eva *dbEnvironmentVariableAudit) PostUpdate(db gorp.SqlExecutor) error {
	var vB, vA sql.NullString

	if eva.VariableBefore != nil {
		v, err := json.Marshal(eva.VariableBefore)
		if err != nil {
			return sdk.WithStack(err)
		}
		vB.Valid = true
		vB.String = string(v)
	}

	v, err := json.Marshal(eva.VariableAfter)
	if err != nil {
		return sdk.WithStack(err)
	}
	vA.Valid = true
	vA.String = string(v)

	query := "update environment_variable_audit set variable_before = $2, variable_after = $3 where id = $1"
	if _, err := db.Exec(query, eva.ID, vB, vA); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// PostInsert is a db hook
func (eva *dbEnvironmentVariableAudit) PostInsert(db gorp.SqlExecutor) error {
	return eva.PostUpdate(db)
}

// PreInsert
func (eva *dbEnvironmentVariableAudit) PreInsert(s gorp.SqlExecutor) error {
	if eva.VariableBefore != nil {
		if sdk.NeedPlaceholder(eva.VariableBefore.Type) {
			eva.VariableBefore.Value = sdk.PasswordPlaceholder
		}
	}
	if sdk.NeedPlaceholder(eva.VariableAfter.Type) {
		eva.VariableAfter.Value = sdk.PasswordPlaceholder
	}
	return nil
}
