package workflow

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

// LoadAll loads all workflows for a project given a user (ie. checking permissions)
func LoadAll(db gorp.SqlExecutor, projectKey string, u *sdk.User) ([]sdk.Workflow, error) {
	res := []sdk.Workflow{}
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
