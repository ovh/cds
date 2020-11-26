package item

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getItems(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItem, error) {
	var res []cdnItemDB
	if err := m.GetAll(ctx, db, q, &res, opts...); err != nil {
		return nil, err
	}

	var verifiedItems []sdk.CDNItem
	for _, i := range res {
		isValid, err := m.CheckSignature(i, i.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "item.get> item %s data corrupted", i.ID)
			continue
		}
		verifiedItems = append(verifiedItems, i.CDNItem)
	}

	return verifiedItems, nil
}

func getItem(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItem, error) {
	var i cdnItemDB
	found, err := m.Get(ctx, db, q, &i, opts...)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get item")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := m.CheckSignature(i, i.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "item.get> item %s data corrupted", i.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &i.CDNItem, nil
}

func LoadIDsToDelete(db gorp.SqlExecutor, size int) ([]string, error) {
	query := `
		SELECT id
		FROM item
		WHERE to_delete = true
		ORDER BY last_modified ASC
		LIMIT $1
	`
	var ids []string
	if _, err := db.Select(&ids, query, size); err != nil {
		return nil, sdk.WithStack(err)
	}
	return ids, nil
}

func DeleteByID(db gorp.SqlExecutor, ids ...string) error {
	query := `
		DELETE FROM item WHERE id = ANY($1)
	`
	_, err := db.Exec(query, pq.StringArray(ids))
	return sdk.WithStack(err)
}

func LoadAll(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, size int, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItem, error) {
	var query = gorpmapper.NewQuery("SELECT * FROM item WHERE to_delete = false ORDER BY created")
	if size > 0 {
		query = gorpmapper.NewQuery("SELECT * FROM item WHERE to_delete = false ORDER BY created LIMIT $1").Args(size)
	}
	return getItems(ctx, m, db, query, opts...)
}

// LoadByID returns an item from database for given id.
func LoadByID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItem, error) {
	query := gorpmapper.NewQuery("SELECT * FROM item WHERE id = $1").Args(id)
	return getItem(ctx, m, db, query, opts...)
}

// LoadByIDs returns items from database for given ids.
func LoadByIDs(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, ids []string, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItem, error) {
	query := gorpmapper.NewQuery("SELECT * FROM item WHERE id = ANY($1)").Args(pq.StringArray(ids))
	return getItems(ctx, m, db, query, opts...)
}

// LoadAndLockByID returns an item from database for given id.
func LoadAndLockByID(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, id string, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItem, error) {
	query := gorpmapper.NewQuery("SELECT * FROM item WHERE id = $1 FOR UPDATE SKIP LOCKED").Args(id)
	return getItem(ctx, m, db, query, opts...)
}

// Insert in database.
func Insert(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *sdk.CDNItem) error {
	i.ID = sdk.UUID()
	i.Created = time.Now()
	i.LastModified = time.Now()

	cdnItem := toItemDB(*i)
	if err := m.InsertAndSign(ctx, db, cdnItem); err != nil {
		return sdk.WrapError(err, "unable to insert item item %s", i.ID)
	}
	*i = cdnItem.CDNItem
	return nil
}

// Update in database
func Update(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *sdk.CDNItem) error {
	i.LastModified = time.Now()
	cdnItem := toItemDB(*i)
	if err := m.UpdateAndSign(ctx, db, cdnItem); err != nil {
		return sdk.WrapError(err, "unable to update item item")
	}
	*i = cdnItem.CDNItem
	return nil
}

func MarkToDeleteByRunIDs(db gorpmapper.SqlExecutorWithTx, runID int64) error {
	query := `
		UPDATE item SET to_delete = true WHERE (api_ref->>'run_id')::int = $1 
	`
	_, err := db.Exec(query, runID)
	return sdk.WrapError(err, "unable to mark item to delete for run %d", runID)
}

// LoadByAPIRefHashAndType load an item by his job id, step order and type
func LoadByAPIRefHashAndType(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, hash string, itemType sdk.CDNItemType, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItem, error) {
	query := gorpmapper.NewQuery(`
		SELECT *
		FROM item
		WHERE api_ref_hash = $1 
		AND type = $2
		AND to_delete = false
	`).Args(hash, itemType)
	return getItem(ctx, m, db, query, opts...)
}

// ComputeSizeByIDs returns the size used by givenn item IDs
func ComputeSizeByIDs(db gorp.SqlExecutor, itemIDs []string) (int64, error) {
	query := `
		SELECT COALESCE(SUM(size), 0) FROM item
		WHERE id = ANY($1) 
	`
	size, err := db.SelectInt(query, pq.StringArray(itemIDs))
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return size, nil
}

func ListNodeRunByProject(db gorp.SqlExecutor, projectKey string) ([]int64, error) {
	var IDs []int64
	query := `
		SELECT 
			DISTINCT((api_ref->>'node_run_id')::int)
		FROM item 
		WHERE api_ref->>'project_key' = $1
	`
	_, err := db.Select(&IDs, query, projectKey)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	return IDs, nil
}

// ComputeSizeByProjectKey returns the size used by a project
func ComputeSizeByProjectKey(db gorp.SqlExecutor, projectKey string) (int64, error) {
	query := `
		SELECT SUM(size) FROM item WHERE api_ref->>'project_key' = $1 
	`
	size, err := db.SelectInt(query, projectKey)
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return size, nil
}

type Stat struct {
	Status string `db:"status"`
	Type   string `db:"type"`
	Number int64  `db:"number"`
}

func CountItems(db gorp.SqlExecutor) (res []Stat, err error) {
	_, err = db.Select(&res, `
	SELECT status, type, count(id) as "number" 
	FROM item 
	WHERE to_delete = false 
	GROUP BY status, type`)
	return res, sdk.WithStack(err)
}

func CountItemsToDelete(db gorp.SqlExecutor) (res []Stat, err error) {
	_, err = db.Select(&res, `
	SELECT type, count(id) as "number" 
	FROM item 
	WHERE to_delete = true 
	GROUP BY type`)
	return res, sdk.WithStack(err)
}

type StatItemPercentil struct {
	Size       int64  `db:"size"`
	Type       string `db:"type"`
	Percentile int64  `db:"percentile"`
}

func CountItemSizePercentil(db gorp.SqlExecutor) (res []StatItemPercentil, err error) {
	_, err = db.Select(&res, `
	with bucket as (
		select type, size, ntile(100) over (partition by type order by size) as percentile 
		from item
		where to_delete = false 
	)
	select bucket.type, bucket.percentile, max(bucket.size) as size
	from bucket
	group by bucket.type, bucket.percentile
	order by bucket.type, bucket.percentile`)
	return res, sdk.WithStack(err)
}
