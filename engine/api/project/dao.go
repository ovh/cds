package project

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadAll returns all projects
func LoadAll(db gorp.SqlExecutor, u *sdk.User, opts ...LoadOptionFunc) ([]sdk.Project, error) {
	var query string
	var args []interface{}
	// Admin can gets all project
	// Users can gets only their projects
	if u == nil || u.Admin {
		query = "select * from project ORDER by project.name, project.projectkey ASC"
	} else {
		query = `SELECT * 
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
		var groupID string
		for i, g := range u.Groups {
			if i == 0 {
				groupID = fmt.Sprintf("%d", g.ID)
			} else {
				groupID += "," + fmt.Sprintf("%d", g.ID)
			}
		}
		args = []interface{}{groupID, group.SharedInfraGroup.ID}
	}
	return loadprojects(db, u, opts, query, args...)
}

// LoadPermissions loads all projects where group has access
func LoadPermissions(db gorp.SqlExecutor, group *sdk.Group) error {
	query := `
		SELECT project.projectKey, project.name, project.last_modified, project_group.role
		FROM project
	 	JOIN project_group ON project_group.project_id = project.id
	 	WHERE project_group.group_id = $1
		ORDER BY project.name ASC`

	rows, err := db.Query(query, group.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var projectKey, projectName string
		var perm int
		var lastModified time.Time
		if err := rows.Scan(&projectKey, &projectName, &lastModified, &perm); err != nil {
			return err
		}
		group.ProjectGroups = append(group.ProjectGroups, sdk.ProjectGroup{
			Project: sdk.Project{
				Key:          projectKey,
				Name:         projectName,
				LastModified: lastModified,
			},
			Permission: perm,
		})
	}
	return nil
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
	proj, err := Load(db, key, nil)
	if err != nil {
		return err
	}

	if err := DeleteByID(db, proj.ID); err != nil {
		return err
	}

	return nil
}

// Insert a new project in database
func Insert(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
	proj.LastModified = time.Now()
	dbProj := dbProject(*proj)
	if err := db.Insert(&dbProj); err != nil {
		return err
	}
	*proj = sdk.Project(dbProj)
	return UpdateLastModified(db, u, proj)
}

// Update a new project in database
func Update(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
	proj.LastModified = time.Now()
	dbProj := dbProject(*proj)
	n, err := db.Update(&dbProj)
	if err != nil {
		return err
	}
	if n == 0 {
		return sdk.ErrNoProject
	}
	*proj = sdk.Project(dbProj)
	return UpdateLastModified(db, u, proj)
}

// UpdateLastModified updates last_modified date on a project given its key
func UpdateLastModified(db gorp.SqlExecutor, u *sdk.User, proj *sdk.Project) error {
	t := time.Now()

	if u != nil {
		cache.SetWithTTL(cache.Key("lastModified", proj.Key), sdk.LastModification{
			Name:         proj.Key,
			Username:     u.Username,
			LastModified: t.Unix(),
		}, 0)
	}

	_, err := db.Exec("update project set last_modified = $2 where projectkey = $1", proj.Key, t)
	proj.LastModified = t

	if u != nil {
		updates := sdk.LastModification{
			Key:          proj.Key,
			Name:         proj.Name,
			LastModified: t.Unix(),
			Username:     u.Username,
			Type:         sdk.ProjectLastModiciationType,
		}
		b, errP := json.Marshal(updates)
		if errP == nil {
			cache.Publish("lastUpdates", string(b))
		}
		return err
	}
	return nil
}

// DeleteByID removes given project from database (project and project_group table)
// DeleteByID also removes all pipelines inside project (pipeline and pipeline_group table).
func DeleteByID(db gorp.SqlExecutor, id int64) error {
	log.Debug("project.Delete> Deleting project %d", id)
	if err := group.DeleteGroupProjectByProject(db, id); err != nil {
		return err
	}

	if err := DeleteAllVariable(db, id); err != nil {
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

type LoadOptionFunc *func(gorp.SqlExecutor, *sdk.Project, *sdk.User) error

// LoadOptions provides all options on project loads functions
var LoadOptions = struct {
	Default                  LoadOptionFunc
	WithApplications         LoadOptionFunc
	WithVariables            LoadOptionFunc
	WithPipelines            LoadOptionFunc
	WithEnvironments         LoadOptionFunc
	WithGroups               LoadOptionFunc
	WithPermission           LoadOptionFunc
	WithRepositoriesManagers LoadOptionFunc
	WithApplicationPipelines LoadOptionFunc
	WithApplicationVariables LoadOptionFunc
	WithKeys                 LoadOptionFunc
}{
	Default:                  &loadDefault,
	WithPipelines:            &loadPipelines,
	WithEnvironments:         &loadEnvironments,
	WithGroups:               &loadGroups,
	WithPermission:           &loadPermission,
	WithRepositoriesManagers: &loadRepositoriesManagers,
	WithApplications:         &loadApplications,
	WithVariables:            &loadVariables,
	WithApplicationPipelines: &loadApplicationPipelines,
	WithApplicationVariables: &loadApplicationVariables,
	WithKeys:                 &loadKeys,
}

// LoadByID returns a project with all its variables and applications given a user. It can also returns pipelines, environments, groups, permission, and repositorires manager. See LoadOptions
func LoadByID(db gorp.SqlExecutor, id int64, u *sdk.User, opts ...LoadOptionFunc) (*sdk.Project, error) {
	return load(db, u, opts, "select * from project where id = $1", id)
}

// Load  returns a project with all its variables and applications given a user. It can also returns pipelines, environments, groups, permission, and repositorires manager. See LoadOptions
func Load(db gorp.SqlExecutor, key string, u *sdk.User, opts ...LoadOptionFunc) (*sdk.Project, error) {
	return load(db, u, opts, "select * from project where projectkey = $1", key)
}

// LoadByPipelineID loads an project from pipeline iD
func LoadByPipelineID(db gorp.SqlExecutor, u *sdk.User, pipelineID int64, opts ...LoadOptionFunc) (*sdk.Project, error) {
	query := `SELECT project.id, project.name, project.projectKey, project.last_modified
	          FROM project
	          JOIN pipeline ON pipeline.project_id = projecT.id
	          WHERE pipeline.id = $1 `
	return load(db, u, opts, query, pipelineID)
}

func loadprojects(db gorp.SqlExecutor, u *sdk.User, opts []LoadOptionFunc, query string, args ...interface{}) ([]sdk.Project, error) {
	log.Debug("loadprojects> %s %v", query, args)
	var res []dbProject
	if _, err := db.Select(&res, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoProject
		}
		return nil, err
	}

	projs := make([]sdk.Project, len(res))
	for i := range res {
		p := &res[i]
		if err := p.PostGet(db); err != nil {
			return nil, err
		}
		proj, err := unwrap(db, p, u, opts)
		if err != nil {
			return nil, err
		}
		projs[i] = *proj
	}

	return projs, nil
}

func load(db gorp.SqlExecutor, u *sdk.User, opts []LoadOptionFunc, query string, args ...interface{}) (*sdk.Project, error) {
	dbProj := &dbProject{}
	if err := db.SelectOne(dbProj, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoProject
		}
		return nil, err
	}

	return unwrap(db, dbProj, u, opts)
}

func unwrap(db gorp.SqlExecutor, p *dbProject, u *sdk.User, opts []LoadOptionFunc) (*sdk.Project, error) {
	proj := sdk.Project(*p)

	for _, f := range opts {
		if err := (*f)(db, &proj, u); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	return &proj, nil
}
