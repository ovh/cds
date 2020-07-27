package featureflipping

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func LoadAll(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor) ([]sdk.Feature, error) {
	query := gorpmapper.NewQuery("SELECT * FROM feature_flipping")
	var result []sdk.Feature
	if err := m.GetAll(ctx, db, query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func LoadByName(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, name string) (sdk.Feature, error) {
	query := gorpmapper.NewQuery("SELECT * FROM feature_flipping WHERE name = $1").Args(name).Limit(1)
	var f sdk.Feature
	found, err := m.Get(ctx, db, query, &f)
	if err != nil {
		return sdk.Feature{}, err
	}
	if !found {
		return sdk.Feature{}, sdk.WithStack(sdk.ErrNotFound)
	}
	return f, nil
}

func Insert(m *gorpmapper.Mapper, db gorp.SqlExecutor, f *sdk.Feature) error {
	return m.Insert(db, f)
}

func Update(m *gorpmapper.Mapper, db gorp.SqlExecutor, f *sdk.Feature) error {
	return m.Update(db, f)
}

func Delete(db gorp.SqlExecutor, id int64) error {
	_, err := db.Exec("DELETE FROM feature_flipping WHERE id = $1", id)
	if err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
