package integration

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// PostGet is a db hook
func (pp *dbProjectIntegration) PostGet(db gorp.SqlExecutor) error {
	model, err := LoadModel(db, pp.IntegrationModelID, false)
	if err != nil {
		return sdk.WrapError(err, "Cannot load model")
	}
	pp.Model = model

	query := "SELECT config FROM project_integration where id = $1"
	s, err := db.SelectNullStr(query, pp.ID)
	if err != nil {
		return sdk.WrapError(err, "Cannot get config")
	}
	if err := gorpmapping.JSONNullString(s, &pp.Config); err != nil {
		return err
	}
	return nil
}

// DeleteIntegration deletes a integration
func DeleteIntegration(db gorp.SqlExecutor, integration sdk.ProjectIntegration) error {
	pp := dbProjectIntegration(integration)
	if _, err := db.Delete(&pp); err != nil {
		return sdk.WrapError(err, "Cannot remove integration")
	}
	return nil
}

// LoadProjectIntegrationByName Load a integration by project key and its name
func LoadProjectIntegrationByName(db gorp.SqlExecutor, key string, name string, clearPwd bool) (sdk.ProjectIntegration, error) {
	var pp dbProjectIntegration
	query := `
		SELECT project_integration.*
		FROM project_integration
		JOIN project ON project.id = project_integration.project_id
		WHERE project.projectkey = $1 AND project_integration.name = $2
	`
	if err := db.SelectOne(&pp, query, key, name); err != nil {
		return sdk.ProjectIntegration{}, sdk.WithStack(err)
	}
	p := sdk.ProjectIntegration(pp)
	for k, v := range p.Config {
		if v.Type == sdk.IntegrationConfigTypePassword {
			if clearPwd {
				decryptedValue, errD := secret.DecryptValue(v.Value)
				if errD != nil {
					return p, sdk.WrapError(errD, "LoadProjectIntegrationByName> Cannot decrypt value")
				}
				v.Value = decryptedValue
				p.Config[k] = v
			} else {
				v.Value = sdk.PasswordPlaceholder
				p.Config[k] = v
			}
		}
	}

	return p, nil
}

// LoadProjectIntegrationsByKeyAndType Load a integration by project key and its name
func LoadProjectIntegrationsByKeyAndType(db gorp.SqlExecutor, key string, integrationType *sdk.IntegrationType, clearPwd bool) ([]sdk.ProjectIntegration, error) {
	var pps []dbProjectIntegration
	query := `
		SELECT project_integration.*
		FROM project_integration
		JOIN project ON project.id = project_integration.project_id
		JOIN integration_model ON integration_model.id = project_integration.integration_model_id
		WHERE project.projectkey = $1
	`
	if integrationType != nil {
		switch *integrationType {
		case sdk.IntegrationTypeEvent:
			query += " AND integration_model.event = true"
		case sdk.IntegrationTypeCompute:
			query += " AND integration_model.compute = true"
		case sdk.IntegrationTypeStorage:
			query += " AND integration_model.storage = true"
		case sdk.IntegrationTypeHook:
			query += " AND integration_model.hook = true"
		case sdk.IntegrationTypeDeployment:
			query += " AND integration_model.deployment = true"
		}
	}
	if _, err := db.Select(&pps, query, key); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}

	projectIntegrations := make([]sdk.ProjectIntegration, len(pps))
	for i, projectIntDB := range pps {
		if err := projectIntDB.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "cannot post get for project integration %s with id %d", projectIntDB.Name, projectIntDB.ID)
		}
		projectInt := sdk.ProjectIntegration(projectIntDB)
		for k, v := range projectInt.Config {
			if v.Type == sdk.IntegrationConfigTypePassword {
				if clearPwd {
					decryptedValue, errD := secret.DecryptValue(v.Value)
					if errD != nil {
						return projectIntegrations, sdk.WrapError(errD, "LoadProjectIntegrationByName> Cannot decrypt value")
					}
					v.Value = decryptedValue
					projectInt.Config[k] = v
				} else {
					v.Value = sdk.PasswordPlaceholder
					projectInt.Config[k] = v
				}
			}
		}
		projectIntegrations[i] = projectInt
	}

	return projectIntegrations, nil
}

// LoadProjectIntegrationByID returns integration, selecting by its id
func LoadProjectIntegrationByID(db gorp.SqlExecutor, id int64, clearPassword bool) (*sdk.ProjectIntegration, error) {
	var pp dbProjectIntegration
	if err := db.SelectOne(&pp, "SELECT * from project_integration WHERE id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := pp.PostGet(db); err != nil {
		return nil, sdk.WrapError(err, "cannot post get for project integration with id %d", id)
	}
	for k, v := range pp.Config {
		if v.Type == sdk.IntegrationConfigTypePassword {
			if clearPassword {
				secret, errD := secret.DecryptValue(v.Value)
				if errD != nil {
					return nil, sdk.WrapError(errD, "LoadIntegrationByID> Cannot decrypt password")
				}
				v.Value = string(secret)
				pp.Config[k] = v
			} else {
				v.Value = sdk.PasswordPlaceholder
				pp.Config[k] = v
			}
		}
	}
	res := sdk.ProjectIntegration(pp)
	return &res, nil
}

// LoadIntegrationsByProjectID load integration integrations by project id
func LoadIntegrationsByProjectID(db gorp.SqlExecutor, id int64, clearPassword bool) ([]sdk.ProjectIntegration, error) {
	integrations := []sdk.ProjectIntegration{}

	var res []dbProjectIntegration
	if _, err := db.Select(&res, "SELECT * from project_integration WHERE project_id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return integrations, nil
		}
		return nil, err
	}

	integrations = make([]sdk.ProjectIntegration, len(res))
	for i := range res {
		pp := &res[i]
		if err := pp.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "Cannot post get")
		}

		for k, v := range pp.Config {
			if v.Type == sdk.IntegrationConfigTypePassword {
				if clearPassword {
					secret, errD := secret.DecryptValue(v.Value)
					if errD != nil {
						return nil, sdk.WrapError(errD, "LoadIntegrationByID> Cannot decrypt password")
					}
					v.Value = string(secret)
					pp.Config[k] = v
				} else {
					v.Value = sdk.PasswordPlaceholder
					pp.Config[k] = v
				}
			}
		}
		integrations[i] = sdk.ProjectIntegration(*pp)
	}
	return integrations, nil
}

// InsertIntegration inserts a integration
func InsertIntegration(db gorp.SqlExecutor, pp *sdk.ProjectIntegration) error {
	for k, v := range pp.Config {
		if v.Type == sdk.IntegrationConfigTypePassword {
			s, errS := secret.EncryptValue(v.Value)
			if errS != nil {
				return sdk.WrapError(errS, "InsertIntegration> Cannot encrypt password")
			}
			v.Value = string(s)
			pp.Config[k] = v
		}
	}
	ppDb := dbProjectIntegration(*pp)
	if err := db.Insert(&ppDb); err != nil {
		return sdk.WrapError(err, "Cannot insert integration")
	}
	*pp = sdk.ProjectIntegration(ppDb)
	return nil
}

// UpdateIntegration Update a integration
func UpdateIntegration(db gorp.SqlExecutor, pp sdk.ProjectIntegration) error {
	for k, v := range pp.Config {
		if v.Type == sdk.IntegrationConfigTypePassword {
			s, errS := secret.EncryptValue(v.Value)
			if errS != nil {
				return sdk.WrapError(errS, "UpdateIntegration> Cannot encrypt password")
			}
			v.Value = string(s)
			pp.Config[k] = v
		}
	}
	ppDb := dbProjectIntegration(pp)
	if _, err := db.Update(&ppDb); err != nil {
		return sdk.WrapError(err, "Cannot update integration")
	}
	return nil
}

// PostUpdate is a db hook
func (pp *dbProjectIntegration) PostUpdate(db gorp.SqlExecutor) error {
	configB, err := gorpmapping.JSONToNullString(pp.Config)
	if err != nil {
		return sdk.WrapError(err, "Cannot post insert integration")
	}

	if _, err := db.Exec("UPDATE project_integration set config = $1 WHERE id = $2", configB, pp.ID); err != nil {
		return sdk.WrapError(err, "Cannot update config")
	}
	return nil
}

// PostInsert is a db hook
func (pp *dbProjectIntegration) PostInsert(db gorp.SqlExecutor) error {
	if err := pp.PostUpdate(db); err != nil {
		return sdk.WrapError(err, "Cannot update")
	}
	return nil
}

// LoadIntegrationsByWorkflowID load integration integrations by Workflow id
func LoadIntegrationsByWorkflowID(db gorp.SqlExecutor, id int64, clearPassword bool) ([]sdk.ProjectIntegration, error) {
	integrations := []sdk.ProjectIntegration{}
	query := `SELECT project_integration.*
	FROM project_integration
		JOIN workflow_project_integration ON workflow_project_integration.project_integration_id = project_integration.id
	WHERE workflow_project_integration.workflow_id = $1`
	var res []dbProjectIntegration
	if _, err := db.Select(&res, query, id); err != nil {
		if err == sql.ErrNoRows {
			return integrations, nil
		}
		return nil, err
	}

	integrations = make([]sdk.ProjectIntegration, len(res))
	for i := range res {
		pp := &res[i]
		if err := pp.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "Cannot post get")
		}

		for k, v := range pp.Config {
			if v.Type == sdk.IntegrationConfigTypePassword {
				if clearPassword {
					secret, errD := secret.DecryptValue(v.Value)
					if errD != nil {
						return nil, sdk.WrapError(errD, "LoadIntegrationByID> Cannot decrypt password")
					}
					v.Value = string(secret)
					pp.Config[k] = v
				} else {
					v.Value = sdk.PasswordPlaceholder
					pp.Config[k] = v
				}
			}
		}
		integrations[i] = sdk.ProjectIntegration(*pp)
	}
	return integrations, nil
}

// AddOnWorkflow link a project integration on a workflow
func AddOnWorkflow(db gorp.SqlExecutor, workflowID int64, projectIntegrationID int64) error {
	query := "INSERT INTO workflow_project_integration (workflow_id, project_integration_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"

	if _, err := db.Exec(query, workflowID, projectIntegrationID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// RemoveFromWorkflow remove a project integration on a workflow
func RemoveFromWorkflow(db gorp.SqlExecutor, workflowID int64, projectIntegrationID int64) error {
	query := "DELETE FROM workflow_project_integration WHERE workflow_id = $1 AND project_integration_id = $2"

	if _, err := db.Exec(query, workflowID, projectIntegrationID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// DeleteFromWorkflow remove a project integration on a workflow
func DeleteFromWorkflow(db gorp.SqlExecutor, workflowID int64) error {
	query := "DELETE FROM workflow_project_integration WHERE workflow_id = $1"

	if _, err := db.Exec(query, workflowID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
