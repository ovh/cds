package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func RunsSecrets(ctx context.Context, dbFunc func() *gorp.DbMap) error {
	log.Info(ctx, "migrate.MigrateRunsSecrets: get runs to migrate")
	for {
		wrMigrated, err := migrateRuns(ctx, dbFunc)
		if err != nil {
			log.Error(ctx, "migrate.MigrateRunsSecrets: Error during migration: %v", err)
			return err
		}
		if wrMigrated == 0 {
			break
		}
	}
	return nil
}

func migrateRuns(ctx context.Context, dbFunc func() *gorp.DbMap) (int, error) {
	dbPrepare := dbFunc()
	wrIds, projIds, appIds, envIds, err := getRunsAndDeps(dbPrepare)
	if err != nil {
		return 0, err
	}

	if len(wrIds) == 0 {
		return 0, nil
	}

	log.Info(ctx, "migrate.MigrateRunsSecrets: Start Prepare migration")
	projVars, err := project.LoadAllVariablesForProjectsWithDecryption(ctx, dbPrepare, projIds)
	if err != nil {
		return 0, err
	}

	projKeys, err := project.LoadAllKeysForProjectsWithDecryption(ctx, dbPrepare, projIds)
	if err != nil {
		return 0, err
	}

	projIntsSlice, err := integration.LoadAllIntegrationsForProjectsWithDecryption(ctx, dbPrepare, projIds)
	if err != nil {
		return 0, err
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

	appVars, err := application.LoadAllVariablesForAppsWithDecryption(ctx, dbPrepare, appIds)
	if err != nil {
		return 0, err
	}

	appKeys, err := application.LoadAllKeysForAppsWithDecryption(ctx, dbPrepare, appIds)
	if err != nil {
		return 0, err
	}

	appDeployments, err := application.LoadAllDeploymnentForAppsWithDecryption(ctx, dbPrepare, appIds)
	if err != nil {
		return 0, err
	}

	appVCS, err := application.LoadAllByIDsWithDecryption(dbPrepare, appIds)
	if err != nil {
		return 0, err
	}
	appStrats := make(map[int64]sdk.RepositoryStrategy, len(appVCS))
	for _, appVCS := range appVCS {
		appStrats[appVCS.ID] = appVCS.RepositoryStrategy
	}
	appVCS = make([]sdk.Application, 0)

	envsVars, err := environment.LoadAllVariablesForEnvsWithDecryption(ctx, dbPrepare, envIds)
	if err != nil {
		return 0, err
	}

	envsKeys, err := environment.LoadAllKeysForEnvsWithDecryption(ctx, dbPrepare, envIds)
	if err != nil {
		return 0, err
	}

	log.Info(ctx, "migrate.MigrateRunsSecrets: start migration loop")

	jobs := make(chan int64, len(wrIds))
	results := make(chan int64, len(wrIds))
	for w := 1; w <= 3; w++ {
		go workerMigrate(ctx, dbFunc(), jobs, results, projVars, projKeys, projInts, appVars, appKeys, appStrats, appDeployments, envsVars, envsKeys)
	}
	for _, id := range wrIds {
		jobs <- id
	}
	close(jobs)

	for a := 0; a < len(wrIds); a++ {
		<-results
	}
	return len(wrIds), nil
}

func workerMigrate(ctx context.Context, db *gorp.DbMap, jobs <-chan int64, results chan<- int64, projVarsMap map[int64][]sdk.ProjectVariable, projKeysMap map[int64][]sdk.ProjectKey, projIntsMap map[int64]map[int64]sdk.ProjectIntegration, appVarsMap map[int64][]sdk.ApplicationVariable, appKeysMap map[int64][]sdk.ApplicationKey, appVCSMaps map[int64]sdk.RepositoryStrategy, appDeploysMaps map[int64]map[int64]sdk.IntegrationConfig, envVarsMap map[int64][]sdk.EnvironmentVariable, envKeysMap map[int64][]sdk.EnvironmentKey) {
	for j := range jobs {
		if err := migrate(ctx, db, j, projVarsMap, projKeysMap, projIntsMap, appVarsMap, appKeysMap, appVCSMaps, appDeploysMaps, envVarsMap, envKeysMap); err != nil {
			log.Error(ctx, "migrate.MigrateRunsSecrets: unable to migrate run %d: %v", j, err)
		}
		results <- j
	}
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
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			log.Info(ctx, "migrate.MigrateRunsSecrets: workflow run locked")
			return nil
		}
		return err
	}

	// Check if workflow has been migrated
	s, err := CountSecret(tx, id)
	if err != nil {
		return err
	}
	if s > 0 {
		log.Info(ctx, "migrate.MigrateRunsSecrets: workflow run already migrated")
		return nil
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
				Name:          fmt.Sprintf("cds.proj.%s", v.Name),
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
					if v.Type != sdk.IntegrationConfigTypePassword {
						continue
					}
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
		if err := workflow.InsertRunSecret(ctx, tx, &s); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	log.Info(ctx, "migrate.MigrateRunsSecrets: workflow run migrated")
	return nil
}

func getRunsAndDeps(db gorp.SqlExecutor) ([]int64, []int64, []int64, []int64, error) {
	// Get all wruns to migrate
	wrs, err := LoadLastRunsByDate(db)
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

//WorkflowRun is an execution instance of a run
type WorkflowRun struct {
	ID        int64        `json:"id" db:"id"`
	ProjectID int64        `json:"project_id,omitempty" db:"project_id"`
	Workflow  sdk.Workflow `json:"workflow" db:"workflow"`
}

// LoadLastRuns returns the last run per last_mdodified
func LoadLastRunsByDate(db gorp.SqlExecutor) ([]sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select workflow_run.id, workflow_run.project_id, workflow_run.workflow
	from workflow_run
	left join workflow_run_secret on workflow_run_secret.workflow_run_id = workflow_run.id
	where workflow_run.read_only = false AND workflow_run_secret.id IS NULL
	order by workflow_run.id desc LIMIT 10000`)
	return loadRuns(db, query)
}

func CountSecret(db gorp.SqlExecutor, id int64) (int64, error) {
	query := `select count(*) from workflow_run_secret where workflow_run_id = $1`
	nb, err := db.SelectInt(query, id)
	if err != nil {
		return 0, err
	}
	return nb, nil
}

func loadRuns(db gorp.SqlExecutor, query string) ([]sdk.WorkflowRun, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wrs := make([]sdk.WorkflowRun, 0)
	for rows.Next() {
		wr := sdk.WorkflowRun{}
		var ww sql.NullString
		if err := rows.Scan(&wr.ID, &wr.ProjectID, &ww); err != nil {
			return nil, err
		}
		if ww.Valid {
			if err := sdk.JSONUnmarshal([]byte(ww.String), &wr.Workflow); err != nil {
				return nil, err
			}
		}
		wrs = append(wrs, wr)
	}
	return wrs, nil
}
