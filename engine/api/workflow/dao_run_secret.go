package workflow

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

const (
	SecretProjContext                   = "proj"
	SecretAppContext                    = "app:%d"
	SecretEnvContext                    = "env:%d"
	SecretProjIntegrationContext        = "integration:%d"
	SecretApplicationIntegrationContext = "app:%d:integration:%s"
)

func InsertRunSecret(ctx context.Context, db gorpmapper.SqlExecutorWithTx, wrSecret *sdk.WorkflowRunSecret) error {
	dbData := &dbWorkflowRunSecret{WorkflowRunSecret: *wrSecret}
	dbData.ID = sdk.UUID()
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*wrSecret = dbData.WorkflowRunSecret
	return nil
}

func loadRunSecretWithDecryption(ctx context.Context, db gorp.SqlExecutor, runID int64, entities []string) (sdk.WorkflowRunSecrets, error) {
	var dbSecrets []dbWorkflowRunSecret
	query := gorpmapping.NewQuery(`SELECT * FROM workflow_run_secret WHERE workflow_run_id = $1 AND context = ANY(string_to_array($2, ',')::text[])`).Args(runID, gorpmapping.IDStringsToQueryString(entities))
	if err := gorpmapping.GetAll(ctx, db, query, &dbSecrets, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}
	secrets := make(sdk.WorkflowRunSecrets, 0, len(dbSecrets))
	for i := range dbSecrets {
		isValid, err := gorpmapping.CheckSignature(dbSecrets[i], dbSecrets[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "secret value corrupted %s", dbSecrets[i].ID)
			continue
		}
		secrets = append(secrets, dbSecrets[i].WorkflowRunSecret)
	}
	return secrets, nil
}

func CountRunSecretsByWorkflowRunID(ctx context.Context, db gorp.SqlExecutor, workflowRunID int64) (int64, error) {
	query := `SELECT COUNT(1) FROM workflow_run_secret WHERE workflow_run_id = $1`
	count, err := db.SelectInt(query, workflowRunID)
	if err != nil {
		return 0, sdk.WrapError(err, "unable to count workflow run secret for workflow run id %d", workflowRunID)
	}
	return count, nil
}

func DeleteRunSecretsByWorkflowRunID(ctx context.Context, db gorpmapper.SqlExecutorWithTx, workflowRunID int64) error {
	query := `DELETE FROM workflow_run_secret WHERE workflow_run_id = $1`
	if _, err := db.Exec(query, workflowRunID); err != nil {
		return sdk.WrapError(err, "unable to delete workflow run secret for workflow run id %d", workflowRunID)
	}
	return nil
}
