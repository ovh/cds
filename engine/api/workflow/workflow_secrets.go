package workflow

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/sdk"
)

func RetrieveSecrets(ctx context.Context, db gorp.SqlExecutor, wf sdk.Workflow) (*PushSecrets, error) {
	secrets := &PushSecrets{
		ApplicationsSecrets: make(map[int64][]sdk.Variable),
		EnvironmentdSecrets: make(map[int64][]sdk.Variable),
	}

	for _, app := range wf.Applications {
		appSecrets, err := LoadApplicationSecrets(ctx, db, app.ID)
		if err != nil {
			return nil, err
		}
		secrets.ApplicationsSecrets[app.ID] = appSecrets
	}

	for _, env := range wf.Environments {
		envSecrets, err := LoadEnvironmentSecrets(db, env.ID)
		if err != nil {
			return nil, err
		}
		secrets.EnvironmentdSecrets[env.ID] = envSecrets
	}
	return secrets, nil
}

func LoadApplicationSecrets(ctx context.Context, db gorp.SqlExecutor, id int64) ([]sdk.Variable, error) {
	appDB, err := application.LoadByIDWithClearVCSStrategyPassword(ctx, db, id,
		application.LoadOptions.WithVariablesWithClearPassword,
		application.LoadOptions.WithClearDeploymentStrategies,
		application.LoadOptions.WithClearKeys)
	if err != nil {
		return nil, err
	}

	secretsVariables := make([]sdk.Variable, 0)

	vars := sdk.VariablesFilter(sdk.FromApplicationVariables(appDB.Variables), sdk.SecretVariable, sdk.KeyVariable)
	for _, v := range vars {
		secretsVariables = append(secretsVariables, sdk.Variable{
			Name:  fmt.Sprintf("cds.app.%s", v.Name),
			Type:  v.Type,
			Value: v.Value,
		})
	}

	for _, k := range appDB.Keys {
		secretsVariables = append(secretsVariables, sdk.Variable{
			Name:  fmt.Sprintf("cds.key.%s.priv", k.Name),
			Type:  string(k.Type),
			Value: k.Private,
		})
	}

	for name, appD := range appDB.DeploymentStrategies {
		for vName, v := range appD {
			if v.Type != sdk.IntegrationConfigTypePassword {
				continue
			}
			secretsVariables = append(secretsVariables, sdk.Variable{
				Name:  fmt.Sprintf("%s:cds.integration.%s.%s", name, sdk.IntegrationVariablePrefixDeployment, vName),
				Type:  sdk.SecretVariable,
				Value: v.Value,
			})
		}
	}
	secretsVariables = append(secretsVariables, sdk.Variable{
		Name:  "git.http.password",
		Type:  sdk.SecretVariable,
		Value: appDB.RepositoryStrategy.Password,
	})
	return secretsVariables, nil
}

func LoadEnvironmentSecrets(db gorp.SqlExecutor, id int64) ([]sdk.Variable, error) {
	secretsVariables := make([]sdk.Variable, 0)
	envVars, err := environment.LoadAllVariablesWithDecryption(db, id)
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

	keys, err := environment.LoadAllKeysWithPrivateContent(db, id)
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
	return secretsVariables, nil
}
