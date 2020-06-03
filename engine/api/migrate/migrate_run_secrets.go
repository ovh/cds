package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var currentProjectID int64
var currentProjectSecrets []sdk.WorkflowRunSecret

func MigrateRunsSecrets(ctx context.Context, db *gorp.DbMap) error {
	query := `
		SELECT workflow_run.id FROM workflow_run 
		LEFT JOIN workflow_run_secret ON workflow_run_secret.workflow_run_id = workflow_run.id
		WHERE read_only = false AND workflow_run_secret.id IS NULL
	`
	rows, err := db.Query(query)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close() // nolint
			return sdk.WithStack(err)
		}
		ids = append(ids, id)
	}

	if err := rows.Close(); err != nil {
		return sdk.WithStack(err)
	}

	var mError = new(sdk.MultiError)
	for _, id := range ids {
		if err := migrateRunSecret(ctx, db, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.MigrateRunsSecrets> unable to migrate run %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func migrateRunSecret(ctx context.Context, db *gorp.DbMap, id int64) error {
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
			log.Info(ctx, "migrateRunSecret: run already locked")
			return nil
		}
		return sdk.WithStack(err)
	}

	secrets, err := workflow.RetrieveSecrets(tx, run.Workflow)
	if err != nil {
		return sdk.WithStack(err)
	}

	// Retrieve project secrets only once
	if currentProjectID != run.ProjectID {
		// Reinit cache
		currentProjectSecrets = make([]sdk.WorkflowRunSecret, 0)
		currentProjectID = run.ProjectID

		// Get new secrets
		proj, err := project.LoadByID(tx, run.ProjectID, project.LoadOptions.WithVariablesWithClearPassword, project.LoadOptions.WithClearKeys)
		if err != nil {
			return err
		}
		// Create a snapshot of project secrets and keys
		pv := sdk.VariablesFilter(proj.Variables, sdk.SecretVariable, sdk.KeyVariable)
		pv = sdk.VariablesPrefix(pv, "cds.proj.")
		for _, v := range pv {
			currentProjectSecrets = append(currentProjectSecrets, sdk.WorkflowRunSecret{
				WorkflowRunID: run.ID,
				Context:       workflow.SecretProjContext,
				Name:          v.Name,
				Value:         []byte(v.Value),
			})
		}
		for _, k := range proj.Keys {
			currentProjectSecrets = append(currentProjectSecrets, sdk.WorkflowRunSecret{
				WorkflowRunID: run.ID,
				Context:       workflow.SecretProjContext,
				Name:          fmt.Sprintf("cds.key.%s.priv", k.Name),
				Value:         []byte(k.Private),
			})
		}
	} else {
		// Update run id
		for i := range currentProjectSecrets {
			currentProjectSecrets[i].WorkflowRunID = run.ID
		}
	}

	for _, s := range currentProjectSecrets {
		if err := workflow.InsertRunSecret(ctx, tx, &s); err != nil {
			return err
		}
	}

	// Find Needed Project Integrations
	ppIDs := make(map[int64]string, 0)
	for _, n := range run.Workflow.WorkflowData.Array() {
		if n.Context == nil || n.Context.ProjectIntegrationID == 0 {
			continue
		}
		ppIDs[n.Context.ProjectIntegrationID] = ""
	}
	for ppID := range ppIDs {
		projectIntegration, err := integration.LoadProjectIntegrationByIDWithClearPassword(tx, ppID)
		if err != nil {
			return err
		}
		ppIDs[ppID] = projectIntegration.Name

		// Project integration secret variable
		for k, v := range projectIntegration.Config {
			if v.Type != sdk.SecretVariable {
				continue
			}
			wrSecret := sdk.WorkflowRunSecret{
				WorkflowRunID: run.ID,
				Context:       fmt.Sprintf(workflow.SecretProjIntegrationContext, ppID),
				Name:          fmt.Sprintf("cds.integration.%s", k),
				Value:         []byte(v.Value),
			}
			if err := workflow.InsertRunSecret(ctx, tx, &wrSecret); err != nil {
				return err
			}

		}
	}

	// Application secret
	for id, variables := range secrets.ApplicationsSecrets {
		// Filter to avoid getting cds.deployment variables
		for _, v := range variables {
			var wrSecret sdk.WorkflowRunSecret
			switch {
			case strings.HasPrefix(v.Name, "cds.app.") || strings.HasPrefix(v.Name, "cds.key."):
				wrSecret = sdk.WorkflowRunSecret{
					WorkflowRunID: run.ID,
					Context:       fmt.Sprintf(workflow.SecretAppContext, id),
					Name:          v.Name,
					Value:         []byte(v.Value),
				}
			case strings.Contains(v.Name, ":cds.integration."):
				piName := strings.SplitN(v.Name, ":", 2)
				wrSecret = sdk.WorkflowRunSecret{
					WorkflowRunID: run.ID,
					Context:       fmt.Sprintf(workflow.SecretApplicationIntegrationContext, id, piName[0]),
					Name:          piName[1],
					Value:         []byte(v.Value),
				}
			default:
				continue
			}
			if err := workflow.InsertRunSecret(ctx, tx, &wrSecret); err != nil {
				return err
			}
		}
	}

	// Environment secret
	for id, variables := range secrets.EnvironmentdSecrets {
		for _, v := range variables {
			wrSecret := sdk.WorkflowRunSecret{
				WorkflowRunID: run.ID,
				Context:       fmt.Sprintf(workflow.SecretEnvContext, id),
				Name:          v.Name,
				Value:         []byte(v.Value),
			}
			if err := workflow.InsertRunSecret(ctx, tx, &wrSecret); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
