package environment

import (
	"database/sql"
	"encoding/base64"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertKey a new environment key in database
func InsertKey(db gorp.SqlExecutor, key *sdk.EnvironmentKey) error {
	dbEnvironmentKey := dbEnvironmentKey(*key)

	s, errE := secret.Encrypt([]byte(key.Private))
	if errE != nil {
		return sdk.WrapError(errE, "InsertKey> Cannot encrypt private key")
	}
	dbEnvironmentKey.Private = string(s)

	if err := db.Insert(&dbEnvironmentKey); err != nil {
		return sdk.WrapError(err, "InsertKey> Cannot insert project key")
	}
	*key = sdk.EnvironmentKey(dbEnvironmentKey)
	return nil
}

// LoadAllEnvironmentKeysByProject Load all environment key for the given project
func LoadAllEnvironmentKeysByProject(db gorp.SqlExecutor, projID int64) ([]sdk.EnvironmentKey, error) {
	var res []dbEnvironmentKey
	query := `
	SELECT DISTINCT ON (environment_key.name, environment_key.type) environment_key.name as d, environment_key.* FROM environment_key
	JOIN environment ON environment.id = environment_key.environment_id
	JOIN project ON project.id = environment.project_id
	WHERE project.id = $1;
	`
	if _, err := db.Select(&res, query, projID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "LoadAllKeys> Cannot load keys")
	}

	keys := make([]sdk.EnvironmentKey, len(res))
	for i := range res {
		p := res[i]
		keys[i] = sdk.EnvironmentKey(p)
	}
	return keys, nil
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
		keys[i].Private = sdk.PasswordPlaceholder
	}
	env.Keys = keys
	return nil
}

// LoadAllBase64Keys Load environment key with encrypted secret
func LoadAllBase64Keys(db gorp.SqlExecutor, env *sdk.Environment) error {
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
		keys[i].Private = base64.StdEncoding.EncodeToString([]byte(keys[i].Private))
	}
	env.Keys = keys
	return nil
}

// LoadAllDecryptedKeys load all keys for the given environment
func LoadAllDecryptedKeys(db gorp.SqlExecutor, env *sdk.Environment) error {
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
		decrypted, err := secret.Decrypt([]byte(keys[i].Private))
		if err != nil {
			log.Error("LoadAllKeys> Unable to decrypt private key %s/%s: %v", env.Name, keys[i].Name, err)
		}
		keys[i].Private = string(decrypted)
	}
	env.Keys = keys
	return nil
}

// DeleteEnvironmentKey Delete the given key from the given project
func DeleteEnvironmentKey(db gorp.SqlExecutor, envID int64, keyName string) error {
	_, err := db.Exec("DELETE FROM environment_key WHERE environment_id = $1 AND name = $2", envID, keyName)
	return sdk.WrapError(err, "DeleteEnvironmentKey> Cannot delete key %s", keyName)
}
