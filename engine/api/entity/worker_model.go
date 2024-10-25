package entity

import (
	"context"
	"encoding/json"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/sdk"
)

func WorkerModelDecryptSecrets(ctx context.Context, db gorp.SqlExecutor, projectID int64, wm *sdk.V2WorkerModel, decryptFunc keys.DecryptFunc) error {
	if wm.Type == sdk.WorkerModelTypeOpenstack {
		return nil
	}
	switch wm.Type {
	case sdk.WorkerModelTypeDocker:
		var spec sdk.V2WorkerModelDockerSpec
		if err := json.Unmarshal(wm.Spec, &spec); err != nil {
			return sdk.WithStack(err)
		}
		if spec.Password == "" {
			return nil
		}
		secret, err := decryptFunc(ctx, db, projectID, spec.Password)
		if err != nil {
			return err
		}
		spec.Password = secret
		wm.Spec, _ = json.Marshal(spec)
	case sdk.WorkerModelTypeVSphere:
		var spec sdk.V2WorkerModelVSphereSpec
		if err := json.Unmarshal(wm.Spec, &spec); err != nil {
			return sdk.WithStack(err)
		}
		if spec.Password == "" {
			return nil
		}
		secret, err := decryptFunc(ctx, db, projectID, spec.Password)
		if err != nil {
			return err
		}
		spec.Password = secret
		wm.Spec, _ = json.Marshal(spec)
	}
	return nil
}
