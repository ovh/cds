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
