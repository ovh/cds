package project

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

type dbProject sdk.Project
type dbVariable sdk.Variable
type dbProjectVariableAudit sdk.ProjectVariableAudit

func init() {
	gorpmapping.Register(gorpmapping.New(dbProject{}, "project", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectVariableAudit{}, "project_variable_audit", true, "id"))
}

// PostGet is a db hook
func (p *dbProject) PostGet(db gorp.SqlExecutor) error {
	metadataStr, err := db.SelectNullStr("select metadata from project where id = $1", p.ID)
	if err != nil {
		return err
	}

	if metadataStr.Valid {
		metadata := sdk.Metadata{}
		if err := json.Unmarshal([]byte(metadataStr.String), &metadata); err != nil {
			return err
		}
		p.Metadata = metadata
	}
	return nil
}

// PostUpdate is a db hook
func (p *dbProject) PostUpdate(db gorp.SqlExecutor) error {
	b, err := json.Marshal(p.Metadata)
	if err != nil {
		return err
	}
	if _, err := db.Exec("update project set metadata = $2 where id = $1", p.ID, b); err != nil {
		return err
	}
	return nil
}

// PostInsert is a db hook
func (p *dbProject) PostInsert(db gorp.SqlExecutor) error {
	return p.PostUpdate(db)
}

// PostGet is a db hook
func (pva *dbProjectVariableAudit) PostGet(db gorp.SqlExecutor) error {
	var before, after sql.NullString
	query := "SELECT variable_before, variable_after from project_variable_audit WHERE id = $1"
	if err := db.QueryRow(query, pva.ID).Scan(&before, &after); err != nil {
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
		pva.VariableBefore = vBefore

	}

	if after.Valid {
		vAfter := &sdk.Variable{}
		if err := json.Unmarshal([]byte(after.String), vAfter); err != nil {
			return err
		}
		if sdk.NeedPlaceholder(vAfter.Type) {
			vAfter.Value = sdk.PasswordPlaceholder
		}
		pva.VariableAfter = vAfter
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

	if pva.VariableAfter != nil {
		v, err := json.Marshal(pva.VariableAfter)
		if err != nil {
			return err
		}
		vA.Valid = true
		vA.String = string(v)
	}

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
			secret, err := secret.Encrypt([]byte(pva.VariableBefore.Value))
			if err != nil {
				return err
			}
			pva.VariableBefore.Value = base64.StdEncoding.EncodeToString(secret)
		}
	}
	if pva.VariableAfter != nil {
		if sdk.NeedPlaceholder(pva.VariableAfter.Type) {
			var err error
			secret, err := secret.Encrypt([]byte(pva.VariableAfter.Value))
			if err != nil {
				return err
			}
			pva.VariableAfter.Value = base64.StdEncoding.EncodeToString(secret)
		}
	}
	return nil
}
