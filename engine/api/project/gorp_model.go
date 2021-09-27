package project

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type dbProject sdk.Project
type dbProjectVariableAudit sdk.ProjectVariableAudit
type dbProjectKey struct {
	gorpmapper.SignedEntity
	sdk.ProjectKey
}

func (e dbProjectKey) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{e.ProjectID, e.ID, e.Name}
	return gorpmapper.CanonicalForms{
		"{{print .ProjectID}}{{print .ID}}{{.Name}}",
	}
}

type dbLabel sdk.Label

type dbProjectVariable struct {
	gorpmapper.SignedEntity
	ID          int64  `db:"id"`
	ProjectID   int64  `db:"project_id"`
	Name        string `db:"var_name"`
	ClearValue  string `db:"var_value"`
	CipherValue string `db:"cipher_value" gorpmapping:"encrypted,ID,Name"`
	Type        string `db:"var_type"`
}

func (e dbProjectVariable) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{e.ProjectID, e.ID, e.Name, e.Type}
	return gorpmapper.CanonicalForms{
		"{{print .ProjectID}}{{print .ID}}{{.Name}}{{.Type}}",
	}
}

func newDBProjectVariable(v sdk.ProjectVariable, projID int64) dbProjectVariable {
	if sdk.NeedPlaceholder(v.Type) {
		return dbProjectVariable{
			ID:          v.ID,
			Name:        v.Name,
			CipherValue: v.Value,
			Type:        v.Type,
			ProjectID:   projID,
		}
	}
	return dbProjectVariable{
		ID:         v.ID,
		Name:       v.Name,
		ClearValue: v.Value,
		Type:       v.Type,
		ProjectID:  projID,
	}
}

func (e dbProjectVariable) Variable() sdk.ProjectVariable {
	if sdk.NeedPlaceholder(e.Type) {
		return sdk.ProjectVariable{
			ID:        e.ID,
			Name:      e.Name,
			Value:     e.CipherValue,
			Type:      e.Type,
			ProjectID: e.ProjectID,
		}
	}

	return sdk.ProjectVariable{
		ID:        e.ID,
		Name:      e.Name,
		Value:     e.ClearValue,
		Type:      e.Type,
		ProjectID: e.ProjectID,
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbProject{}, "project", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectVariableAudit{}, "project_variable_audit", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectKey{}, "project_key", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbLabel{}, "project_label", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectVariable{}, "project_variable", true, "id"))
}

// PostGet is a db hook
func (pva *dbProjectVariableAudit) PostGet(db gorp.SqlExecutor) error {
	var before, after sql.NullString
	query := "SELECT variable_before, variable_after from project_variable_audit WHERE id = $1"
	if err := db.QueryRow(query, pva.ID).Scan(&before, &after); err != nil {
		return err
	}

	if before.Valid {
		vBefore := &sdk.ProjectVariable{}
		if err := sdk.JSONUnmarshal([]byte(before.String), vBefore); err != nil {
			return err
		}
		if sdk.NeedPlaceholder(vBefore.Type) {
			vBefore.Value = sdk.PasswordPlaceholder
		}
		pva.VariableBefore = vBefore

	}

	if after.Valid {
		vAfter := &sdk.ProjectVariable{}
		if err := sdk.JSONUnmarshal([]byte(after.String), vAfter); err != nil {
			return err
		}
		if sdk.NeedPlaceholder(vAfter.Type) {
			vAfter.Value = sdk.PasswordPlaceholder
		}
		pva.VariableAfter = *vAfter
	}

	return nil
}

// PostUpdate is a db hook
func (pva *dbProjectVariableAudit) PostUpdate(db gorp.SqlExecutor) error {
	var vB, vA sql.NullString

	if pva.VariableBefore != nil {
		v, err := json.Marshal(pva.VariableBefore)
		if err != nil {
			return err
		}
		vB.Valid = true
		vB.String = string(v)
	}

	v, err := json.Marshal(pva.VariableAfter)
	if err != nil {
		return err
	}
	vA.Valid = true
	vA.String = string(v)

	query := "update project_variable_audit set variable_before = $2, variable_after = $3 where id = $1"
	if _, err := db.Exec(query, pva.ID, vB, vA); err != nil {
		return err
	}
	return nil
}

// PostInsert is a db hook
func (pva *dbProjectVariableAudit) PostInsert(db gorp.SqlExecutor) error {
	return pva.PostUpdate(db)
}

// PreInsert
func (pva *dbProjectVariableAudit) PreInsert(s gorp.SqlExecutor) error {
	if pva.VariableBefore != nil {
		if sdk.NeedPlaceholder(pva.VariableBefore.Type) {
			pva.VariableBefore.Value = sdk.PasswordPlaceholder
		}
	}
	if sdk.NeedPlaceholder(pva.VariableAfter.Type) {
		pva.VariableAfter.Value = sdk.PasswordPlaceholder
	}

	return nil
}
