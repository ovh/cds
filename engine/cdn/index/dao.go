package index

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getItem(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query) (*Item, error) {
	var i Item

	found, err := m.Get(ctx, db, q, &i)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumer")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := m.CheckSignature(i, i.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "index.get> index %s data corrupted", i.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &i, nil
}

// LoadItemByID returns an item from database for given id.
func LoadItemByID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string) (*Item, error) {
	query := gorpmapper.NewQuery("SELECT * FROM index WHERE id = $1").Args(id)
	return getItem(ctx, m, db, query)
}

// LoadAndLockItemByID returns an item from database for given id.
func LoadAndLockItemByID(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, id string) (*Item, error) {
	query := gorpmapper.NewQuery("SELECT * FROM index WHERE id = $1 FOR UPDATE SKIP LOCKED").Args(id)
	return getItem(ctx, m, db, query)
}

// InsertItem in database.
func InsertItem(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *Item) error {
	i.ID = sdk.UUID()
	i.Created = time.Now()
	i.LastModified = time.Now()
	if err := m.InsertAndSign(ctx, db, i); err != nil {
		return sdk.WrapError(err, "unable to insert index item")
	}
	return nil
}

// UpdateItem in database
func UpdateItem(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *Item) error {
	i.LastModified = time.Now()
	if err := m.UpdateAndSign(ctx, db, i); err != nil {
		return sdk.WrapError(err, "unable to update index item")
	}
	return nil
}

// LoadItemByJobStepAndType load an item by his job id, step order and type
func LoadItemByApiRefHashAndType(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, hash string, typ string) (*Item, error) {
	query := gorpmapper.NewQuery(`
		SELECT * 
		FROM index 
		WHERE 
			api_ref_hash = $1 AND
			type = $2
	`).Args(hash, typ)
	return getItem(ctx, m, db, query)
}
