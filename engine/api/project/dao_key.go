package project

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertKey a new project key in database
func InsertKey(db gorp.SqlExecutor, key *sdk.ProjectKey) error {
	dbProjKey := dbProjectKey(*key)

	s, errE := secret.Encrypt([]byte(key.Private))
	if errE != nil {
		return sdk.WrapError(errE, "InsertKey> Cannot encrypt private key")
	}
	dbProjKey.Private = string(s)

	if err := db.Insert(&dbProjKey); err != nil {
		return sdk.WrapError(err, "InsertKey> Cannot insert project key")
	}
	*key = sdk.ProjectKey(dbProjKey)
	return nil
}

// LoadAllKeys load all keys for the given project
func LoadAllKeys(db gorp.SqlExecutor, proj *sdk.Project) error {
	var res []dbProjectKey
	if _, err := db.Select(&res, "SELECT * FROM project_key WHERE project_id = $1 and builtin = false", proj.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "LoadAllKeys> Cannot load keys")
	}

	keys := make([]sdk.ProjectKey, len(res))
	for i := range res {
		p := res[i]
		keys[i] = sdk.ProjectKey(p)
		decrypted, err := secret.Decrypt([]byte(keys[i].Private))
		if err != nil {
			log.Error("LoadAllKeys> Unable to decrypt private key %s/%s: %v", proj.Key, keys[i].Name, err)
		}
		keys[i].Private = string(decrypted)
	}
	proj.Keys = keys
	return nil
}

// DeleteProjectKey Delete the given key from the given project
func DeleteProjectKey(db gorp.SqlExecutor, projectID int64, keyName string) error {
	_, err := db.Exec("DELETE FROM project_key WHERE project_id = $1 AND name = $2", projectID, keyName)
	return sdk.WrapError(err, "DeleteProjectKey> Cannot delete key %s", keyName)
}
