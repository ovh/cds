package environment

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export an environment
func Export(ctx context.Context, db gorp.SqlExecutor, key string, envName string, encryptFunc sdk.EncryptFunc) (exportentities.Environment, error) {
	// Load app
	env, err := LoadEnvironmentByName(db, key, envName)
	if err != nil {
		return exportentities.Environment{}, sdk.WrapError(err, "cannot load %s", envName)
	}

	// Reload variables with clear password
	variables, err := LoadAllVariablesWithDecryption(db, env.ID)
	if err != nil {
		return exportentities.Environment{}, err
	}
	env.Variables = variables

	// Reload key
	keys, err := LoadAllKeysWithPrivateContent(db, env.ID)
	if err != nil {
		return exportentities.Environment{}, sdk.WrapError(err, "cannot load env %s keys", envName)
	}
	env.Keys = keys

	return ExportEnvironment(ctx, db, *env, encryptFunc, fmt.Sprintf("env:%d", env.ID))
}

// ExportEnvironment encrypt and export
func ExportEnvironment(ctx context.Context, db gorp.SqlExecutor, env sdk.Environment, encryptFunc sdk.EncryptFunc, encryptPrefix string) (exportentities.Environment, error) {
	var envvars []sdk.EnvironmentVariable
	for _, v := range env.Variables {
		switch v.Type {
		case sdk.KeyVariable:
			return exportentities.Environment{}, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unsupported variable %s", v.Name)
		case sdk.SecretVariable:
			content, err := encryptFunc(ctx, db, env.ProjectID, fmt.Sprintf("envID:%d:%s", env.ID, v.Name), v.Value)
			if err != nil {
				return exportentities.Environment{}, sdk.WrapError(err, "unable to encrypt var for env %d in project %d", env.ID, env.ProjectID)
			}
			v.Value = content
			envvars = append(envvars, v)
		default:
			envvars = append(envvars, v)
		}
	}
	env.Variables = envvars

	var keys []exportentities.EncryptedKey
	for _, k := range env.Keys {
		content, err := encryptFunc(ctx, db, env.ProjectID, fmt.Sprintf("envID:%d:%s", env.ID, k.Name), k.Private)
		if err != nil {
			return exportentities.Environment{}, sdk.WrapError(err, "unable to encrypt key for env %d in project %d", env.ID, env.ProjectID)
		}
		ek := exportentities.EncryptedKey{
			Type:    string(k.Type),
			Name:    k.Name,
			Content: content,
		}
		keys = append(keys, ek)
	}

	return exportentities.NewEnvironment(env, keys), nil
}
