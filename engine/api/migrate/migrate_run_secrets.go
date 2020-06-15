package migrate

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func RunsSecrets(ctx context.Context, db *gorp.DbMap) error {
	log.Info(ctx, "migrate.MigrateRunsSecrets: start Prepare migration")
	wrIds, projIds, appIds, envIds, err := getRunsAndDeps(db, 3)

	projVars, err := project.LoadAllVariablesForProjectsWithDecryption(ctx, db, projIds)
	if err != nil {
		return err
	}
	defer func() {
		projVars = make(map[int64][]sdk.ProjectVariable, 0)
	}()

	projKeys, err := project.LoadAllKeysForProjectsWithDecryption(ctx, db, projIds)
	if err != nil {
		return err
	}
	defer func() {
		projKeys = make(map[int64][]sdk.ProjectKey, 0)
	}()

	projIntsSlice, err := integration.LoadAllIntegrationsForProjectsWithDecryption(ctx, db, projIds)
	if err != nil {
		return err
	}
	projInts := make(map[int64]map[int64]sdk.ProjectIntegration, len(projIntsSlice))
	for id, v := range projIntsSlice {
		mIntegrations := make(map[int64]sdk.ProjectIntegration, len(v))
		for i := range v {
			mIntegrations[v[i].ID] = v[i]
		}
		projInts[id] = mIntegrations
	}
	projIntsSlice = make(map[int64][]sdk.ProjectIntegration, 0)
	defer func() {
		projInts = make(map[int64]map[int64]sdk.ProjectIntegration, 0)
	}()

	appVars, err := application.LoadAllVariablesForAppsWithDecryption(ctx, db, appIds)
	if err != nil {
		return err
	}
	defer func() {
		appVars = make(map[int64][]sdk.ApplicationVariable, 0)
	}()

	appKeys, err := application.LoadAllKeysForAppsWithDecryption(ctx, db, appIds)
	if err != nil {
		return err
	}
	defer func() {
		appKeys = make(map[int64][]sdk.ApplicationKey, 0)
	}()

	appDeployments, err := application.LoadAllDeploymnentForAppsWithDecryption(ctx, db, appIds)
	if err != nil {
		return err
	}
	defer func() {
		appDeployments = make(map[int64]map[int64]sdk.IntegrationConfig, 0)
	}()

	appVCS, err := application.LoadAllByIDsWithDecryption(db, appIds)
	if err != nil {
		return err
	}
	appStrats := make(map[int64]sdk.RepositoryStrategy, len(appVCS))
	for _, appVCS := range appVCS {
		appStrats[appVCS.ID] = appVCS.RepositoryStrategy
	}
	appVCS = make([]sdk.Application, 0)
	defer func() {
		appStrats = make(map[int64]sdk.RepositoryStrategy, 0)
	}()

	envsVars, err := environment.LoadAllVariablesForEnvsWithDecryption(ctx, db, envIds)
	if err != nil {
		return err
	}
	defer func() {
		envsVars = make(map[int64][]sdk.EnvironmentVariable, 0)
	}()

	envsKeys, err := environment.LoadAllKeysForEnvsWithDecryption(ctx, db, envIds)
	if err != nil {
		return err
	}
	defer func() {
		envsKeys = make(map[int64][]sdk.EnvironmentKey, 0)
	}()

	log.Info(ctx, "migrate.MigrateRunsSecrets: start migration")
	for _, id := range wrIds {
		if err := migrate(ctx, db, id, projVars, projKeys, projInts, appVars, appKeys, appStrats, appDeployments, envsVars, envsKeys); err != nil {
			log.Error(ctx, "unable to migrate run %d: %v", id, err)
		}
	}
	return nil
}

func migrate(ctx context.Context, db *gorp.DbMap, id int64, projVarsMap map[int64][]sdk.ProjectVariable, projKeysMap map[int64][]sdk.ProjectKey, projIntsMap map[int64]map[int64]sdk.ProjectIntegration, appVarsMap map[int64][]sdk.ApplicationVariable, appKeysMap map[int64][]sdk.ApplicationKey, appVCSMaps map[int64]sdk.RepositoryStrategy, appDeploysMaps map[int64]map[int64]sdk.IntegrationConfig, envVarsMap map[int64][]sdk.EnvironmentVariable, envKeysMap map[int64][]sdk.EnvironmentKey) interface{} {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() //nolint

	run, err := workflow.LoadAndLockRunByID(tx, id, workflow.LoadRunOptions{
		DisableDetailledNodeRun: true,
	})
	if err != nil {
		return err
	}

	secrets := make([]sdk.WorkflowRunSecret, 0)

	// Add Project var secrets
	if projVars, ok := projVarsMap[run.ProjectID]; ok {
		for _, v := range projVars {
			if !sdk.NeedPlaceholder(v.Type) {
				continue
			}
			secrets = append(secrets, sdk.WorkflowRunSecret{
				WorkflowRunID: run.ID,
				Context:       workflow.SecretProjContext,
				Name:          fmt.Sprintf("cds.proj.%s.", v.Name),
				Type:          v.Type,
				Value:         []byte(v.Value),
			})
		}
	}
	// Add Project key secrets
	if projKeys, ok := projKeysMap[run.ProjectID]; ok {
		for _, v := range projKeys {
			secrets = append(secrets, sdk.WorkflowRunSecret{
				WorkflowRunID: run.ID,
				Context:       workflow.SecretProjContext,
				Name:          fmt.Sprintf("cds.key.%s.priv", v.Name),
				Type:          string(v.Type),
				Value:         []byte(v.Private),
			})
		}
	}

	// Check Integration on workflow
	integrationUsed := make(map[int64]string, 0)
	for id, v := range run.Workflow.ProjectIntegrations {
		integrationUsed[id] = v.Name
	}

	// get secrets for all used integration
	integrationsInProject, ok := projIntsMap[run.ProjectID]
	if ok {
		for k := range run.Workflow.ProjectIntegrations {
			intInProj, ok := integrationsInProject[k]
			if !ok {
				delete(integrationUsed, k)
				continue
			}
			integrationUsed[k] = intInProj.Name
			for k, v := range intInProj.Config {
				if v.Type != sdk.SecretVariable {
					continue
				}
				secrets = append(secrets, sdk.WorkflowRunSecret{
					WorkflowRunID: run.ID,
					Context:       fmt.Sprintf(workflow.SecretProjIntegrationContext, intInProj.ID),
					Name:          fmt.Sprintf("cds.integration.%s", k),
					Type:          v.Type,
					Value:         []byte(v.Value),
				})
			}
			break
		}
	}

	// Application secret
	for id := range run.Workflow.Applications {
		if appVars, ok := appVarsMap[id]; ok {
			for _, v := range appVars {
				if !sdk.NeedPlaceholder(v.Type) {
					continue
				}
				secrets = append(secrets, sdk.WorkflowRunSecret{
					WorkflowRunID: run.ID,
					Context:       fmt.Sprintf(workflow.SecretAppContext, id),
					Name:          fmt.Sprintf("cds.app.%s", v.Name),
					Type:          v.Type,
					Value:         []byte(v.Value),
				})
			}
		}
		if appKeys, ok := appKeysMap[id]; ok {
			for _, k := range appKeys {
				secrets = append(secrets, sdk.WorkflowRunSecret{
					WorkflowRunID: run.ID,
					Context:       fmt.Sprintf(workflow.SecretAppContext, id),
					Name:          fmt.Sprintf("cds.key.%s.priv", k.Name),
					Type:          string(k.Type),
					Value:         []byte(k.Private),
				})
			}
		}
		if appStrat, ok := appVCSMaps[id]; ok {
			secrets = append(secrets, sdk.WorkflowRunSecret{
				WorkflowRunID: run.ID,
				Context:       fmt.Sprintf(workflow.SecretAppContext, id),
				Name:          "git.http.password",
				Type:          "string",
				Value:         []byte(appStrat.Password),
			})
		}

		if appDeployments, ok := appDeploysMaps[id]; ok {
			for depID, depValue := range appDeployments {
				for vName, v := range depValue {
					secrets = append(secrets, sdk.WorkflowRunSecret{
						WorkflowRunID: run.ID,
						Context:       fmt.Sprintf(workflow.SecretApplicationIntegrationContext, id, integrationsInProject[depID].Name),
						Name:          fmt.Sprintf("cds.integration.%s", vName),
						Type:          v.Type,
						Value:         []byte(v.Value),
					})
				}
			}
		}
	}

	// Environment secret
	for id := range run.Workflow.Environments {
		if envVars, ok := envVarsMap[id]; ok {
			for _, v := range envVars {
				if !sdk.NeedPlaceholder(v.Type) {
					continue
				}
				secrets = append(secrets, sdk.WorkflowRunSecret{
					WorkflowRunID: run.ID,
					Context:       fmt.Sprintf(workflow.SecretEnvContext, id),
					Name:          fmt.Sprintf("cds.env.%s", v.Name),
					Type:          v.Type,
					Value:         []byte(v.Value),
				})
			}
		}
		if envKeys, ok := envKeysMap[id]; ok {
			for _, k := range envKeys {
				secrets = append(secrets, sdk.WorkflowRunSecret{
					WorkflowRunID: run.ID,
					Context:       fmt.Sprintf(workflow.SecretEnvContext, id),
					Name:          fmt.Sprintf("cds.key.%s.priv", k.Name),
					Type:          string(k.Type),
					Value:         []byte(k.Private),
				})
			}
		}

	}

	for _, s := range secrets {
		if err := workflow.InsertRunSecret(ctx, db, &s); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func getRunsAndDeps(db gorp.SqlExecutor, months int) ([]int64, []int64, []int64, []int64, error) {
	// Get all wruns to migrate
	wrs, err := workflow.LoadLastRunsByDate(db, months)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// Retrieve all dep
	wrIds := make([]int64, 0, len(wrs))
	projs := make(map[int64]struct{})
	apps := make(map[int64]struct{})
	envs := make(map[int64]struct{})

	for _, wr := range wrs {
		wrIds = append(wrIds, wr.ID)
		projs[wr.ProjectID] = struct{}{}
		for appID := range wr.Workflow.Applications {
			apps[appID] = struct{}{}
		}
		for envID := range wr.Workflow.Environments {
			envs[envID] = struct{}{}
		}
	}
	return wrIds, sdk.IntMapToSlice(projs), sdk.IntMapToSlice(apps), sdk.IntMapToSlice(envs), nil
}
