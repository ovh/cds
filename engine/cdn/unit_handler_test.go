package cdn

import (
	"context"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMarkItemUnitAsDeleteHandler(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, db := newTestService(t)

	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)

	// Add Storage unit
	unit := sdk.CDNUnit{
		Name:    "cds-backend",
		Created: time.Now(),
		Config:  sdk.ServiceConfig{},
	}
	require.NoError(t, storage.InsertUnit(ctx, s.Mapper, db, &unit))
	// Add Item
	for i := 0; i < 10; i++ {
		it := sdk.CDNItem{
			ID:     sdk.UUID(),
			Size:   12,
			Type:   sdk.CDNTypeItemStepLog,
			Status: sdk.CDNStatusItemIncoming,

			APIRefHash: sdk.RandomString(10),
		}
		require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &it))

		// Add storage unit
		ui := sdk.CDNItemUnit{
			Type:   sdk.CDNTypeItemStepLog,
			ItemID: it.ID,
			UnitID: unit.ID,
		}
		require.NoError(t, storage.InsertItemUnit(ctx, s.Mapper, db, &ui))
	}

	vars := map[string]string{
		"id": unit.ID,
	}
	uri := s.Router.GetRoute("DELETE", s.deleteUnitHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, "DELETE", uri, nil)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)

	uriMarkItem := s.Router.GetRoute("DELETE", s.markItemUnitAsDeleteHandler, vars)
	require.NotEmpty(t, uri)
	reqMarkItem := newRequest(t, "DELETE", uriMarkItem, nil)
	recMarkItem := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(recMarkItem, reqMarkItem)
	require.Equal(t, 204, recMarkItem.Code)

	cpt := 0
	for {
		if cpt >= 10 {
			t.FailNow()
		}
		uis, err := storage.LoadAllItemUnitsToDeleteByUnit(ctx, s.Mapper, db, unit.ID)
		require.NoError(t, err)
		if len(uis) != 10 {
			time.Sleep(250 * time.Millisecond)
			cpt++
			continue
		}

		for _, ui := range uis {
			require.NoError(t, storage.DeleteItemUnit(s.Mapper, db, &ui))
		}
		break
	}

	uriDel := s.Router.GetRoute("DELETE", s.deleteUnitHandler, vars)
	require.NotEmpty(t, uri)
	reqDel := newRequest(t, "DELETE", uriDel, nil)
	recDel := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(recDel, reqDel)
	require.Equal(t, 204, recDel.Code)

}
