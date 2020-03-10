package environment

import (
	"context"
	"fmt"
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export an environment
func Export(ctx context.Context, db gorp.SqlExecutor, cache cache.Store, key string, envName string, f exportentities.Format, encryptFunc sdk.EncryptFunc, w io.Writer) (int, error) {
	// Load app
	env, err := LoadEnvironmentByName(db, key, envName)
	if err != nil {
		return 0, sdk.WrapError(err, "environment.Export> Cannot load %s", envName)
	}

	// reload variables with clear password
	variables, err := LoadAllVariablesWithDecrytion(db, env.ID)
	if err != nil {
		return 0, err
	}
	env.Variables = variables

	// Reload key
	keys, err := LoadAllKeysWithPrivateContent(db, env.ID)
	if err != nil {
		return 0, sdk.WrapError(err, "environment.Export> Cannot load env %s keys", envName)
	}
	env.Keys = keys

	return ExportEnvironment(db, *env, f, encryptFunc, w)
}

// ExportEnvironment encrypt and export
func ExportEnvironment(db gorp.SqlExecutor, env sdk.Environment, f exportentities.Format, encryptFunc sdk.EncryptFunc, w io.Writer) (int, error) {
	// Parse variables
	envvars := []sdk.Variable{}
	for _, v := range env.Variables {
		switch v.Type {
		case sdk.KeyVariable:
			return 0, sdk.WrapError(fmt.Errorf("Unsupported variable %s", v.Name), "environment.Export> Unable to export application")
		case sdk.SecretVariable:
			content, err := encryptFunc(db, env.ProjectID, fmt.Sprintf("envID:%d:%s", env.ID, v.Name), v.Value)
			if err != nil {
				return 0, sdk.WrapError(err, "Unknown key type")
			}
			v.Value = content
			envvars = append(envvars, v)
		default:
			envvars = append(envvars, v)
		}
	}
	env.Variables = envvars

	// Prepare keys
	keys := []exportentities.EncryptedKey{}

	// Parse keys
	for _, k := range env.Keys {
		content, err := encryptFunc(db, env.ProjectID, fmt.Sprintf("envID:%d:%s", env.ID, k.Name), k.Private)
		if err != nil {
			return 0, sdk.WrapError(err, "Unable to encrypt key")
		}
		ek := exportentities.EncryptedKey{
			Type:    k.Type,
			Name:    k.Name,
			Content: content,
		}
		keys = append(keys, ek)
	}

	e := exportentities.NewEnvironment(env, keys)
	btes, errMarshal := exportentities.Marshal(e, f)
	if errMarshal != nil {
		return 0, sdk.WrapError(errMarshal, "environment.Export")
	}

	return w.Write(btes)
}
