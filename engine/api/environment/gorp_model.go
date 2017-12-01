package environment

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

type dbEnvironmentVariableAudit sdk.EnvironmentVariableAudit
type dbEnvironmentKey sdk.EnvironmentKey

func init() {
	gorpmapping.Register(gorpmapping.New(dbEnvironmentVariableAudit{}, "environment_variable_audit", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbEnvironmentKey{}, "environment_key", false))
}

// PostGet is a db hook
func (eva *dbEnvironmentVariableAudit) PostGet(db gorp.SqlExecutor) error {
	var before, after sql.NullString
	query := "SELECT variable_before, variable_after from environment_variable_audit WHERE id = $1"
	if err := db.QueryRow(query, eva.ID).Scan(&before, &after); err != nil {
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
		eva.VariableBefore = vBefore

	}

	if after.Valid {
		vAfter := &sdk.Variable{}
		if err := json.Unmarshal([]byte(after.String), vAfter); err != nil {
			return err
		}
		if sdk.NeedPlaceholder(vAfter.Type) {
			vAfter.Value = sdk.PasswordPlaceholder
		}
		eva.VariableAfter = vAfter
	}

	return nil
}

// PostUpdate is a db hook
func (eva *dbEnvironmentVariableAudit) PostUpdate(db gorp.SqlExecutor) error {
	var vB, vA sql.NullString

	if eva.VariableBefore != nil {
		v, err := json.Marshal(eva.VariableBefore)
		if err != nil {
			return err
		}
		vB.Valid = true
		vB.String = string(v)
	}

	if eva.VariableAfter != nil {
		v, err := json.Marshal(eva.VariableAfter)
		if err != nil {
			return err
		}
		vA.Valid = true
		vA.String = string(v)
	}

	query := "update environment_variable_audit set variable_before = $2, variable_after = $3 where id = $1"
	if _, err := db.Exec(query, eva.ID, vB, vA); err != nil {
		return err
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
			secret, err := secret.Encrypt([]byte(eva.VariableBefore.Value))
			if err != nil {
				return err
			}
			eva.VariableBefore.Value = base64.StdEncoding.EncodeToString(secret)
		}
	}
	if eva.VariableAfter != nil {
		if sdk.NeedPlaceholder(eva.VariableAfter.Type) {
			var err error
			secret, err := secret.Encrypt([]byte(eva.VariableAfter.Value))
			if err != nil {
				return err
			}
			eva.VariableAfter.Value = base64.StdEncoding.EncodeToString(secret)
		}
	}
	return nil
}
