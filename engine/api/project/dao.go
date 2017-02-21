package project

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadAll returns all projects
func LoadAll(db gorp.SqlExecutor, u *sdk.User, opts ...loadOptionFunc) ([]sdk.Project, error) {
	var query string
	var args []interface{}
	// Admin can gets all project
	// Users can gets only their projects
	if u == nil || u.Admin {
		query = "select * from project ORDER by project.name, project.projectkey ASC"
	} else {
		query = `select * 
            from project 
            JOIN project_group ON project.id = project_group.project_id
            JOIN group_user ON project_group.group_id = group_user.group_id
            WHERE group_user.user_id = $1
            ORDER by project.name, project.projectkey ASC`
		args = []interface{}{u.ID}
	}
	return loadprojects(db, u, opts, query, args...)
}

func loadprojects(db gorp.SqlExecutor, u *sdk.User, opts []loadOptionFunc, query string, args ...interface{}) ([]sdk.Project, error) {
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

func unwrap(db gorp.SqlExecutor, p *dbProject, u *sdk.User, opts []loadOptionFunc) (*sdk.Project, error) {
	proj := sdk.Project(*p)

	//By default we load applications and variables
	if err := loadApplications(db, &proj, u); err != nil {
		return nil, err
	}

	if err := loadAllVariables(db, &proj); err != nil {
		return nil, err
	}

	for _, f := range opts {
		if err := f(db, &proj, u); err != nil {
			return nil, err
		}
	}

	return &proj, nil
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

// DeleteByID removes given project from database (project and project_group table)
// DeleteByID also removes all pipelines inside project (pipeline and pipeline_group table).
func DeleteByID(db gorp.SqlExecutor, id int64) error {
	log.Debug("project.Delete> Deleting project %d", id)
	if err := group.DeleteGroupProjectByProject(db, id); err != nil {
		return err
	}

	if err := DeleteAllVariableFromProject(db, id); err != nil {
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

type loadOptionFunc func(gorp.SqlExecutor, *sdk.Project, *sdk.User) error

// LoadOptions provides all options on project loads functions
var LoadOptions = struct {
	WithPipelines            loadOptionFunc
	WithEnvironments         loadOptionFunc
	WithGroups               loadOptionFunc
	WithPermission           loadOptionFunc
	WithRepositoriesManagers loadOptionFunc
}{
	WithPipelines:            loadPipelines,
	WithEnvironments:         loadEnvironments,
	WithGroups:               loadGroups,
	WithPermission:           loadPermission,
	WithRepositoriesManagers: loadRepositoriesManagers,
}

// Load  returns a project with all its variables and applications given a user
func Load(db gorp.SqlExecutor, key string, u *sdk.User, opts ...loadOptionFunc) (*sdk.Project, error) {
	return load(db, u, opts, "select * from project where projectkey = $1", key)
}

func load(db gorp.SqlExecutor, u *sdk.User, opts []loadOptionFunc, query string, args ...interface{}) (*sdk.Project, error) {
	dbProj := &dbProject{}
	if err := db.SelectOne(dbProj, query, args...); err != nil {
		return nil, err
	}

	return unwrap(db, dbProj, u, opts)
}
