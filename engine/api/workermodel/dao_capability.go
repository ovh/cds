package workermodel

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadCapabilities retrieves capabilities of given worker model.
func LoadCapabilities(ctx context.Context, db gorp.SqlExecutor, workerModelID int64) (sdk.RequirementList, error) {
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
