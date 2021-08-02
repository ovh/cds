package item

import (
	"context"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
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
		item, err := i.ToCDSItem()
		if err != nil {
			return nil, err
		}

		verifiedItems = append(verifiedItems, item)
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
	item, err := i.ToCDSItem()
	return &item, sdk.WithStack(err)
}

func LoadIDsToDelete(db gorp.SqlExecutor, offset int, limit int) ([]string, error) {
	query := `
		SELECT id
		FROM item
		WHERE to_delete = true
		ORDER BY created ASC
		OFFSET $1
		LIMIT $2
	`
	var ids []string
	if _, err := db.Select(&ids, query, offset, limit); err != nil {
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
	if err := m.InsertAndSign(ctx, db, &cdnItem); err != nil {
		return sdk.WrapError(err, "unable to insert item item %s", i.ID)
	}
	var err error
	*i, err = cdnItem.ToCDSItem()
	if err != nil {
		return err
	}
	return nil
}

// Update in database
func Update(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *sdk.CDNItem) error {
	i.LastModified = time.Now()
	cdnItem := toItemDB(*i)
	if err := m.UpdateAndSign(ctx, db, &cdnItem); err != nil {
		return sdk.WrapError(err, "unable to update item item")
	}
	var err error
	*i, err = cdnItem.ToCDSItem()
	if err != nil {
		return err
	}
	return nil
}

func MarkToDeleteByRunIDs(db gorpmapper.SqlExecutorWithTx, runID int64) error {
	runIdS := strconv.FormatInt(runID, 10)
	query := `
		UPDATE item SET to_delete = true WHERE api_ref->>'run_id' = $1
	`
	_, err := db.Exec(query, runIdS)
	return sdk.WrapError(err, "unable to mark item to delete for run %d", runID)
}

func LoadWorkerCacheItemByProjectAndCacheTag(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, projKey string, cacheTag string) (*sdk.CDNItem, error) {
	query := gorpmapper.NewQuery(`
		SELECT *
		FROM item
		WHERE type = $1
		AND (api_ref->>'project_key')::text = $2
		AND (api_ref->>'cache_tag')::text = $3
		AND to_delete = false
    ORDER BY created DESC
    LIMIT 1
  `).Args(sdk.CDNTypeItemWorkerCache, projKey, cacheTag)
	return getItem(ctx, m, db, query)
}

func LoadWorkerCacheItemsByProjectAndCacheTag(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, projKey string, cacheTag string) ([]sdk.CDNItem, error) {
	query := gorpmapper.NewQuery(`
		SELECT *
		FROM item
		WHERE type = $1
		AND (api_ref->>'project_key')::text = $2
		AND (api_ref->>'cache_tag')::text = $3
		AND to_delete = false
  `).Args(sdk.CDNTypeItemWorkerCache, projKey, cacheTag)
	return getItems(ctx, m, db, query)
}

// LoadByJobRunID load an item by his job id and type
func LoadByJobRunID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, jobRunId int64, itemTypes []string, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItem, error) {
	query := gorpmapper.NewQuery(`
		SELECT *
		FROM item
		WHERE api_ref->>'node_run_job_id' = $1
		AND type = ANY($2)
		AND to_delete = false
	`).Args(strconv.FormatInt(jobRunId, 10), pq.StringArray(itemTypes))
	return getItems(ctx, m, db, query, opts...)
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
	SELECT status, type, count(status) as "number"
	FROM item
	GROUP BY status, type`)
	return res, sdk.WithStack(err)
}

func CountItemsToDelete(db gorp.SqlExecutor) (int64, error) {
	query := `SELECT count(1) as "number"
	FROM item
	WHERE to_delete = true`
	nb, err := db.SelectInt(query)
	return nb, sdk.WithStack(err)
}

type StatItemPercentil struct {
	Size       int64  `db:"size"`
	Type       string `db:"type"`
	Percentile int64  `db:"percentile"`
}

func CountItemSizePercentil(db gorp.SqlExecutor) ([]StatItemPercentil, error) {
	type DBResult struct {
		Type       string  `db:"type"`
		Percent100 float64 `db:"percentile_100"`
		Percent99  float64 `db:"percentile_99"`
		Percent95  float64 `db:"percentile_95"`
		Percent90  float64 `db:"percentile_90"`
		Percent75  float64 `db:"percentile_75"`
		Percent50  float64 `db:"percentile_50"`
	}
	var result []DBResult

	query := `
    SELECT type,
		percentile_cont(1) within group (order by size) as percentile_100,
		percentile_cont(0.99) within group (order by size) as percentile_99,
		percentile_cont(0.95) within group (order by size) as percentile_95,
		percentile_cont(0.90) within group (order by size) as percentile_90,
		percentile_cont(0.75) within group (order by size) as percentile_75,
		percentile_cont(0.50) within group (order by size) as percentile_50
	FROM item
	GROUP BY type
	`
	_, err := db.Select(&result, query)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	res := make([]StatItemPercentil, 0, 5*len(result))
	for _, r := range result {
		res = append(res, StatItemPercentil{
			Type:       r.Type,
			Percentile: 100,
			Size:       int64(r.Percent100),
		})
		res = append(res, StatItemPercentil{
			Type:       r.Type,
			Percentile: 99,
			Size:       int64(r.Percent99),
		})
		res = append(res, StatItemPercentil{
			Type:       r.Type,
			Percentile: 95,
			Size:       int64(r.Percent95),
		})
		res = append(res, StatItemPercentil{
			Type:       r.Type,
			Percentile: 90,
			Size:       int64(r.Percent90),
		})
		res = append(res, StatItemPercentil{
			Type:       r.Type,
			Percentile: 75,
			Size:       int64(r.Percent75),
		})
		res = append(res, StatItemPercentil{
			Type:       r.Type,
			Percentile: 50,
			Size:       int64(r.Percent50),
		})
	}
	return res, sdk.WithStack(err)
}

func LoadOldItemByStatusAndDuration(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, status string, duration int, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItem, error) {
	query := gorpmapper.NewQuery(`
		SELECT item.*
		FROM item
		WHERE
			item.status = $1 AND
            item.last_modified < NOW() - $2 * INTERVAL '1 second'
		ORDER BY item.last_modified ASC
	`).Args(status, duration)
	return getItems(ctx, m, db, query, opts...)
}

func LoadRunResultByRunID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, runID string) ([]sdk.CDNItem, error) {
	query := gorpmapper.NewQuery(`
		WITH allResults AS (
			SELECT api_ref->>'artifact_name' as name, api_ref->>'run_job_id' as run_job_id, id
			FROM item
			WHERE api_ref->>'run_id'::text = $1 AND type = $2  AND to_delete = false
		),
		deduplication AS (
			SELECT distinct on (name) *
			FROM allResults
			ORDER BY name, run_job_id DESC
		)
		SELECT * FROM item WHERE id IN (SELECT id FROM deduplication)
	`).Args(runID, sdk.CDNTypeItemRunResult)
	return getItems(ctx, m, db, query)
}
