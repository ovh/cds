package application

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type dbApplicationVariableAudit sdk.ApplicationVariableAudit

type dbApplicationVulnerability sdk.Vulnerability

func init() {
	gorpmapping.Register(gorpmapping.New(dbApplication{}, "application", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationVariableAudit{}, "application_variable_audit", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationKey{}, "application_key", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationVulnerability{}, "application_vulnerability", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationVariable{}, "application_variable", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbApplicationDeploymentStrategy{}, "application_deployment_strategy", true, "id"))
}

// PostGet is a db hook
func (ava *dbApplicationVariableAudit) PostGet(db gorp.SqlExecutor) error {
	var before, after sql.NullString
	query := "SELECT variable_before, variable_after from application_variable_audit WHERE id = $1"
	if err := db.QueryRow(query, ava.ID).Scan(&before, &after); err != nil {
		return err
	}

	if before.Valid {
		vBefore := &sdk.ApplicationVariable{}
		if err := sdk.JSONUnmarshal([]byte(before.String), vBefore); err != nil {
			return err
		}
		if sdk.NeedPlaceholder(vBefore.Type) {
			vBefore.Value = sdk.PasswordPlaceholder
		}
		ava.VariableBefore = vBefore

	}

	if after.Valid {
		vAfter := &sdk.ApplicationVariable{}
		if err := sdk.JSONUnmarshal([]byte(after.String), vAfter); err != nil {
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
			return sdk.WithStack(err)
		}
		vB.Valid = true
		vB.String = string(v)
	}

	v, err := json.Marshal(ava.VariableAfter)
	if err != nil {
		return sdk.WithStack(err)
	}
	vA.Valid = true
	vA.String = string(v)

	query := "update application_variable_audit set variable_before = $2, variable_after = $3 where id = $1"
	if _, err := db.Exec(query, ava.ID, vB, vA); err != nil {
		return sdk.WithStack(err)
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
