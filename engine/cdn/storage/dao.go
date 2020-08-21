package storage

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
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
	log.Debug("storage.LoadUnitByName> name=%s", name)

	query := gorpmapper.NewQuery("SELECT * FROM storage_unit WHERE name = $1").Args(name)
	return getUnit(ctx, m, db, query)
}

// InsertUnit in database.
func InsertUnit(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, i *Unit) error {
	log.Debug("storage.InsertUnit> %+v", i)
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

func InsertItemUnit(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, unitID, itemID string) (*ItemUnit, error) {
	var iu = ItemUnit{
		ID:     sdk.UUID(),
		ItemID: itemID,
		UnitID: unitID,
	}
	if err := m.InsertAndSign(ctx, db, &iu); err != nil {
		return nil, sdk.WrapError(err, "unable to insert storage unit iotem")
	}
	return &iu, nil
}

func UpdateItemUnit(ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx, u *ItemUnit) error {
	if err := m.UpdateAndSign(ctx, db, u); err != nil {
		return sdk.WrapError(err, "unable to update storage unit item")
	}
	return nil
}

func LoadItemByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, itemID string, opts ...gorpmapper.GetOptionFunc) (*ItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_index WHERE unit_id = $1 and item_id = $2").Args(unitID, itemID)
	return getItemUnit(ctx, m, db, query)
}

func LoadAndLockItemByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, itemID string, opts ...gorpmapper.GetOptionFunc) (*ItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_index WHERE unit_id = $1 and item_id = $2 FOR UPDATE SKIP LOCKED").Args(unitID, itemID)
	return getItemUnit(ctx, m, db, query)
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

	items := make([]ItemUnit, len(verifiedItems))
	for i := range verifiedItems {
		items[i] = *verifiedItems[i]
	}

	return items, nil
}

func LoadAllItemIDUnknownByUnit(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, unitID string, limit int) ([]string, error) {
	query := `SELECT id
		FROM index
		EXCEPT
		SELECT item_id
		FROM storage_unit_index
		WHERE unit_id = $1
		LIMIT $2
	`

	var res []string
	if _, err := db.Select(&res, query, unitID, limit); err != nil {
		return nil, sdk.WithStack(err)
	}

	return res, nil
}
