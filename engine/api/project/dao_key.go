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

// LoadAllKeysByID Load all project key for the given project
func LoadAllKeysByID(db gorp.SqlExecutor, ID int64) ([]sdk.ProjectKey, error) {
	var res []dbProjectKey
	if _, err := db.Select(&res, "SELECT * FROM project_key WHERE project_id = $1 and builtin = false", ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "LoadAllKeys> Cannot load keys")
	}

	keys := make([]sdk.ProjectKey, len(res))
	for i := range res {
		p := res[i]
		keys[i] = sdk.ProjectKey(p)
	}
	return keys, nil
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
		keys[i].Private = sdk.PasswordPlaceholder
	}
	proj.Keys = keys
	return nil
}

// LoadAllDecryptedKeys load all keys for the given project
func LoadAllDecryptedKeys(db gorp.SqlExecutor, proj *sdk.Project) error {
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

func loadBuildinKey(db gorp.SqlExecutor, projectID int64) (sdk.ProjectKey, error) {
	var k sdk.ProjectKey
	var res dbProjectKey
	if err := db.SelectOne(&res, "SELECT * FROM project_key WHERE project_id = $1 and builtin = true and name = 'builtin'", projectID); err != nil {
		if err == sql.ErrNoRows {
			return k, sdk.ErrBuiltinKeyNotFound
		}
		return k, sdk.WrapError(err, "loadBuildinKey> Cannot load keys")
	}

	k = sdk.ProjectKey(res)
	decrypted, err := secret.Decrypt([]byte(k.Private))
	if err != nil {
		return k, sdk.WrapError(err, "loadBuildinKey> Unable to decrypt key")
	}
	k.Private = string(decrypted)

	return k, nil
}
