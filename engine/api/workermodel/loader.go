package workermodel

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc for worker model.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.Model) error

// LoadOptions provides all options to load worker models.
var LoadOptions = struct {
	Default          LoadOptionFunc
	WithGroup        LoadOptionFunc
	WithCapabilities LoadOptionFunc
}{
	Default:          loadDefault,
	WithGroup:        loadGroup,
	WithCapabilities: loadCapabilities,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, ms ...*sdk.Model) error {
	if err := loadGroup(ctx, db, ms...); err != nil {
		return err
	}
	return loadCapabilities(ctx, db, ms...)
}

func loadGroup(ctx context.Context, db gorp.SqlExecutor, ms ...*sdk.Model) error {
	gs, err := group.LoadAllByIDs(ctx, db, sdk.ModelsToGroupIDs(ms))
	if err != nil {
		return err
	}

	m := make(map[int64]sdk.Group, len(gs))
	for i := range gs {
		m[gs[i].ID] = gs[i]
	}

	for _, model := range ms {
		if g, ok := m[model.GroupID]; ok {
			model.Group = &g
			model.IsOfficial = model.GroupID == group.SharedInfraGroup.ID
		}
	}

	return nil
}

func loadCapabilities(ctx context.Context, db gorp.SqlExecutor, ms ...*sdk.Model) error {
	for i := range ms {
		rs, err := LoadCapabilitiesByModelID(ctx, db, ms[i].ID)
		if err != nil {
			return err
		}
		ms[i].RegisteredCapabilities = rs
	}
	return nil
}
