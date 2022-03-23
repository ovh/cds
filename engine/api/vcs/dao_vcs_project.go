package vcs

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, vcsProject *sdk.VCSProject) error {
	vcsProject.ID = sdk.UUID()
	vcsProject.Created = time.Now()
	dbData := &dbVCSProject{VCSProject: *vcsProject}
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*vcsProject = dbData.VCSProject
	return nil
}

func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, vcsProject *sdk.VCSProject) error {
	var dbData = dbVCSProject{VCSProject: *vcsProject}
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbData); err != nil {
		return err
	}
	*vcsProject = dbData.VCSProject
	return nil
}

func Delete(db gorpmapper.SqlExecutorWithTx, projectID int64, name string) error {
	_, err := db.Exec("DELETE FROM vcs_project WHERE project_id = $1 AND name = $2", projectID, name)
	return sdk.WrapError(err, "cannot delete vcs_project %d/%s", projectID, name)
}

func LoadAllVCSByProject(ctx context.Context, db gorp.SqlExecutor, projectID int64) ([]sdk.VCSProject, error) {
	return loadAllVCSByProject(ctx, db, projectID)
}

func loadAllVCSByProject(ctx context.Context, db gorp.SqlExecutor, projectID int64, opts ...gorpmapping.GetOptionFunc) ([]sdk.VCSProject, error) {
	var res []dbVCSProject

	query := gorpmapping.NewQuery(`SELECT * FROM vcs_project WHERE project_id = $1`).Args(projectID)

	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}
	vcsProjects := make([]sdk.VCSProject, 0, len(res))

	for _, res := range res {
		isValid, err := gorpmapping.CheckSignature(res, res.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for vcs_project %s", res.ID)
		}
		if !isValid {
			log.Error(ctx, "vcs_project %d data corrupted", res.ID)
			continue
		}
		vcsProjects = append(vcsProjects, res.VCSProject)
	}
	return vcsProjects, nil
}

func LoadVCSByProject(ctx context.Context, db gorp.SqlExecutor, projectID int64, vcsName string, opts ...gorpmapping.GetOptionFunc) (*sdk.VCSProject, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM vcs_project WHERE project_id = $1 AND name = $2`).Args(projectID, vcsName)
	var res dbVCSProject
	found, err := gorpmapping.Get(context.Background(), db, query, &res, opts...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(res, res.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(context.Background(), "vcs_project %d data corrupted", res.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &res.VCSProject, nil
}
