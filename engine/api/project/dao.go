package project

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func loadAllByRepo(ctx context.Context, db gorp.SqlExecutor, store cache.Store, query string, args []interface{}, opts ...LoadOptionFunc) (sdk.Projects, error) {
	return loadprojects(ctx, db, store, opts, query, args...)
}

// LoadAllByRepoAndGroupIDs returns all projects with an application linked to the repo against the groups
func LoadAllByRepoAndGroupIDs(ctx context.Context, db gorp.SqlExecutor, store cache.Store, groupIDs []int64, repo string, opts ...LoadOptionFunc) (sdk.Projects, error) {
	query := `SELECT DISTINCT project.*
		FROM  project
		JOIN  application on project.id = application.project_id
		WHERE application.repo_fullname = $3
		AND   project.id IN (
			SELECT project_group.project_id
			FROM project_group
			WHERE
				project_group.group_id = ANY(string_to_array($1, ',')::int[])
				OR
				$2 = ANY(string_to_array($1, ',')::int[])
		)`
	args := []interface{}{gorpmapping.IDsToQueryString(groupIDs), group.SharedInfraGroup.ID, repo}
	return loadAllByRepo(ctx, db, store, query, args, opts...)
}

// LoadAllByRepo returns all projects with an application linked to the repo
func LoadAllByRepo(ctx context.Context, db gorp.SqlExecutor, store cache.Store, repo string, opts ...LoadOptionFunc) (sdk.Projects, error) {
	query := `SELECT DISTINCT project.*
	FROM  project
	JOIN  application on project.id = application.project_id
	WHERE application.repo_fullname = $1
	ORDER by project.name, project.projectkey ASC`
	args := []interface{}{repo}
	return loadAllByRepo(ctx, db, store, query, args, opts...)
}

// LoadAllByGroupIDs returns all projects given groups
func LoadAllByGroupIDs(ctx context.Context, db gorp.SqlExecutor, store cache.Store, IDs []int64, opts ...LoadOptionFunc) (sdk.Projects, error) {
	query := `SELECT project.*
	FROM project
	WHERE project.id IN (
		SELECT project_group.project_id
		FROM project_group
		WHERE
			project_group.group_id = ANY(string_to_array($1, ',')::int[])
			OR
			$2 = ANY(string_to_array($1, ',')::int[])
	)
	ORDER by project.name, project.projectkey ASC`
	args := []interface{}{gorpmapping.IDsToQueryString(IDs), group.SharedInfraGroup.ID}
	return loadprojects(ctx, db, store, opts, query, args...)
}

// LoadAll returns all projects
func LoadAll(ctx context.Context, db gorp.SqlExecutor, store cache.Store, opts ...LoadOptionFunc) (sdk.Projects, error) {
	query := "select project.* from project ORDER by project.name, project.projectkey ASC"
	return loadprojects(ctx, db, store, opts, query)
}

// LoadPermissions loads all projects where group has access
func LoadPermissions(db gorp.SqlExecutor, groupID int64) ([]sdk.ProjectGroup, error) {
	res := []sdk.ProjectGroup{}
	query := `
		SELECT project.projectKey, project.name, project.last_modified, project_group.role
		FROM project
	 	JOIN project_group ON project_group.project_id = project.id
	 	WHERE project_group.group_id = $1
		ORDER BY project.name ASC`

	rows, err := db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var projectKey, projectName string
		var perm int
		var lastModified time.Time
		if err := rows.Scan(&projectKey, &projectName, &lastModified, &perm); err != nil {
			return nil, err
		}
		res = append(res, sdk.ProjectGroup{
			Project: sdk.Project{
				Key:          projectKey,
				Name:         projectName,
				LastModified: lastModified,
			},
			Permission: perm,
		})
	}
	return res, nil
}

// Exist checks whether a project exists or not
func Exist(db gorp.SqlExecutor, projectKey string) (bool, error) {
	query := `SELECT COUNT(id) FROM project WHERE project.projectKey = $1`
	var nb int64
	err := db.QueryRow(query, projectKey).Scan(&nb)
	if err != nil {
		return false, err
	}
	if nb != 0 {
		return true, nil
	}
	return false, nil
}

// Delete delete one or more projects given the key
func Delete(db gorp.SqlExecutor, store cache.Store, key string) error {
	proj, err := Load(db, store, key, nil)
	if err != nil {
		return err
	}

	return DeleteByID(db, proj.ID)
}

// BuiltinGPGKey is a const
const BuiltinGPGKey = "builtin"

// Insert a new project in database
func Insert(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	if err := proj.IsValid(); err != nil {
		return sdk.WrapError(err, "project is not valid")
	}

	proj.LastModified = time.Now()
	dbProj := dbProject(*proj)
	if err := db.Insert(&dbProj); err != nil {
		return err
	}
	*proj = sdk.Project(dbProj)

	k, err := keys.GeneratePGPKeyPair(BuiltinGPGKey)
	if err != nil {
		return sdk.WrapError(err, "Unable to generate PGPKeyPair: %v", err)
	}

	pk := sdk.ProjectKey{}
	pk.KeyID = k.KeyID
	pk.Name = BuiltinGPGKey
	pk.Private = k.Private
	pk.Public = k.Public
	pk.Type = sdk.KeyTypePGP
	pk.ProjectID = proj.ID
	pk.Builtin = true

	if err := InsertKey(db, &pk); err != nil {
		return sdk.WrapError(err, "Unable to insert PGPKeyPair")
	}

	return nil
}

// Update a new project in database
func Update(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	if err := proj.IsValid(); err != nil {
		return sdk.WrapError(err, "project is not valid")
	}

	proj.LastModified = time.Now()
	dbProj := dbProject(*proj)
	n, err := db.Update(&dbProj)
	if err != nil {
		return err
	}
	if n == 0 {
		return sdk.WithStack(sdk.ErrNoProject)
	}
	*proj = sdk.Project(dbProj)
	return nil
}

// DeleteByID removes given project from database (project and project_group table)
// DeleteByID also removes all pipelines inside project (pipeline and pipeline_group table).
func DeleteByID(db gorp.SqlExecutor, id int64) error {
	if err := DeleteAllVariables(db, id); err != nil {
		return err
	}

	if err := environment.DeleteAllEnvironment(db, id); err != nil {
		return err
	}

	if _, err := db.Exec(`DELETE FROM repositories_manager_project WHERE id_project = $1`, id); err != nil {
		return err
	}

	if _, err := db.Exec(`DELETE FROM project WHERE project.id = $1`, id); err != nil {
		return err
	}
	return nil
}

// LoadProjectByNodeJobRunID return a project from node job run id
func LoadProjectByNodeJobRunID(ctx context.Context, db gorp.SqlExecutor, store cache.Store, nodeJobRunID int64, opts ...LoadOptionFunc) (*sdk.Project, error) {
	query := `
		SELECT project.* FROM project
		JOIN workflow_run ON workflow_run.project_id = project.id
		JOIN workflow_node_run ON workflow_node_run.workflow_run_id = workflow_run.id
		JOIN workflow_node_run_job ON workflow_node_run_job.workflow_node_run_id = workflow_node_run.id
		WHERE workflow_node_run_job.id = $1
	`
	return load(ctx, db, store, opts, query, nodeJobRunID)
}

// LoadByID returns a project with all its variables and applications given a user. It can also returns pipelines, environments, groups, permission, and repositorires manager. See LoadOptions
func LoadByID(db gorp.SqlExecutor, store cache.Store, id int64, opts ...LoadOptionFunc) (*sdk.Project, error) {
	return load(context.TODO(), db, store, opts, "select project.* from project where id = $1", id)
}

// Load  returns a project with all its variables and applications given a user. It can also returns pipelines, environments, groups, permission, and repositorires manager. See LoadOptions
func Load(db gorp.SqlExecutor, store cache.Store, key string, opts ...LoadOptionFunc) (*sdk.Project, error) {
	return load(nil, db, store, opts, "select project.* from project where projectkey = $1", key)
}

// LoadProjectByWorkflowID loads a project from workflow iD
func LoadProjectByWorkflowID(db gorp.SqlExecutor, store cache.Store, workflowID int64, opts ...LoadOptionFunc) (*sdk.Project, error) {
	query := `SELECT project.id, project.name, project.projectKey, project.last_modified
	          FROM project
	          JOIN workflow ON workflow.project_id = project.id
	          WHERE workflow.id = $1 `
	return load(context.TODO(), db, store, opts, query, workflowID)
}

func loadprojects(ctx context.Context, db gorp.SqlExecutor, store cache.Store, opts []LoadOptionFunc, query string, args ...interface{}) ([]sdk.Project, error) {
	var res []dbProject
	if _, err := db.Select(&res, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNoProject)
		}
		return nil, sdk.WithStack(err)
	}

	projs := make([]sdk.Project, 0, len(res))
	for i := range res {
		p := &res[i]
		if err := p.PostGet(db); err != nil {
			log.Error(ctx, "loadprojects> PostGet error (ID=%d, Key:%s): %v", p.ID, p.Key, err)
			continue
		}
		proj, err := unwrap(db, store, p, opts)
		if err != nil {
			log.Error(ctx, "loadprojects> unwrap error (ID=%d, Key:%s): %v", p.ID, p.Key, err)
			continue
		}
		projs = append(projs, *proj)
	}

	return projs, nil
}

func load(ctx context.Context, db gorp.SqlExecutor, store cache.Store, opts []LoadOptionFunc, query string, args ...interface{}) (*sdk.Project, error) {
	var end func()
	_, end = observability.Span(ctx, "project.load")
	defer end()

	dbProj := &dbProject{}

	if err := db.SelectOne(dbProj, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNoProject)
		}
		return nil, sdk.WithStack(err)
	}

	return unwrap(db, store, dbProj, opts)
}

func unwrap(db gorp.SqlExecutor, store cache.Store, p *dbProject, opts []LoadOptionFunc) (*sdk.Project, error) {
	proj := sdk.Project(*p)

	for _, f := range opts {
		if f == nil {
			continue
		}
		if err := f(db, store, &proj); err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return nil, err
		}
	}

	return &proj, nil
}

// Labels return list of labels given a project ID
func Labels(db gorp.SqlExecutor, projectID int64) ([]sdk.Label, error) {
	var labels []sdk.Label
	query := `
	SELECT project_label.*
		FROM project_label
		WHERE project_label.project_id = $1
		ORDER BY project_label.name
	`
	if _, err := db.Select(&labels, query, projectID); err != nil {
		if err == sql.ErrNoRows {
			return labels, nil
		}
		return labels, sdk.WrapError(err, "Cannot load labels")
	}

	return labels, nil
}

// LabelByName return a label given his name and project id
func LabelByName(db gorp.SqlExecutor, projectID int64, labelName string) (sdk.Label, error) {
	var label sdk.Label
	err := db.SelectOne(&label, "SELECT project_label.* FROM project_label WHERE project_id = $1 AND name = $2", projectID, labelName)

	return label, err
}

// DeleteLabel delete a label given a label ID
func DeleteLabel(db gorp.SqlExecutor, labelID int64) error {
	query := "DELETE FROM project_label WHERE id = $1"
	if _, err := db.Exec(query, labelID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "Cannot delete labels")
	}

	return nil
}

// InsertLabel insert a label
func InsertLabel(db gorp.SqlExecutor, label *sdk.Label) error {
	if err := label.IsValid(); err != nil {
		return err
	}

	lbl := dbLabel(*label)
	if err := db.Insert(&lbl); err != nil {
		return sdk.WrapError(err, "Cannot insert labels")
	}
	*label = sdk.Label(lbl)

	return nil
}

// UpdateLabel update a label
func UpdateLabel(db gorp.SqlExecutor, label *sdk.Label) error {
	if err := label.IsValid(); err != nil {
		return err
	}

	lbl := dbLabel(*label)
	if _, err := db.Update(&lbl); err != nil {
		return sdk.WrapError(err, "Cannot update labels")
	}
	*label = sdk.Label(lbl)

	return nil
}

// UpdateFavorite add or delete project from user favorites
func UpdateFavorite(db gorp.SqlExecutor, projectID int64, userID string, add bool) error {
	var query string
	if add {
		query = "INSERT INTO project_favorite (authentified_user_id, project_id) VALUES ($1, $2)"
	} else {
		query = "DELETE FROM project_favorite WHERE authentified_user_id = $1 AND project_id = $2"
	}

	_, err := db.Exec(query, userID, projectID)
	return sdk.WithStack(err)
}
