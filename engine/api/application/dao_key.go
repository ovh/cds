package application

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertKey a new application key in database
func InsertKey(db gorp.SqlExecutor, key *sdk.ApplicationKey) error {
	dbAppKey := dbApplicationKey(*key)

	s, errE := secret.Encrypt([]byte(key.Private))
	if errE != nil {
		return sdk.WrapError(errE, "InsertKey> Cannot encrypt private key")
	}
	dbAppKey.Private = string(s)

	if err := db.Insert(&dbAppKey); err != nil {
		return sdk.WrapError(err, "InsertKey> Cannot insert application key")
	}
	*key = sdk.ApplicationKey(dbAppKey)
	return nil
}

// LoadAllKeys load all keys for the given application
func LoadAllKeys(db gorp.SqlExecutor, app *sdk.Application) error {
	var res []dbApplicationKey
	if _, err := db.Select(&res, "SELECT * FROM application_key WHERE application_id = $1", app.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "LoadAllKeys> Cannot load keys")
	}

	keys := make([]sdk.ApplicationKey, len(res))
	for i := range res {
		p := res[i]
		keys[i] = sdk.ApplicationKey(p)
		decrypted, err := secret.Decrypt([]byte(keys[i].Private))
		if err != nil {
			log.Error("LoadAllKeys> Unable to decrypt private key %s/%s: %v", app.Name, keys[i].Name, err)
		}
		keys[i].Private = string(decrypted)
	}
	app.Keys = keys
	return nil
}

// DeleteApplicationKey Delete the given key from the given application
func DeleteApplicationKey(db gorp.SqlExecutor, appID int64, keyName string) error {
	_, err := db.Exec("DELETE FROM application_key WHERE application_id = $1 AND name = $2", appID, keyName)
	return sdk.WrapError(err, "DeleteApplicationKey> Cannot delete key %s", keyName)
}
