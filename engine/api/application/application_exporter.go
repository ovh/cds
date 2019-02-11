package application

import (
	"fmt"
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export an application
func Export(db gorp.SqlExecutor, cache cache.Store, key string, appName string, f exportentities.Format, encryptFunc sdk.EncryptFunc, w io.Writer) (int, error) {
	// Load app
	app, errload := LoadByName(db, cache, key, appName,
		LoadOptions.WithVariablesWithClearPassword, LoadOptions.WithClearKeys, LoadOptions.WithClearDeploymentStrategies,
	)
	if errload != nil {
		return 0, sdk.WrapError(errload, "application.Export> Cannot load application %s", appName)
	}

	if errD := DecryptVCSStrategyPassword(app); errD != nil {
		return 0, sdk.WrapError(errD, "application.Export> Cannot decrypt vcs password")
	}

	return ExportApplication(db, *app, f, encryptFunc, w)
}

// ExportApplication encrypt and export
func ExportApplication(db gorp.SqlExecutor, app sdk.Application, f exportentities.Format, encryptFunc sdk.EncryptFunc, w io.Writer) (int, error) {
	// Parse variables
	appvars := []sdk.Variable{}
	for _, v := range app.Variable {
		switch v.Type {
		case sdk.KeyVariable:
			return 0, sdk.NewErrorFrom(sdk.ErrUnknownError, "variable %s: variable of type key are deprecated. Please use the standard keys from your project or your application", v.Name)
		case sdk.SecretVariable:
			content, err := encryptFunc(db, app.ProjectID, fmt.Sprintf("appID:%d:%s", app.ID, v.Name), v.Value)
			if err != nil {
				return 0, sdk.WrapError(err, "Unknown key type")
			}
			v.Value = content
			appvars = append(appvars, v)
		default:
			appvars = append(appvars, v)
		}
	}
	app.Variable = appvars

	// Prepare keys
	keys := []exportentities.EncryptedKey{}
	// Parse keys
	for _, k := range app.Keys {
		content, err := encryptFunc(db, app.ProjectID, fmt.Sprintf("appID:%d:%s", app.ID, k.Name), k.Private)
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

	if app.RepositoryStrategy.Password != "" {
		content, err := encryptFunc(db, app.ProjectID, fmt.Sprintf("appID:%d:%s", app.ID, "vcs:password"), app.RepositoryStrategy.Password)
		if err != nil {
			return 0, sdk.WrapError(err, "Unable to encrypt password")
		}
		app.RepositoryStrategy.Password = content
	}

	for pfName, pfConfig := range app.DeploymentStrategies {
		for k, v := range pfConfig {
			if v.Type == sdk.SecretVariable {
				content, err := encryptFunc(db, app.ProjectID, fmt.Sprintf("appID:%d:%s:%s:%s", app.ID, pfName, k, "deployment:password"), v.Value)
				if err != nil {
					return 0, sdk.WrapError(err, "Unable to encrypt password")
				}
				v.Value = content
				app.DeploymentStrategies[pfName][k] = v
			}
		}
	}

	eapp, err := exportentities.NewApplication(app, keys)
	if err != nil {
		return 0, sdk.WrapError(err, "Unable to export application")
	}

	// Marshal to the desired format
	b, err := exportentities.Marshal(eapp, f)
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	return w.Write(b)
}
