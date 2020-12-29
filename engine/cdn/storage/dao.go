package storage

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query) (*sdk.CDNUnit, error) {
	var u unitDB
	found, err := m.Get(ctx, db, q, &u)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get storage_unit")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := m.CheckSignature(u, u.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "storage.getUnit> storage_unit %s data corrupted", u.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &u.CDNUnit, nil
}

// LoadUnitByID returns a unit from database for given id.
func LoadUnitByID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string) (*sdk.CDNUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit WHERE id = $1").Args(id)
	return getUnit(ctx, m, db, query)
}

// LoadUnitByName returns a unit from database for given name.
func LoadUnitByName(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, name string) (*sdk.CDNUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit WHERE name = $1").Args(name)
	return getUnit(ctx, m, db, query)
}

// InsertUnit in database.
func InsertUnit(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *sdk.CDNUnit) error {
	i.ID = sdk.UUID()
	i.Created = time.Now()

	unitDB := toUnitDB(*i)
	if err := m.InsertAndSign(ctx, db, unitDB); err != nil {
		return sdk.WrapError(err, "unable to insert storage unit")
	}
	*i = unitDB.CDNUnit
	return nil
}

type LoadUnitOptionFunc func(context.Context, gorp.SqlExecutor, ...*unitDB) error

// LoadAllUnits loads all the units from the database
func LoadAllUnits(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, opts ...LoadUnitOptionFunc) ([]sdk.CDNUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit")
	return getAllUnits(ctx, m, db, query, opts...)
}

func getAllUnits(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, query gorpmapper.Query, opts ...LoadUnitOptionFunc) ([]sdk.CDNUnit, error) {
	var res []unitDB
	if err := m.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	// Check signature of data, if invalid do not return it
	verifiedUnits := make([]*unitDB, 0, len(res))
	for i := range res {
		isValid, err := m.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "storage.getAll> storage_unit %s data corrupted", res[i].ID)
			continue
		}
		verifiedUnits = append(verifiedUnits, &res[i])
	}

	if len(verifiedUnits) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, verifiedUnits...); err != nil {
				return nil, err
			}
		}
	}

	units := make([]sdk.CDNUnit, len(verifiedUnits))
	for i := range verifiedUnits {
		units[i] = verifiedUnits[i].CDNUnit
	}

	return units, nil
}

func InsertItemUnit(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, iu *sdk.CDNItemUnit) error {
	if iu.ID == "" {
		iu.ID = sdk.UUID()
	}
	itemUnitDN := toItemUnitDB(*iu)
	if err := m.InsertAndSign(ctx, db, itemUnitDN); err != nil {
		return sdk.WrapError(err, "unable to insert storage unit item")
	}
	return nil
}

func MarkItemUnitToDelete(db gorpmapper.SqlExecutorWithTx, ids []string) (int, error) {
	res, err := db.Exec(`UPDATE storage_unit_item SET to_delete = true WHERE id = ANY($1)`, pq.StringArray(ids))
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	n, err := res.RowsAffected()
	return int(n), sdk.WithStack(err)
}

func DeleteItemUnit(m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, iu *sdk.CDNItemUnit) error {
	itemUnitDN := toItemUnitDB(*iu)
	if err := m.Delete(db, itemUnitDN); err != nil {
		return sdk.WrapError(err, "unable to delete item unit %s", iu.ID)
	}
	return nil
}

func LoadAllSynchronizedItemIDs(db gorp.SqlExecutor, bufferUnitID string, maxStorageCount int64) ([]string, error) {
	var itemIDs []string
	query := `
	WITH inBuffer as (
		SELECT item_id 
		FROM storage_unit_item
		WHERE unit_id = $2
	)
	SELECT item_id
	FROM storage_unit_item
	WHERE item_id = ANY (select item_id from inBuffer)
	GROUP BY item_id
	HAVING COUNT(unit_id) = $1
	`
	if _, err := db.Select(&itemIDs, query, maxStorageCount, bufferUnitID); err != nil {
		return nil, sdk.WrapError(err, "unable to get item ids")
	}
	return itemIDs, nil
}

func LoadOldItemUnitByItemStatusAndDuration(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, status string, duration int, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery(`
		SELECT storage_unit_item.*
		FROM storage_unit_item
		LEFT JOIN item ON item.id = storage_unit_item.item_id
		WHERE
			item.status = $1 AND
            item.last_modified < NOW() - $2 * INTERVAL '1 second'
		ORDER BY item.last_modified ASC
	`).Args(status, duration)
	return getAllItemUnits(ctx, m, db, query, opts...)
}

func LoadItemUnitByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, itemID string, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE unit_id = $1 and item_id = $2 AND to_delete = false LIMIT 1").Args(unitID, itemID)
	return getItemUnit(ctx, m, db, query, opts...)
}

func LoadItemUnitsByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, size *int, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE unit_id = $1 AND to_delete = false ORDER BY last_modified ASC LIMIT $2").Args(unitID, size)
	return getAllItemUnits(ctx, m, db, query, opts...)
}

func CountItemUnitsByUnitAndHashLocator(db gorp.SqlExecutor, unitID string, hashLocator string, size *int) (int64, error) {
	query := "SELECT COUNT(*) FROM storage_unit_item WHERE unit_id = $1 AND hash_locator = $2 AND to_delete = false LIMIT $3"
	nb, err := db.SelectInt(query, unitID, hashLocator, size)
	return nb, sdk.WithStack(err)
}

func LoadItemUnitByID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE id = $1 AND to_delete = false").Args(id)
	return getItemUnit(ctx, m, db, query, opts...)
}

func getItemUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItemUnit, error) {
	var i itemUnitDB
	found, err := m.Get(ctx, db, q, &i, opts...)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get storage_unit item")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := m.CheckSignature(i, i.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "storage.getItemUnit> storage_unit_item %s data corrupted", i.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	i.Item, err = item.LoadByID(ctx, m, db, i.ItemID, opts...)
	if err != nil {
		return nil, err
	}

	return &i.CDNItemUnit, nil
}

func CountItemUnitsToDeleteByItemID(db gorp.SqlExecutor, itemID string) (int64, error) {
	query := "SELECT COUNT(id) FROM storage_unit_item WHERE item_id = $1 AND to_delete = true"
	nb, err := db.SelectInt(query, itemID)
	return nb, sdk.WithStack(err)
}

func LoadAllItemUnitsToDeleteByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE unit_id = $1 AND to_delete = true ORDER BY last_modified ASC").Args(unitID)
	return getAllItemUnits(ctx, m, db, query, opts...)
}

func LoadAllItemUnitsIDsByItemIDsAndUnitID(db gorp.SqlExecutor, unitID string, itemID []string) ([]string, error) {
	var IDs []string
	query := "SELECT id FROM storage_unit_item WHERE item_id = ANY($1) AND unit_id = $2 AND to_delete = false"
	if _, err := db.Select(&IDs, query, pq.StringArray(itemID), unitID); err != nil {
		return nil, sdk.WithStack(err)
	}
	return IDs, nil
}

func LoadAllItemUnitsIDsByItemID(db gorp.SqlExecutor, itemID string) ([]string, error) {
	var IDs []string
	query := "SELECT storage_unit_item.id FROM storage_unit_item WHERE item_id = $1 AND to_delete = false"
	if _, err := db.Select(&IDs, query, itemID); err != nil {
		return nil, sdk.WithStack(err)
	}
	return IDs, nil
}

func LoadAllItemUnitsByItemIDs(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, itemID string, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE item_id = $1 AND to_delete = false").Args(itemID)
	allItemUnits, err := getAllItemUnits(ctx, m, db, query, opts...)
	return allItemUnits, sdk.WithStack(err)
}

func getAllItemUnits(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, query gorpmapper.Query, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItemUnit, error) {
	var res []itemUnitDB
	if err := m.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	// Check signature of data, if invalid do not return it
	verifiedItems := make([]*itemUnitDB, 0, len(res))
	for i := range res {
		isValid, err := m.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "storage.getAllItemUnits> storage_unit_item %s data corrupted", res[i].ID)
			continue
		}
		verifiedItems = append(verifiedItems, &res[i])
	}

	itemUnits := make([]sdk.CDNItemUnit, len(verifiedItems))
	itemIDs := make([]string, len(verifiedItems))
	for i := range verifiedItems {
		itemUnits[i] = verifiedItems[i].CDNItemUnit
		itemIDs[i] = itemUnits[i].ItemID
	}

	items, err := item.LoadByIDs(ctx, m, db, itemIDs, opts...)
	if err != nil {
		return nil, err
	}

	for x := range itemUnits {
		for y := range items {
			if itemUnits[x].ItemID == items[y].ID {
				itemUnits[x].Item = &items[y]
				break
			}
		}
	}

	return itemUnits, nil
}

func CountItemCompleted(db gorp.SqlExecutor) (int64, error) {
	return db.SelectInt("SELECT COUNT(id) from item WHERE item.status = $1 AND to_delete = false", sdk.CDNStatusItemCompleted)
}

func CountItemIncoming(db gorp.SqlExecutor) (int64, error) {
	return db.SelectInt("SELECT COUNT(id) from item WHERE item.status <> $1", sdk.CDNStatusItemCompleted)
}

func CountItemUnitByUnit(db gorp.SqlExecutor, unitID string) (int64, error) {
	return db.SelectInt("SELECT COUNT(unit_id) from storage_unit_item WHERE unit_id = $1", unitID)
}

func LoadAllItemIDUnknownByUnit(db gorp.SqlExecutor, unitID string, syncMinNbElements, maxLimit int64) ([]string, error) {
	var res []string

	countItems, err := CountItemCompleted(db)
	if err != nil {
		return res, err
	}
	countStorageUnitItems, err := CountItemUnitByUnit(db, unitID)
	if err != nil {
		return res, err
	}

	// Compute the diff to evaluate the count of items to sync for given unit
	expectedCountItemToSync := countItems - countStorageUnitItems
	if expectedCountItemToSync <= syncMinNbElements {
		return res, nil
	}

	limit := expectedCountItemToSync
	if expectedCountItemToSync > maxLimit {
		limit = maxLimit
	}

	// When diff is greater than limit we also order by unit_id to improve query response time
	var query string
	if expectedCountItemToSync > maxLimit {
		query = `
      SELECT item.id
      FROM item
      LEFT JOIN storage_unit_item sui ON item.id = sui.item_id AND sui.unit_id = $1
      WHERE item.status = $3 AND sui.unit_id is null
      AND item.to_delete = false
      ORDER BY sui.unit_id NULLS FIRST, item.created DESC
      LIMIT $2
    `
	} else {
		query = `
      SELECT item.id
      FROM item
      LEFT JOIN storage_unit_item sui ON item.id = sui.item_id AND sui.unit_id = $1
      WHERE item.status = $3 AND sui.unit_id is null
      AND item.to_delete = false
      ORDER BY item.created DESC
      LIMIT $2
    `
	}

	if _, err := db.Select(&res, query, unitID, limit, sdk.CDNStatusItemCompleted); err != nil {
		return nil, sdk.WithStack(err)
	}

	return res, nil
}

type Stat struct {
	StorageName string `db:"-"`
	Type        string `db:"type"`
	Number      int64  `db:"number"`
}

func CountItemsForUnit(db gorp.SqlExecutor, unitID string) (res []Stat, err error) {
	_, err = db.Select(&res, `select type, count(id) as "number"
	from storage_unit_item
	where unit_id = $1
	group by type`, unitID)
	return res, sdk.WithStack(err)
}

func CountItemUnitToDelete(db gorp.SqlExecutor) (res []Stat, err error) {
	_, err = db.Select(&res, `select storage_unit.name as "storage_name", item.type, count(storage_unit_item.id) as "number"
	from storage_unit_item
	join item on item.id = storage_unit_item.item_id
	join storage_unit on storage_unit.id = storage_unit_item.unit_id AND storage_unit_item.to_delete = true
	group by storage_unit.name, item.type`)
	return res, sdk.WithStack(err)
}
