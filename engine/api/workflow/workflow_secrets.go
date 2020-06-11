package workflow

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/sdk"
)

func RetrieveSecrets(db gorp.SqlExecutor, wf sdk.Workflow) (*PushSecrets, error) {
	secrets := &PushSecrets{
		ApplicationsSecrets: make(map[int64][]sdk.Variable),
		EnvironmentdSecrets: make(map[int64][]sdk.Variable),
	}

	for _, app := range wf.Applications {
		secretsVariables := make([]sdk.Variable, 0)
		appVars, err := application.LoadAllVariablesWithDecrytion(db, app.ID)
		if err != nil {
			return nil, err
		}
		vars := sdk.VariablesFilter(sdk.FromAplicationVariables(appVars), sdk.SecretVariable, sdk.KeyVariable)
		for _, v := range vars {
			secretsVariables = append(secretsVariables, sdk.Variable{
				Name:  fmt.Sprintf("cds.app.%s", v.Name),
				Type:  v.Type,
				Value: v.Value,
			})
		}

		keys, err := application.LoadAllKeysWithPrivateContent(db, app.ID)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			secretsVariables = append(secretsVariables, sdk.Variable{
				Name:  fmt.Sprintf("cds.key.%s.priv", k.Name),
				Type:  string(k.Type),
				Value: k.Private,
			})
		}

		appDeploymentStrats, err := application.LoadDeploymentStrategies(db, app.ID, true)
		if err != nil {
			return nil, err
		}
		for name, appD := range appDeploymentStrats {
			for vName, v := range appD {
				secretsVariables = append(secretsVariables, sdk.Variable{
					Name:  fmt.Sprintf("%s:cds.integration.%s", name, vName),
					Type:  sdk.SecretVariable,
					Value: v.Value,
				})
			}
		}

		secrets.ApplicationsSecrets[app.ID] = secretsVariables
	}

	for _, env := range wf.Environments {
		secretsVariables := make([]sdk.Variable, 0)
		envVars, err := environment.LoadAllVariablesWithDecrytion(db, env.ID)
		if err != nil {
			return nil, err
		}
		vars := sdk.VariablesFilter(sdk.FromEnvironmentVariables(envVars), sdk.SecretVariable, sdk.KeyVariable)
		for _, v := range vars {
			secretsVariables = append(secretsVariables, sdk.Variable{
				Name:  fmt.Sprintf("cds.env.%s", v.Name),
				Type:  v.Type,
				Value: v.Value,
			})
		}

		keys, err := environment.LoadAllKeysWithPrivateContent(db, env.ID)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			secretsVariables = append(secretsVariables, sdk.Variable{
				Name:  fmt.Sprintf("cds.key.%s.priv", k.Name),
				Type:  string(k.Type),
				Value: k.Private,
			})
		}
		secrets.EnvironmentdSecrets[env.ID] = secretsVariables
	}
	return secrets, nil
}
