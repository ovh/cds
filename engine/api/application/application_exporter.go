package application

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export an application
func Export(ctx context.Context, db gorp.SqlExecutor, key string, appName string, encryptFunc sdk.EncryptFunc) (exportentities.Application, error) {
	app, err := LoadByNameWithClearVCSStrategyPassword(ctx, db, key, appName,
		LoadOptions.WithVariablesWithClearPassword,
		LoadOptions.WithClearKeys,
		LoadOptions.WithClearDeploymentStrategies,
	)
	if err != nil {
		return exportentities.Application{}, sdk.WrapError(err, "cannot load application %s", appName)
	}

	return ExportApplication(ctx, db, *app, encryptFunc, fmt.Sprintf("appID:%d", app.ID))
}

// ExportApplication encrypt and export
func ExportApplication(ctx context.Context, db gorp.SqlExecutor, app sdk.Application, encryptFunc sdk.EncryptFunc, encryptPrefix string) (exportentities.Application, error) {
	var appvars []sdk.ApplicationVariable
	for _, v := range app.Variables {
		switch v.Type {
		case sdk.KeyVariable:
			return exportentities.Application{}, sdk.NewErrorFrom(sdk.ErrUnknownError,
				"variable %s: variable of type key are deprecated. Please use the standard keys from your project or your application", v.Name)
		case sdk.SecretVariable:
			content, err := encryptFunc(ctx, db, app.ProjectID, fmt.Sprintf("%s:%s", encryptPrefix, v.Name), v.Value)
			if err != nil {
				return exportentities.Application{}, sdk.WrapError(err, "unknown key type")
			}
			v.Value = content
			appvars = append(appvars, v)
		default:
			appvars = append(appvars, v)
		}
	}
	app.Variables = appvars

	// Prepare keys
	var keys []exportentities.EncryptedKey
	// Parse keys
	for _, k := range app.Keys {
		content, err := encryptFunc(ctx, db, app.ProjectID, fmt.Sprintf("%s:%s", encryptPrefix, k.Name), k.Private)
		if err != nil {
			return exportentities.Application{}, sdk.WrapError(err, "unable to encrypt key")
		}
		ek := exportentities.EncryptedKey{
			Type:    string(k.Type),
			Name:    k.Name,
			Content: content,
		}
		keys = append(keys, ek)
	}

	if app.RepositoryStrategy.Password != "" {
		content, err := encryptFunc(ctx, db, app.ProjectID, fmt.Sprintf("%s:%s", encryptPrefix, "vcs:password"), app.RepositoryStrategy.Password)
		if err != nil {
			return exportentities.Application{}, sdk.WrapError(err, "unable to encrypt password")
		}
		app.RepositoryStrategy.Password = content
	}

	for pfName, pfConfig := range app.DeploymentStrategies {
		for k, v := range pfConfig {
			if v.Type == sdk.SecretVariable {
				content, err := encryptFunc(ctx, db, app.ProjectID, fmt.Sprintf("%s:%s:%s:%s", encryptPrefix, pfName, k, "deployment:password"), v.Value)
				if err != nil {
					return exportentities.Application{}, sdk.WrapError(err, "Unable to encrypt password")
				}
				v.Value = content
				app.DeploymentStrategies[pfName][k] = v
			}
		}
	}

	return exportentities.NewApplication(app, keys)
}
