package environment

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// Insert a new environment key in database
func InsertKey(db gorp.SqlExecutor, key *sdk.EnvironmentKey) error {
	dbEnvironmentKey := dbEnvironmentKey(*key)

	s, errE := secret.Encrypt([]byte(key.Private))
	if errE != nil {
		return sdk.WrapError(errE, "InsertKey> Cannot encrypt private key")
	}
	key.Private = string(s)

	if err := db.Insert(&dbEnvironmentKey); err != nil {
		return sdk.WrapError(err, "InsertKey> Cannot insert project key")
	}
	*key = sdk.EnvironmentKey(dbEnvironmentKey)
	return nil
}

// LoadAllKeys load all keys for the given environment
func LoadAllKeys(db gorp.SqlExecutor, env *sdk.Environment) error {
	var res []dbEnvironmentKey
	if _, err := db.Select(&res, "SELECT * FROM environment_key WHERE environment_id = $1", env.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "LoadAllKeys> Cannot load keys")
	}

	keys := make([]sdk.EnvironmentKey, len(res))
	for i := range res {
		p := res[i]
		keys[i] = sdk.EnvironmentKey(p)
	}
	env.Keys = keys
	return nil
}

// DeleteEnvironmentKey Delete the given key from the given project
func DeleteEnvironmentKey(db gorp.SqlExecutor, envID int64, keyName string) error {
	_, err := db.Exec("DELETE FROM environment_key WHERE environment_id = $1 AND name = $2", envID, keyName)
	return sdk.WrapError(err, "DeleteEnvironmentKey> Cannot delete key %s", keyName)
}
