package project

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

//LoadAllVariables loads all variable into a project
func LoadAllVariables(db gorp.SqlExecutor, proj *sdk.Project, args ...GetAllVariableFuncArg) error {
	vars, err := GetAllVariableInProject(db, proj.ID, args...)
	if err != nil {
		return err
	}
	proj.Variable = vars
	return nil
}
