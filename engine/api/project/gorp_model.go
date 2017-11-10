package project

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"

	"github.com/go-gorp/gorp"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

type dbProject sdk.Project
type dbVariable sdk.Variable
type dbProjectVariableAudit sdk.ProjectVariableAudit
type dbProjectKey sdk.ProjectKey

func init() {
	gorpmapping.Register(gorpmapping.New(dbProject{}, "project", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectVariableAudit{}, "project_variable_audit", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectKey{}, "project_key", false))
}

// PostGet is a db hook
func (p *dbProject) PostGet(db gorp.SqlExecutor) error {
	var fields = struct {
		Metadata   sql.NullString `db:"metadata"`
		VCSServers []byte         `db:"vcs_servers"`
	}{}

	if err := db.QueryRow("select metadata,vcs_servers from project where id = $1", p.ID).Scan(&fields.Metadata, &fields.VCSServers); err != nil {
		return err
	}

	if err := gorpmapping.JSONNullString(fields.Metadata, &p.Metadata); err != nil {
		return err
	}

	if len(fields.VCSServers) > 0 {
		clearVCSServer, err := secret.Decrypt([]byte(fields.VCSServers))
		if err != nil {
			return err
		}

		if len(clearVCSServer) > 0 {
			if err := yaml.Unmarshal(clearVCSServer, &p.VCSServers); err != nil {
				return sdk.WrapError(err, "Unable to load project %d", p.ID)
			}
		}
	}

	return nil
}

// PostUpdate is a db hook
func (p *dbProject) PostUpdate(db gorp.SqlExecutor) error {
	b, err := json.Marshal(p.Metadata)
	if err != nil {
		return err
	}

	b1, err := yaml.Marshal(p.VCSServers)
	if err != nil {
		return err
	}

	encryptedVCSServerStr, err := secret.Encrypt(b1)
	if err != nil {
		return err
	}

	if _, err := db.Exec("update project set metadata = $2, vcs_servers = $3 where id = $1", p.ID, b, encryptedVCSServerStr); err != nil {
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
