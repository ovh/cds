package environment

import (
	"fmt"
	"io"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

type encryptFunc func(gorp.SqlExecutor, int64, string, string) (string, error)

// Export an environment
func Export(db gorp.SqlExecutor, cache cache.Store, key string, envName string, f exportentities.Format, withPermissions bool, u *sdk.User, encryptFunc encryptFunc, w io.Writer) error {
	// Load app
	env, errload := LoadEnvironmentByName(db, key, envName)
	if errload != nil {
		return sdk.WrapError(errload, "environment.Export> Cannot load %s", envName)
	}

	// Load permissions
	if withPermissions {
		perms, err := group.LoadGroupsByEnvironment(db, env.ID)
		if err != nil {
			return sdk.WrapError(err, "environment.Export> Cannot load %s permissions", envName)
		}
		env.EnvironmentGroups = perms
	}

	// Parse variables
	envvars := []sdk.Variable{}
	for _, v := range env.Variable {
		switch v.Type {
		case sdk.KeyVariable:
			return sdk.WrapError(fmt.Errorf("Unsupported variable %s", v.Name), "environment.Export> Unable to export application")
		case sdk.SecretVariable:
			content, err := encryptFunc(db, env.ProjectID, fmt.Sprintf("envID:%d:%s", env.ID, v.Name), v.Value)
			if err != nil {
				return sdk.WrapError(err, "environment.Export> Unknown key type")
			}
			v.Value = content
			envvars = append(envvars, v)
		default:
			envvars = append(envvars, v)
		}
	}
	env.Variable = envvars

	// Prepare keys
	keys := []exportentities.EncryptedKey{}

	// Parse keys
	for _, k := range env.Keys {
		content, err := encryptFunc(db, env.ProjectID, fmt.Sprintf("envID:%d:%s", env.ID, k.Name), k.Private)
		if err != nil {
			return sdk.WrapError(err, "environment.Export> Unable to encrypt key")
		}
		ek := exportentities.EncryptedKey{
			Type:    k.Type,
			Name:    k.Name,
			Content: content,
		}
		keys = append(keys, ek)
	}

	e := exportentities.NewEnvironment(env, withPermissions, keys)
	btes, errMarshal := exportentities.Marshal(e, f)
	if errMarshal != nil {
		return sdk.WrapError(errMarshal, "environment.Export")
	}

	_, err := w.Write(btes)
	return err
}
