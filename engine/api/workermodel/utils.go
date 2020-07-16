package workermodel

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

var defaultEnvs = map[string]string{
	"CDS_SINGLE_USE":          "1",
	"CDS_TTL":                 "{{.TTL}}",
	"CDS_GRAYLOG_HOST":        "{{.GraylogHost}}",
	"CDS_GRAYLOG_PORT":        "{{.GraylogPort}}",
	"CDS_GRAYLOG_EXTRA_KEY":   "{{.GraylogExtraKey}}",
	"CDS_GRAYLOG_EXTRA_VALUE": "{{.GraylogExtraValue}}",
}

func mergeModelEnvsWithDefaultEnvs(m *workerModel) {
	if m.Type != sdk.Docker {
		return
	}

	if m.ModelDocker.Envs == nil {
		m.ModelDocker.Envs = make(map[string]string)
	}
	for envName := range defaultEnvs {
		if _, ok := m.ModelDocker.Envs[envName]; !ok {
			m.ModelDocker.Envs[envName] = defaultEnvs[envName]
		}
	}
}

const registryPasswordSecretName = "secrets.registry_password"

// If a docker registry password is given as password input we want to save it as a secret.
// Also we will reset the input fields to prevent the stortage of the clear value.
// The password field will be set with a template pattern to allow an hatchery to interpolate its value.
func replaceDockerRegistryPassword(db gorp.SqlExecutor, dbmodel *workerModel) (bool, string, error) {
	// Not a docker model or not with a private registry, clean password data
	if dbmodel.Type != sdk.Docker || !dbmodel.ModelDocker.Private {
		dbmodel.ModelDocker.Registry = ""
		dbmodel.ModelDocker.Username = ""
		dbmodel.ModelDocker.Password = ""
		if dbmodel.ID > 0 {
			if err := DeleteSecretRegistryPasswordForModelID(db, dbmodel.ID); err != nil {
				return false, "", err
			}
		}
		return false, "", nil
	}

	// Password not changed
	if dbmodel.ModelDocker.Password == "{{."+registryPasswordSecretName+"}}" {
		return false, "", nil
	}

	clearPassword := dbmodel.ModelDocker.Password
	dbmodel.ModelDocker.Password = "{{." + registryPasswordSecretName + "}}"
	return true, clearPassword, nil
}

func storeDockerRegistryPassword(ctx context.Context, db gorpmapping.SqlExecutorWithTx, workerModelID int64, password string) error {
	s, err := LoadSecretByModelIDAndName(ctx, db, workerModelID, registryPasswordSecretName)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}
	if s == nil {
		s = &sdk.WorkerModelSecret{
			Name:          registryPasswordSecretName,
			WorkerModelID: workerModelID,
			Value:         password,
		}
		return InsertSecret(ctx, db, s)
	}

	s.Value = password

	return UpdateSecret(ctx, db, s)
}
