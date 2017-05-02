package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

// LoadAll loads all workflows for a project. All users in a project can list all workflows in a project
func LoadAll(db gorp.SqlExecutor, projectKey string) ([]sdk.Workflow, error) {
	res := []sdk.Workflow{}
	dbRes := []Workflow{}

	query := `
		select * 
		from workflow
		join project on project.id = workflow.project_id
		order by workflow.name asc
		where project.projectkey = $1`

	if _, err := db.Select(&dbRes, query, projectKey); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "LoadAll> Unable to load workflows project %s", projectKey)
	}

	for _, w := range dbRes {
		if err := w.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "LoadAll> Unable to load workflows project %s", projectKey)
		}
		res = append(res, sdk.Workflow(w))
	}

	return res, nil
}

// Load loads a workflow for a given user (ie. checking permissions)
func Load(db gorp.SqlExecutor, projectKey, name string, u *sdk.User) (*sdk.Workflow, error) {
	return nil, nil
}

// LoadByID loads a workflow for a given user (ie. checking permissions)
func LoadByID(db gorp.SqlExecutor, id int64, name string, u *sdk.User) (*sdk.Workflow, error) {
	return nil, nil
}
