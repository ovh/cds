package project

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// Insert a new project key in database
func InsertKey(db gorp.SqlExecutor, key *sdk.ProjectKey, u *sdk.User) error {
	dbProjKey := dbProjectKey(*key)
	if err := db.Insert(&dbProjKey); err != nil {
		return err
	}
	*key = sdk.ProjectKey(dbProjKey)
	return nil
}

// LoadAllKeys load all keys for the given project
func LoadAllKeys(db gorp.SqlExecutor, proj *sdk.Project) error {
	var res []dbProjectKey
	if _, err := db.Select(&res, "SELECT * FROM project_key WHERE project_id = $1", proj.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	keys := make([]sdk.ProjectKey, len(res))
	for i := range res {
		p := res[i]
		keys[i] = sdk.ProjectKey(p)
	}
	proj.Keys = keys
	return nil
}

// DeleteProjectKey Delete the given key from the given project
func DeleteProjectKey(db gorp.SqlExecutor, projectID int64, keyName string) error {
	_, err := db.Exec("DELETE FROM project_key WHERE project_id = $1 AND name = $2", projectID, keyName)
	return err
}
