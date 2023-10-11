package vcs

import (
	"context"
	"github.com/ovh/cds/sdk/telemetry"
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
	vcsProject.LastModified = time.Now()
	dbData := &dbVCSProject{VCSProject: *vcsProject}
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*vcsProject = dbData.VCSProject
	return nil
}

func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, vcsProject *sdk.VCSProject) error {
	vcsProject.LastModified = time.Now()
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

func getVCSProject(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) (*sdk.VCSProject, error) {
	var res dbVCSProject
	found, err := gorpmapping.Get(ctx, db, q, &res, opts...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(res, res.Signature)
	if err != nil {
		return nil, sdk.WrapError(err, "error when checking signature for vcs_project %s", res.ID)
	}
	if !isValid {
		log.Error(ctx, "vcs_project %d data corrupted", res.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)

	}
	return &res.VCSProject, nil
}

func getAllVCSProject(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) ([]sdk.VCSProject, error) {
	var res []dbVCSProject
	if err := gorpmapping.GetAll(ctx, db, q, &res, opts...); err != nil {
		return nil, err
	}
	vcProjects := make([]sdk.VCSProject, 0, len(res))

	for _, res := range res {
		isValid, err := gorpmapping.CheckSignature(res, res.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for vcs_project %s", res.ID)
		}
		if !isValid {
			log.Error(ctx, "vcs_project %d data corrupted", res.ID)
			continue
		}
		vcProjects = append(vcProjects, res.VCSProject)
	}
	return vcProjects, nil
}

func LoadAllVCSByProject(ctx context.Context, db gorp.SqlExecutor, projectKey string, opts ...gorpmapping.GetOptionFunc) ([]sdk.VCSProject, error) {
	var res []dbVCSProject

	query := gorpmapping.NewQuery(`SELECT vcs_project.* FROM vcs_project JOIN project ON project.id = vcs_project.project_id WHERE project.projectkey = $1`).Args(projectKey)

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

func LoadVCSByProject(ctx context.Context, db gorp.SqlExecutor, projectKey string, vcsName string, opts ...gorpmapping.GetOptionFunc) (*sdk.VCSProject, error) {
	ctx, next := telemetry.Span(ctx, "vcs.LoadVCSByProject")
	defer next()
	query := gorpmapping.NewQuery(`SELECT vcs_project.* FROM vcs_project JOIN project ON project.id = vcs_project.project_id WHERE project.projectkey = $1 AND vcs_project.name = $2`).Args(projectKey, vcsName)
	return getVCSProject(ctx, db, query, opts...)
}

func LoadVCSByIDAndProjectKey(ctx context.Context, db gorp.SqlExecutor, projectKey string, vcsID string, opts ...gorpmapping.GetOptionFunc) (*sdk.VCSProject, error) {
	ctx, next := telemetry.Span(ctx, "vcs.LoadVCSByIDAndProjectKey")
	defer next()
	query := gorpmapping.NewQuery(`SELECT vcs_project.* FROM vcs_project JOIN project ON project.id = vcs_project.project_id WHERE project.projectkey = $1 AND vcs_project.id = $2`).Args(projectKey, vcsID)
	return getVCSProject(ctx, db, query, opts...)
}

func LoadAllVCSGerrit(ctx context.Context, db gorp.SqlExecutor, opts ...gorpmapping.GetOptionFunc) ([]sdk.VCSProject, error) {
	query := gorpmapping.NewQuery(`SELECT vcs_project.* FROM vcs_project WHERE vcs_project.type = 'gerrit'`)
	return getAllVCSProject(ctx, db, query, opts...)
}
