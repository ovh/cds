package integration

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DeleteIntegration deletes a integration
func DeleteIntegration(db gorp.SqlExecutor, integration sdk.ProjectIntegration) error {
	pp := dbProjectIntegration{ProjectIntegration: integration}
	if _, err := db.Delete(&pp); err != nil {
		return sdk.WrapError(err, "Cannot remove integration")
	}
	return nil
}

func load(db gorp.SqlExecutor, query gorpmapping.Query) (sdk.ProjectIntegration, error) {
	pi, err := loadWithClearPassword(db, query)
	pi.Blur()
	pi.Model.Blur()
	return pi, err
}

func loadWithClearPassword(db gorp.SqlExecutor, query gorpmapping.Query) (sdk.ProjectIntegration, error) {
	var pp dbProjectIntegration
	found, err := gorpmapping.Get(context.Background(), db, query, &pp, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return sdk.ProjectIntegration{}, err
	}
	if !found {
		return sdk.ProjectIntegration{}, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(pp, pp.Signature)
	if err != nil {
		return sdk.ProjectIntegration{}, err
	}
	if !isValid {
		log.Error(context.Background(), "integration.LoadModelByName> model  %d data corrupted", pp.ID)
		return sdk.ProjectIntegration{}, sdk.WithStack(sdk.ErrNotFound)
	}

	imodel, err := LoadModelWithClearPassword(db, pp.IntegrationModelID)
	if err != nil {
		return sdk.ProjectIntegration{}, err
	}
	pp.Model = imodel

	return pp.ProjectIntegration, nil
}

// LoadProjectIntegrationByName Load a integration by project key and its name
func LoadProjectIntegrationByName(db gorp.SqlExecutor, key string, name string) (sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery(`
		SELECT project_integration.*
		FROM project_integration
		JOIN project ON project.id = project_integration.project_id
		WHERE project.projectkey = $1 AND project_integration.name = $2`).Args(key, name)

	return load(db, query)
}

func LoadProjectIntegrationByNameWithClearPassword(db gorp.SqlExecutor, key string, name string) (sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery(`
	SELECT project_integration.*
	FROM project_integration
	JOIN project ON project.id = project_integration.project_id
	WHERE project.projectkey = $1 AND project_integration.name = $2`).Args(key, name)

	return loadWithClearPassword(db, query)
}

// LoadProjectIntegrationByID returns integration, selecting by its id
func LoadProjectIntegrationByID(db gorp.SqlExecutor, id int64) (*sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery("SELECT * from project_integration WHERE id = $1").Args(id)
	pp, err := load(db, query)
	return &pp, err
}

func LoadProjectIntegrationByIDWithClearPassword(db gorp.SqlExecutor, id int64) (*sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery("SELECT * from project_integration WHERE id = $1").Args(id)
	pp, err := loadWithClearPassword(db, query)
	return &pp, err
}

func loadAllWithClearPassword(db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectIntegration, error) {
	var pp []dbProjectIntegration
	if err := gorpmapping.GetAll(context.Background(), db, query, &pp, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}

	var integrations = make([]sdk.ProjectIntegration, len(pp))
	for i, p := range pp {
		isValid, err := gorpmapping.CheckSignature(p, p.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(context.Background(), "integration.loadAll> model %d data corrupted", p.ID)
			continue
		}

		imodel, err := LoadModelWithClearPassword(db, p.IntegrationModelID)
		if err != nil {
			return nil, err
		}
		p.Model = imodel
		integrations[i] = p.ProjectIntegration
	}
	return integrations, nil

}
func loadAll(db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectIntegration, error) {
	var pp []dbProjectIntegration
	if err := gorpmapping.GetAll(context.Background(), db, query, &pp, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}
	var integrations = make([]sdk.ProjectIntegration, len(pp))
	for i, p := range pp {
		isValid, err := gorpmapping.CheckSignature(p, p.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(context.Background(), "integration.loadAll> model %d data corrupted", p.ID)
			continue
		}

		imodel, err := LoadModel(db, p.IntegrationModelID)
		if err != nil {
			return nil, err
		}
		p.Model = imodel
		integrations[i] = p.ProjectIntegration
		integrations[i].Blur()
	}
	return integrations, nil
}

// LoadIntegrationsByProjectIDWithClearPassword load integration integrations by project id
func LoadIntegrationsByProjectIDWithClearPassword(db gorp.SqlExecutor, id int64) ([]sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery("SELECT * from project_integration WHERE project_id = $1").Args(id)
	return loadAllWithClearPassword(db, query)
}

// LoadIntegrationsByProjectID load integration integrations by project id
func LoadIntegrationsByProjectID(db gorp.SqlExecutor, id int64) ([]sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery("SELECT * from project_integration WHERE project_id = $1").Args(id)
	return loadAll(db, query)
}

// LoadIntegrationsByIDs load integration integrations by id
func LoadIntegrationsByIDs(db gorp.SqlExecutor, ids []int64) ([]sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery("SELECT * from project_integration WHERE id = ANY($1)").Args(pq.Int64Array(ids))
	return loadAll(db, query)
}

// InsertIntegration inserts a integration
func InsertIntegration(db gorpmapper.SqlExecutorWithTx, pp *sdk.ProjectIntegration) error {
	oldConfig := pp.Config.Clone()
	ppDb := dbProjectIntegration{ProjectIntegration: *pp}
	if err := gorpmapping.InsertAndSign(context.Background(), db, &ppDb); err != nil {
		return sdk.WrapError(err, "Cannot insert integration")
	}
	*pp = ppDb.ProjectIntegration
	pp.Config = oldConfig
	pp.Blur()
	return nil
}

// UpdateIntegration Update a integration
func UpdateIntegration(db gorpmapper.SqlExecutorWithTx, pp sdk.ProjectIntegration) error {
	var oldConfig *sdk.ProjectIntegration

	givenConfig := pp.Config.Clone()
	for k, cfg := range givenConfig {
		if cfg.Type == sdk.IntegrationConfigTypePassword && cfg.Value == sdk.PasswordPlaceholder {
			if oldConfig == nil {
				// reload the previous config to ensure we don't store placeholder
				var err error
				oldConfig, err = LoadProjectIntegrationByIDWithClearPassword(db, pp.ID)
				if err != nil {
					return err
				}
			}
			cfg.Value = oldConfig.Config[k].Value
		}
		givenConfig[k] = cfg
	}

	pp.Config = givenConfig
	ppDb := dbProjectIntegration{ProjectIntegration: pp}
	if err := gorpmapping.UpdateAndSign(context.Background(), db, &ppDb); err != nil {
		return sdk.WrapError(err, "Cannot update integration")
	}
	pp.Config = givenConfig
	pp.Blur()
	return nil
}

// AddOnWorkflow link a project integration on a workflow
func AddOnWorkflow(db gorp.SqlExecutor, workflowID int64, projectIntegrationID int64) error {
	query := "INSERT INTO workflow_project_integration (workflow_id, project_integration_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
	if _, err := db.Exec(query, workflowID, projectIntegrationID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// LoadWorkflowIntegrationsByWorkflowID load workflow integrations by workflowid
func LoadWorkflowIntegrationsByWorkflowID(db gorp.SqlExecutor, id int64) ([]sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery(`
		SELECT project_integration.*
		FROM project_integration
		JOIN workflow_project_integration ON project_integration.id = workflow_project_integration.project_integration_id
		WHERE workflow_project_integration.workflow_id = $1
	`).Args(id)
	return loadAll(db, query)
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

// LoadAllIntegrationsForProjectsWithDecryption load all integrations for all given project, with decryption
func LoadAllIntegrationsForProjectsWithDecryption(ctx context.Context, db gorp.SqlExecutor, projIDs []int64) (map[int64][]sdk.ProjectIntegration, error) {
	return loadAllIntegrationsForProjects(ctx, db, projIDs, gorpmapping.GetOptions.WithDecryption)
}

func loadAllIntegrationsForProjects(ctx context.Context, db gorp.SqlExecutor, projIDs []int64, opts ...gorpmapping.GetOptionFunc) (map[int64][]sdk.ProjectIntegration, error) {
	var res []dbProjectIntegration
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_integration
		WHERE project_id = ANY($1)
		ORDER BY project_id
	`).Args(pq.Int64Array(projIDs))
	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	projsInts := make(map[int64][]sdk.ProjectIntegration)

	for i := range res {
		dbProjInt := res[i]
		isValid, err := gorpmapping.CheckSignature(dbProjInt, dbProjInt.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project.loadAllIntegrationsForProjects> project integration id %d data corrupted", dbProjInt.ID)
			continue
		}
		if _, ok := projsInts[dbProjInt.ProjectID]; !ok {
			projsInts[dbProjInt.ProjectID] = make([]sdk.ProjectIntegration, 0)
		}
		pIntegration := dbProjInt.ProjectIntegration
		projsInts[dbProjInt.ProjectID] = append(projsInts[dbProjInt.ProjectID], pIntegration)
	}
	return projsInts, nil
}
