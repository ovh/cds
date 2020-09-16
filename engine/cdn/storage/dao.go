package storage

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query) (*Unit, error) {
	var u Unit
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
		log.Error(ctx, "index.get> storage_unit %s data corrupted", u.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &u, nil
}

// LoadUnitByID returns a unit from database for given id.
func LoadUnitByID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string) (*Unit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit WHERE id = $1").Args(id)
	return getUnit(ctx, m, db, query)
}

// LoadUnitByName returns a unit from database for given name.
func LoadUnitByName(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, name string) (*Unit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit WHERE name = $1").Args(name)
	return getUnit(ctx, m, db, query)
}

// InsertUnit in database.
func InsertUnit(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *Unit) error {
	i.ID = sdk.UUID()
	i.Created = time.Now()
	if err := m.InsertAndSign(ctx, db, i); err != nil {
		return sdk.WrapError(err, "unable to insert storage unit")
	}
	return nil
}

type LoadUnitOptionFunc func(context.Context, gorp.SqlExecutor, ...*Unit) error

// LoadAllUnits loads all the units from the database
func LoadAllUnits(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, opts ...LoadUnitOptionFunc) ([]Unit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit")
	return getAllUnits(ctx, m, db, query, opts...)
}

func getAllUnits(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, query gorpmapper.Query, opts ...LoadUnitOptionFunc) ([]Unit, error) {
	var res []Unit
	if err := m.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	// Check signature of data, if invalid do not return it
	verifiedUnits := make([]*Unit, 0, len(res))
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

	units := make([]Unit, len(verifiedUnits))
	for i := range verifiedUnits {
		units[i] = *verifiedUnits[i]
	}

	return units, nil
}

func InsertItemUnit(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, iu *ItemUnit) error {
	if iu.ID == "" {
		iu.ID = sdk.UUID()
	}
	if err := m.InsertAndSign(ctx, db, iu); err != nil {
		return sdk.WrapError(err, "unable to insert storage unit iotem")
	}
	return nil
}

func UpdateItemUnit(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, u *ItemUnit) error {
	if err := m.UpdateAndSign(ctx, db, u); err != nil {
		return sdk.WrapError(err, "unable to update storage unit item")
	}
	return nil
}

func DeleteItemUnit(m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, u *ItemUnit) error {
	if err := m.Delete(db, u); err != nil {
		return sdk.WrapError(err, "unable to delete item unit %s", u.ID)
	}
	return nil
}

func DeleteItemsUnit(db gorp.SqlExecutor, unitID string, itemIDs []string) error {
	query := `
		DELETE FROM storage_unit_index
		WHERE unit_id = $1 AND item_id = ANY($2)
	`
	_, err := db.Exec(query, unitID, pq.StringArray(itemIDs))
	return sdk.WrapError(err, "unable to remove items from unit %s", itemIDs)
}

// LoadAllItemsIDInBufferAndAllUnitsExceptCDS loads all that are presents in all backend ( except cds backend )
func LoadAllItemsIDInBufferAndAllUnitsExceptCDS(db gorp.SqlExecutor, cdsBackendID string) ([]string, error) {
	var itemIDs []string
	query := `
		SELECT item_id 
		FROM (
			SELECT COUNT(*) as nb, item_id 
			FROM storage_unit_index
			WHERE unit_id != $1
			GROUP BY item_id
		) as cc
		WHERE nb = (SELECT COUNT(*) FROM storage_unit WHERE id != $1)
	`
	if _, err := db.Select(&itemIDs, query, cdsBackendID); err != nil {
		return nil, sdk.WrapError(err, "unable to get item ids")
	}
	return itemIDs, nil
}

func LoadOldItemUnitByItemStatusAndDuration(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, status string, duration int, opts ...gorpmapper.GetOptionFunc) ([]ItemUnit, error) {
	query := gorpmapper.NewQuery(`
		SELECT storage_unit_index.*
		FROM storage_unit_index
		LEFT JOIN index ON index.id = storage_unit_index.item_id
		WHERE
			index.status = $1 AND
            index.last_modified < NOW() - $2 * INTERVAL '1 second'
		ORDER BY index.last_modified ASC
	`).Args(status, duration)
	return getAllItemUnits(ctx, m, db, query, opts...)
}

func LoadItemUnitByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, itemID string, opts ...gorpmapper.GetOptionFunc) (*ItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_index WHERE unit_id = $1 and item_id = $2 LIMIT 1").Args(unitID, itemID)
	return getItemUnit(ctx, m, db, query, opts...)
}

func LoadItemUnitsByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, size int, opts ...gorpmapper.GetOptionFunc) ([]ItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_index WHERE unit_id = $1 ORDER BY last_modified ASC LIMIT $2").Args(unitID, size)
	return getAllItemUnits(ctx, m, db, query, opts...)
}

func LoadItemUnitByID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string, opts ...gorpmapper.GetOptionFunc) (*ItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_index WHERE id = $1").Args(id)
	return getItemUnit(ctx, m, db, query, opts...)
}

func getItemUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, q gorpmapper.Query, opts ...gorpmapper.GetOptionFunc) (*ItemUnit, error) {
	var i ItemUnit
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
		log.Error(ctx, "index.get> storage_unit_index %s data corrupted", i.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	i.Item, err = index.LoadItemByID(ctx, m, db, i.ItemID, opts...)
	if err != nil {
		return nil, err
	}

	return &i, nil
}

func LoadAllItemUnitsByItemID(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, itemID string, opts ...gorpmapper.GetOptionFunc) ([]ItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_index WHERE item_id = $1").Args(itemID)
	return getAllItemUnits(ctx, m, db, query, opts...)
}

func getAllItemUnits(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, query gorpmapper.Query, opts ...gorpmapper.GetOptionFunc) ([]ItemUnit, error) {
	var res []ItemUnit
	if err := m.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	// Check signature of data, if invalid do not return it
	verifiedItems := make([]*ItemUnit, 0, len(res))
	for i := range res {
		isValid, err := m.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "storage.getAllItemUnits> storage_unit_index %s data corrupted", res[i].ID)
			continue
		}
		verifiedItems = append(verifiedItems, &res[i])
	}

	itemUnits := make([]ItemUnit, len(verifiedItems))
	itemIDs := make([]string, len(verifiedItems))
	for i := range verifiedItems {
		itemUnits[i] = *verifiedItems[i]
		itemIDs = append(itemIDs, itemUnits[i].ItemID)
	}

	items, err := index.LoadItemByIDs(ctx, m, db, itemIDs, opts...)
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

func LoadAllItemIDUnknownByUnit(db gorp.SqlExecutor, unitID string, limit int) ([]string, error) {
	query := `
		SELECT * 
		FROM (
			SELECT index.id 
			FROM index
			JOIN storage_unit_index ON index.id = storage_unit_index.item_id
			WHERE index.status = $3
			EXCEPT 
			SELECT item_id
			FROM storage_unit_index  
			WHERE unit_id = $1
		) IDS
		LIMIT $2
	`

	var res []string
	if _, err := db.Select(&res, query, unitID, limit, index.StatusItemCompleted); err != nil {
		return nil, sdk.WithStack(err)
	}

	return res, nil
}
