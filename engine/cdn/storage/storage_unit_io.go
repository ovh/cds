package storage

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	rs  = rand.NewSource(time.Now().Unix())
	rnd = rand.New(rs)
)

type Source interface {
	NewReader(context.Context) (io.ReadCloser, error)
	Read(io.Reader, io.Writer) error
	Name() string
	SyncBandwidth() float64
}

type source interface {
	NewReader(context.Context, sdk.CDNItemUnit) (io.ReadCloser, error)
	Read(sdk.CDNItemUnit, io.Reader, io.Writer) error
	Name() string
	SyncBandwidth() float64
}

type iuSource struct {
	iu     sdk.CDNItemUnit
	source source
}

func (s *iuSource) NewReader(ctx context.Context) (io.ReadCloser, error) {
	return s.source.NewReader(ctx, s.iu)
}
func (s *iuSource) Read(r io.Reader, w io.Writer) error {
	return s.source.Read(s.iu, r, w)
}
func (s *iuSource) Name() string {
	return s.source.Name()
}
func (s *iuSource) SyncBandwidth() float64 {
	return s.source.SyncBandwidth()
}

func (r RunningStorageUnits) GetSource(ctx context.Context, i *sdk.CDNItem) (Source, error) {
	ok, err := r.Buffer.ItemExists(ctx, r.m, r.db, *i)
	if err != nil {
		return nil, err
	}

	if ok {
		iu, err := LoadItemUnitByUnit(ctx, r.m, r.db, r.Buffer.ID(), i.ID, gorpmapper.GetOptions.WithDecryption)
		if err != nil {
			return nil, err
		}
		return &iuSource{iu: *iu, source: r.Buffer}, nil
	}

	// Find a storage unit where the item is complete
	mapItemUnits, err := LoadAllItemUnitsByItemIDs(ctx, r.m, r.db, []string{i.ID})
	if err != nil {
		return nil, err
	}

	var itemUnits = mapItemUnits[i.ID]
	if len(itemUnits) == 0 {
		log.Warning(ctx, "item %s can't be found. No unit knows it...", i.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	// Random pick a unit
	idx := 0
	if len(itemUnits) > 1 {
		idx = rnd.Intn(len(itemUnits))
	}
	refItemUnit := itemUnits[idx]
	refUnitID := refItemUnit.UnitID
	refUnit, err := LoadUnitByID(ctx, r.m, r.db, refUnitID)
	if err != nil {
		return nil, err
	}

	unit := r.Storage(refUnit.Name)
	if unit == nil {
		return nil, sdk.WithStack(fmt.Errorf("unable to find unit %s", refUnit.Name))
	}

	return &iuSource{iu: refItemUnit, source: unit}, nil
}

func (r RunningStorageUnits) GetItemUnitByLocatorByUnit(ctx context.Context, locator string, unitID string, opts ...gorpmapper.GetOptionFunc) ([]sdk.CDNItemUnit, error) {
	// Load all the itemUnit for the unit and the same hashLocator
	hashLocator := r.HashLocator(locator)
	return LoadItemUnitsByUnitAndHashLocator(ctx, r.m, r.db, unitID, hashLocator, nil, opts...)
}
