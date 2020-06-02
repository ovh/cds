package workflow

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func InsertRunSecret(ctx context.Context, db gorp.SqlExecutor, wrSecret *sdk.WorkflowRunSecret) error {
	dbData := &dbWorkflowRunSecret{WorkflowRunSecret: *wrSecret}
	dbData.ID = sdk.UUID()
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*wrSecret = dbData.WorkflowRunSecret
	return nil
}

func loadRunSecretWithDecryption(ctx context.Context, db gorp.SqlExecutor, runID int64, context string) ([]sdk.Variable, error) {
	var dbSecrets []dbWorkflowRunSecret
	query := gorpmapping.NewQuery(`SELECT * FROM workflow_run_secret WHERE workflow_run_id = $1 AND context = $2`).Args(runID, context)
	if err := gorpmapping.GetAll(ctx, db, query, &dbSecrets, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}
	secrets := make([]sdk.Variable, len(dbSecrets))
	for i := range dbSecrets {
		isValid, err := gorpmapping.CheckSignature(dbSecrets[i], dbSecrets[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "workflow.loadRunSecretWithDecryption> secret value corrupted", dbSecrets[i].ID)
			continue
		}
		secrets[i] = sdk.Variable{
			Name:  dbSecrets[i].Name,
			Value: string(dbSecrets[i].Value),
		}
	}
	return secrets, nil
}
