package storage

import (
	"context"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type ItemToSync struct {
	ItemID  string    `db:"id"`
	Created time.Time `db:"created"`
}

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
		WHERE unit_id = $2 AND last_modified < NOW() - INTERVAL '15 minutes'
	)
	SELECT item_id
	FROM storage_unit_item
	WHERE item_id = ANY (select item_id from inBuffer)
	GROUP BY item_id
	HAVING COUNT(unit_id) >= $1
	`
	if _, err := db.Select(&itemIDs, query, maxStorageCount, bufferUnitID); err != nil {
		return nil, sdk.WrapError(err, "unable to get item ids")
	}
	return itemIDs, nil
}

func LoadLastItemUnitByJobUnitType(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, jobRunID int64, cdnType sdk.CDNItemType, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItemUnit, error) {
	url := `
		SELECT sui.* FROM storage_unit_item  sui
		JOIN item on item.id = sui.item_id
		WHERE item.api_ref->>'node_run_job_id' = $1 AND sui.unit_id= $2  AND sui.type = $3
        ORDER BY item.api_ref->>'step_order' DESC LIMIT 1
	`
	query := gorpmapper.NewQuery(url).Args(strconv.FormatInt(jobRunID, 10), unitID, cdnType)
	return getItemUnit(ctx, m, db, query, opts...)
}

func LoadLastItemUnitByRunJobIDUnitType(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, runJobID string, cdnType sdk.CDNItemType, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItemUnit, error) {
	url := `
		SELECT sui.* FROM storage_unit_item  sui
		JOIN item on item.id = sui.item_id
		WHERE item.api_ref->>'run_job_id' = $1 AND sui.unit_id= $2  AND sui.type = $3
        ORDER BY item.api_ref->>'step_order' DESC LIMIT 1
	`
	query := gorpmapper.NewQuery(url).Args(runJobID, unitID, cdnType)
	return getItemUnit(ctx, m, db, query, opts...)
}

func LoadItemUnitByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, itemID string, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE unit_id = $1 and item_id = $2 AND to_delete = false LIMIT 1").Args(unitID, itemID)
	return getItemUnit(ctx, m, db, query, opts...)
}

func LoadItemUnitsByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, size *int, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE unit_id = $1 AND to_delete = false ORDER BY last_modified ASC LIMIT $2").Args(unitID, size)
	return getAllItemUnits(ctx, m, db, query, opts...)
}

func HasItemUnitsByUnitAndHashLocator(db gorp.SqlExecutor, unitID string, hashLocator string, itemType sdk.CDNItemType) (bool, error) {
	query := "SELECT id FROM storage_unit_item WHERE unit_id = $1 AND hash_locator = $2 AND type = $3 AND to_delete = false LIMIT 1"
	var ids []string
	_, err := db.Select(&ids, query, unitID, hashLocator, itemType)
	return len(ids) > 0, sdk.WithStack(err)
}

func HashItemUnitByApiRefHash(db gorp.SqlExecutor, apiRefHash string, unitID string) (bool, error) {
	query := `
		SELECT count(sui.id) FROM storage_unit_item sui
		JOIN item on item.id = sui.item_id
		WHERE item.api_ref_hash = $1 AND unit_id = $2
	`
	nb, err := db.SelectInt(query, apiRefHash, unitID)
	if err != nil {
		return false, sdk.WithStack(err)
	}
	return nb > 0, nil
}

func LoadItemUnitByID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE id = $1 AND to_delete = false").Args(id)
	return getItemUnit(ctx, m, db, query, opts...)
}

func getItemUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query, opts ...gorpmapper.GetOptionFunc) (*sdk.CDNItemUnit, error) {
	var i itemUnitDB
	found, err := m.Get(ctx, db, q, &i, opts...)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get storage_unit item with query: %v args:%+v", q.Query, q.Arguments)
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

func LoadAllItemUnitsToDeleteByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, limit int, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE unit_id = $1 AND to_delete = true ORDER BY last_modified ASC LIMIT $2").Args(unitID, limit)
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

func LoadAllItemUnitsIDsByUnitID(db gorp.SqlExecutor, unitID string, offset, limit int64) ([]string, error) {
	var IDs []string
	query := "SELECT id FROM storage_unit_item WHERE unit_id = $1 AND to_delete = false ORDER BY id ASC OFFSET $2 LIMIT $3"
	if _, err := db.Select(&IDs, query, unitID, offset, limit); err != nil {
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

	itemUnitsValid := make([]sdk.CDNItemUnit, 0)
	for x := range itemUnits {
		for y := range items {
			if itemUnits[x].ItemID == items[y].ID {
				itemUnits[x].Item = &items[y]
				break
			}
		}
		// we could have no item found in some case, it this item is purged
		// between the first select from storage_unit_item and the LoadByIDs
		if itemUnits[x].Item != nil {
			itemUnitsValid = append(itemUnitsValid, itemUnits[x])
		}
	}

	return itemUnitsValid, nil
}

func LoadAllItemIDUnknownByUnit(db gorp.SqlExecutor, unitID string, offset int64, limit int64) ([]ItemToSync, error) {
	var res []ItemToSync

	query := `
		WITH inUnit as (
    		SELECT item_id, unit_id
    		FROM storage_unit_item
    		WHERE unit_id = $1
		)
		SELECT item.id, item.created
		FROM item
		LEFT JOIN inUnit on item.id = inUnit.item_id
		WHERE inUnit.unit_id is NULL AND item.status = $2 AND item.to_delete = false
		ORDER BY inUnit.item_id NULLS FIRST,  item.created ASC
		OFFSET $3
		LIMIT $4;
	`

	if _, err := db.Select(&res, query, unitID, sdk.CDNStatusItemCompleted, offset, limit); err != nil {
		return nil, sdk.WithStack(err)
	}

	return res, nil
}

type Stat struct {
	StorageName string `db:"storage_name"`
	Type        string `db:"type"`
	Number      int64  `db:"number"`
}

func CountItemsForUnit(db gorp.SqlExecutor, unitID string) (int64, error) {
	nb, err := db.SelectInt(`select count(id)
	from storage_unit_item
	where unit_id = $1`, unitID)
	return nb, sdk.WithStack(err)
}

func CountItemsForUnitByType(db gorp.SqlExecutor, unitID, stype string) (res []Stat, err error) {
	_, err = db.Select(&res, `select count(type) as "number"
	from storage_unit_item
	where unit_id = $1
	and type = $2`, unitID, stype)
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

func DeleteUnit(m *gorpmapper.Mapper, db gorp.SqlExecutor, u *sdk.CDNUnit) error {
	unitDB := toUnitDB(*u)
	return m.Delete(db, unitDB)
}
