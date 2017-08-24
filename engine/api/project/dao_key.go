package project

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"database/sql"
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

func LoadAllKeys(db gorp.SqlExecutor, proj *sdk.Project) error {
	var res []dbProjectKey
	if _, err := db.Select(&res, "SELECT * FROM project_key WHERE project_ID = $1", proj.ID); err != nil {
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


