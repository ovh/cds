package workflow_v2

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func getAllWorkflowVersions(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) ([]sdk.V2WorkflowVersion, error) {
	var dbWkfVersion []dbV2WorkflowVersion
	if err := gorpmapping.GetAll(ctx, db, query, &dbWkfVersion, opts...); err != nil {
		return nil, sdk.WithStack(err)
	}
	wkfVersions := make([]sdk.V2WorkflowVersion, 0, len(dbWkfVersion))
	for _, wv := range dbWkfVersion {
		wkfVersions = append(wkfVersions, wv.V2WorkflowVersion)
	}
	return wkfVersions, nil
}

func getWorkflowVersion(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) (*sdk.V2WorkflowVersion, error) {
	var dbWkfVersion dbV2WorkflowVersion
	found, err := gorpmapping.Get(ctx, db, query, &dbWkfVersion, opts...)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &dbWkfVersion.V2WorkflowVersion, nil
}

func InsertWorkflowVersion(ctx context.Context, db gorpmapper.SqlExecutorWithTx, v *sdk.V2WorkflowVersion) error {
	_, next := telemetry.Span(ctx, "workflow_v2.InsertWorkflowVersion")
	defer next()
	v.ID = sdk.UUID()
	v.Created = time.Now()
	dbWkfVersion := &dbV2WorkflowVersion{V2WorkflowVersion: *v}

	if err := gorpmapping.Insert(db, dbWkfVersion); err != nil {
		return err
	}
	*v = dbWkfVersion.V2WorkflowVersion
	return nil
}

func DeleteWorkflowVersion(ctx context.Context, db gorpmapper.SqlExecutorWithTx, v *sdk.V2WorkflowVersion) error {
	dbWkfVersion := &dbV2WorkflowVersion{V2WorkflowVersion: *v}
	if err := gorpmapping.Delete(db, dbWkfVersion); err != nil {
		return err
	}
	return nil
}

func LoadWorkflowVersion(ctx context.Context, db gorp.SqlExecutor, projKey, vcs, repository, wkf, version string) (*sdk.V2WorkflowVersion, error) {
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_version WHERE project_key = $1 AND workflow_vcs = $2 AND workflow_repository = $3 AND workflow_name = $4 AND version = $5").
		Args(projKey, vcs, repository, wkf, version)
	return getWorkflowVersion(ctx, db, query)
}

func LoadAllVerionsByWorkflow(ctx context.Context, db gorp.SqlExecutor, projKey, vcs, repository, wkf string) ([]sdk.V2WorkflowVersion, error) {
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_version WHERE project_key = $1 AND workflow_vcs = $2 AND workflow_repository = $3 AND workflow_name = $4").
		Args(projKey, vcs, repository, wkf)
	return getAllWorkflowVersions(ctx, db, query)
}

type V2WorkflowVersionWorkflowShort struct {
	DistinctID         string `db:"id"`
	ProjectKey         string `db:"project_key"`
	WorkflowVCS        string `db:"workflow_vcs"`
	WorkflowRepository string `db:"workflow_repository"`
	WorkflowName       string `db:"workflow_name"`
}

func (w V2WorkflowVersionWorkflowShort) String() string {
	return fmt.Sprintf("%s/%s/%s/%s", w.ProjectKey, w.WorkflowVCS, w.WorkflowRepository, w.WorkflowName)
}

func LoadDistinctWorkflowVersionByWorkflow(ctx context.Context, db gorp.SqlExecutor) ([]V2WorkflowVersionWorkflowShort, error) {
	var results []V2WorkflowVersionWorkflowShort
	query := `
		SELECT 
			DISTINCT(project_key, workflow_vcs, workflow_repository, workflow_name) as id, 
			project_key, workflow_vcs, workflow_repository, workflow_name 
		FROM v2_workflow_version`
	if _, err := db.Select(&results, query); err != nil {
		return nil, err
	}
	return results, nil
}
