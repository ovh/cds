package project

import (
	"context"
	"database/sql"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

func loadAllByRepo(ctx context.Context, db gorp.SqlExecutor, query string, args []interface{}, opts ...LoadOptionFunc) (sdk.Projects, error) {
	return loadprojects(ctx, db, opts, query, args...)
}

// LoadAllByRepoAndGroupIDs returns all projects with an application linked to the repo against the groups
func LoadAllByRepoAndGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64, repo string, opts ...LoadOptionFunc) (sdk.Projects, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "project.LoadAllByRepoAndGroupIDs")
	defer end()
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
	return loadAllByRepo(ctx, db, query, args, opts...)
}

// LoadAllByRepo returns all projects with an application linked to the repo
func LoadAllByRepo(ctx context.Context, db gorp.SqlExecutor, store cache.Store, repo string, opts ...LoadOptionFunc) (sdk.Projects, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "project.LoadAllByRepo")
	defer end()
	query := `SELECT DISTINCT project.*
	FROM  project
	JOIN  application on project.id = application.project_id
	WHERE application.repo_fullname = $1
	ORDER by project.name, project.projectkey ASC`
	args := []interface{}{repo}
	return loadAllByRepo(ctx, db, query, args, opts...)
}

// LoadAllByGroupIDs returns all projects given groups
func LoadAllByGroupIDs(ctx context.Context, db gorp.SqlExecutor, store cache.Store, IDs []int64, opts ...LoadOptionFunc) (sdk.Projects, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "project.LoadAllByGroupIDs")
	defer end()
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
	return loadprojects(ctx, db, opts, query, args...)
}

// LoadAll returns all projects
func LoadAll(ctx context.Context, db gorp.SqlExecutor, store cache.Store, opts ...LoadOptionFunc) (sdk.Projects, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "project.LoadAll")
	defer end()
	query := "select project.* from project ORDER by project.name, project.projectkey ASC"
	return loadprojects(ctx, db, opts, query)
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
func Delete(db gorp.SqlExecutor, key string) error {
	proj, err := Load(context.Background(), db, key, nil)
	if err != nil {
		return err
	}

	return DeleteByID(db, proj.ID)
}

// BuiltinGPGKey is a const
const BuiltinGPGKey = "builtin"

// Insert a new project in database
func Insert(db gorpmapper.SqlExecutorWithTx, proj *sdk.Project) error {
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
func Update(db gorp.SqlExecutor, proj *sdk.Project) error {
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

	if _, err := db.Exec(`DELETE FROM project WHERE project.id = $1`, id); err != nil {
		return sdk.WithStack(err)
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
	return load(ctx, db, opts, query, nodeJobRunID)
}

// LoadByID returns a project with all its variables and applications given a user. It can also returns pipelines, environments, groups, permission, and repositorires manager. See LoadOptions
func LoadByID(db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.Project, error) {
	return load(context.Background(), db, opts, "select project.* from project where id = $1", id)
}

// Load  returns a project with all its variables and applications given a user. It can also returns pipelines, environments, groups, permission, and repositorires manager. See LoadOptions
func Load(ctx context.Context, db gorp.SqlExecutor, key string, opts ...LoadOptionFunc) (*sdk.Project, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "project.Load")
	defer end()
	return load(ctx, db, opts, "select project.* from project where projectkey = $1", key)
}

// LoadProjectByWorkflowID loads a project from workflow iD
func LoadProjectByWorkflowID(db gorp.SqlExecutor, workflowID int64, opts ...LoadOptionFunc) (*sdk.Project, error) {
	query := `SELECT project.id, project.name, project.projectKey, project.last_modified
	          FROM project
	          JOIN workflow ON workflow.project_id = project.id
	          WHERE workflow.id = $1 `
	return load(context.Background(), db, opts, query, workflowID)
}

func loadprojects(ctx context.Context, db gorp.SqlExecutor, opts []LoadOptionFunc, query string, args ...interface{}) ([]sdk.Project, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "project.loadprojects")
	defer end()

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
		proj, err := unwrap(ctx, db, p, opts)
		if err != nil {
			log.Error(ctx, "loadprojects> unwrap error (ID=%d, Key:%s): %v", p.ID, p.Key, err)
			continue
		}
		projs = append(projs, *proj)
	}

	return projs, nil
}

func load(ctx context.Context, db gorp.SqlExecutor, opts []LoadOptionFunc, query string, args ...interface{}) (*sdk.Project, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "project.load")
	defer end()

	dbProj := &dbProject{}

	if err := db.SelectOne(dbProj, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNoProject)
		}
		return nil, sdk.WithStack(err)
	}

	return unwrap(ctx, db, dbProj, opts)
}

func unwrap(ctx context.Context, db gorp.SqlExecutor, p *dbProject, opts []LoadOptionFunc) (*sdk.Project, error) {
	ctx, end := telemetry.Span(ctx, "project.unwrap")
	defer end()

	proj := sdk.Project(*p)

	for _, f := range opts {
		if f == nil {
			continue
		}
		name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		nameSplitted := strings.Split(name, "/")
		name = nameSplitted[len(nameSplitted)-1]
		_, end = telemetry.Span(ctx, name)
		if err := f(db, &proj); err != nil && sdk.Cause(err) != sql.ErrNoRows {
			end()
			return nil, err
		}
		end()
	}

	vcsServers, err := repositoriesmanager.LoadAllProjectVCSServerLinksByProjectID(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	proj.VCSServers = vcsServers

	return &proj, nil
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
