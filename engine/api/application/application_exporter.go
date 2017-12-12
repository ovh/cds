package application

import (
	"fmt"
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export an application
func Export(db gorp.SqlExecutor, cache cache.Store, key string, appName string, f exportentities.Format, withPermissions bool, u *sdk.User, encryptFunc sdk.EncryptFunc, w io.Writer) (int, error) {
	// Load app
	app, errload := LoadByName(db, cache, key, appName, u,
		LoadOptions.WithVariablesWithClearPassword,
		LoadOptions.WithKeys,
	)
	if errload != nil {
		return 0, sdk.WrapError(errload, "application.Export> Cannot load application %s", appName)
	}

	// Load permissions
	if withPermissions {
		perms, err := group.LoadGroupsByApplication(db, app.ID)
		if err != nil {
			return 0, sdk.WrapError(err, "application.Export> Cannot load application %s permissions", appName)
		}
		app.ApplicationGroups = perms
	}
	return ExportApplication(db, *app, f, withPermissions, encryptFunc, w)
}

// ExportApplication encrypt and export
func ExportApplication(db gorp.SqlExecutor, app sdk.Application, f exportentities.Format, withPermissions bool, encryptFunc sdk.EncryptFunc, w io.Writer) (int, error) {
	// Parse variables
	appvars := []sdk.Variable{}
	for _, v := range app.Variable {
		switch v.Type {
		case sdk.KeyVariable:
			return 0, sdk.WrapError(fmt.Errorf("Unsupported variable %s", v.Name), "application.Export> Unable to export application")
		case sdk.SecretVariable:
			content, err := encryptFunc(db, app.ProjectID, fmt.Sprintf("appID:%d:%s", app.ID, v.Name), v.Value)
			if err != nil {
				return 0, sdk.WrapError(err, "application.Export> Unknown key type")
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
			return 0, sdk.WrapError(err, "application.Export> Unable to encrypt key")
		}
		ek := exportentities.EncryptedKey{
			Type:    k.Type,
			Name:    k.Name,
			Content: content,
		}
		keys = append(keys, ek)
	}

	eapp, err := exportentities.NewApplication(app, withPermissions, keys)
	if err != nil {
		return 0, sdk.WrapError(err, "application.Export> Unable to export application")
	}

	// Marshal to the desired format
	b, err := exportentities.Marshal(eapp, f)
	if err != nil {
		return 0, sdk.WrapError(err, "application.Export>")
	}

	return w.Write(b)
}
