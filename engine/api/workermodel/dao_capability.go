package workermodel

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

// LoadCapabilitiesByModelID retrieves capabilities of given worker model.
func LoadCapabilitiesByModelID(ctx context.Context, db gorp.SqlExecutor, workerModelID int64) (sdk.RequirementList, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_capability
    WHERE worker_model_id = $1
    ORDER BY name
  `).Args(workerModelID)

	cs := []workerModelCapability{}
	if err := gorpmapping.GetAll(ctx, db, query, &cs); err != nil {
		return nil, sdk.WrapError(err, "cannot get capabilities for worker model %d", workerModelID)
	}

	rs := make(sdk.RequirementList, 0, len(cs))
	for i := range cs {
		rs = append(rs, sdk.Requirement{
			Name:  cs[i].Name,
			Type:  cs[i].Type,
			Value: cs[i].Argument,
		})
	}
	return rs, nil
}

// DeleteCapabilitiesByModelID removes all capabilities from database for given worker model id.
func DeleteCapabilitiesByModelID(db gorp.SqlExecutor, workerModelID int64) error {
	_, err := db.Exec("DELETE FROM worker_capability WHERE worker_model_id = $1", workerModelID)
	return sdk.WrapError(err, "unable to remove worker capabilities for model with id %d", workerModelID)
}

// InsertCapabilityForModelID inserts given capability in database.
func InsertCapabilityForModelID(db gorp.SqlExecutor, workerModelID int64, r *sdk.Requirement) error {
	c := workerModelCapability{
		WorkerModelID: workerModelID,
		Type:          r.Type,
		Name:          r.Name,
		Argument:      r.Value,
	}
	return sdk.WithStack(db.Insert(&c))
}
