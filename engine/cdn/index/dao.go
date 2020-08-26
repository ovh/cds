package index

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getItems(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query, opts ...gorpmapper.GetOptionFunc) ([]Item, error) {
	var res []Item
	if err := m.GetAll(ctx, db, q, &res, opts...); err != nil {
		return nil, err
	}

	var verifiedItems []Item
	for _, i := range res {
		isValid, err := m.CheckSignature(i, i.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "index.get> index %s data corrupted", i.ID)
			continue
		}
		verifiedItems = append(verifiedItems, i)
	}

	return verifiedItems, nil
}

func getItem(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query, opts ...gorpmapper.GetOptionFunc) (*Item, error) {
	var i Item
	found, err := m.Get(ctx, db, q, &i, opts...)
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

func LoadAllItems(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, size int, opts ...gorpmapper.GetOptionFunc) ([]Item, error) {
	query := gorpmapper.NewQuery("SELECT * FROM index ORDER BY created LIMIT $1").Args(size)
	return getItems(ctx, m, db, query, opts...)
}

// LoadItemByID returns an item from database for given id.
func LoadItemByID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string, opts ...gorpmapper.GetOptionFunc) (*Item, error) {
	query := gorpmapper.NewQuery("SELECT * FROM index WHERE id = $1").Args(id)
	return getItem(ctx, m, db, query, opts...)
}

// LoadItemByIDs returns items from database for given ids.
func LoadItemByIDs(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, ids []string, opts ...gorpmapper.GetOptionFunc) ([]Item, error) {
	query := gorpmapper.NewQuery("SELECT * FROM index WHERE id = ANY($1)").Args(pq.StringArray(ids))
	return getItems(ctx, m, db, query, opts...)
}

// LoadAndLockItemByID returns an item from database for given id.
func LoadAndLockItemByID(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, id string, opts ...gorpmapper.GetOptionFunc) (*Item, error) {
	query := gorpmapper.NewQuery("SELECT * FROM index WHERE id = $1 FOR UPDATE SKIP LOCKED").Args(id)
	return getItem(ctx, m, db, query, opts...)
}

// InsertItem in database.
func InsertItem(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *Item) error {
	i.ID = sdk.UUID()
	i.Created = time.Now()
	i.LastModified = time.Now()
	if err := m.InsertAndSign(ctx, db, i); err != nil {
		return sdk.WrapError(err, "unable to insert index item %s", i.ID)
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

func DeleteItem(m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *Item) error {
	if err := m.Delete(db, i); err != nil {
		return sdk.WrapError(err, "unable to delete item %s", i.ID)
	}
	return nil
}

// LoadItemByAPIRefHashAndType load an item by his job id, step order and type
func LoadItemByAPIRefHashAndType(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, hash string, typ string, opts ...gorpmapper.GetOptionFunc) (*Item, error) {
	query := gorpmapper.NewQuery(`
		SELECT *
		FROM index
		WHERE
			api_ref_hash = $1 AND
			type = $2
	`).Args(hash, typ)
	return getItem(ctx, m, db, query)
}
