package cdn

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/ovh/symmecrypt/keyloader"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
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
		uis, err := storage.LoadAllItemUnitsToDeleteByUnit(ctx, s.Mapper, db, unit.ID, 100)
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

func TestPostAdminResyncBackendWithDatabaseHandler(t *testing.T) {
	s, db := newTestService(t)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	cdntest.ClearSyncRedisSet(context.TODO(), t, s.Cache, "local_storage")

	// Start CDN
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	tmpDir2, err := os.MkdirTemp("", t.Name()+"-cdn-2-*")
	require.NoError(t, err)

	cdnUnits, err := storage.Init(ctx, s.Mapper, s.Cache, db.DbMap, sdk.NewGoRoutines(ctx), storage.Configuration{
		SyncSeconds:     1,
		SyncNbElements:  1000,
		PurgeNbElements: 1000,
		PurgeSeconds:    30,
		HashLocatorSalt: "thisismysalt",
		Buffers: map[string]storage.BufferConfiguration{
			"refis_buffer": {
				Redis: &sdk.RedisConf{
					Host:     cfg["redisHost"],
					Password: cfg["redisPassword"],
					DbIndex:  0,
				},
				BufferType: storage.CDNBufferTypeLog,
			},
			"local_buffer": {
				Local: &storage.LocalBufferConfiguration{
					Path: tmpDir,
					Encryption: []*keyloader.KeyConfig{
						{
							Key:        "iamakey.iamakey.iamakey.iamakey.",
							Cipher:     aesgcm.CipherName,
							Identifier: "local-bukker-id",
						},
					},
				},
				BufferType: storage.CDNBufferTypeFile,
			},
		},
		Storages: map[string]storage.StorageConfiguration{
			"local_storage": {
				SyncParallel:  10,
				SyncBandwidth: int64(1024 * 1024),
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir2,
					Encryption: []convergent.ConvergentEncryptionConfig{
						{
							Cipher:      aesgcm.CipherName,
							LocatorSalt: "secret_locator_salt",
							SecretValue: "secret_value",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	s.Units = cdnUnits
	cdnUnits.Start(ctx, sdk.NewGoRoutines(ctx))

	// Create an Item
	it, err := s.loadOrCreateItem(context.TODO(), sdk.CDNTypeItemRunResult, cdn.Signature{
		RunID:      1,
		JobID:      1,
		WorkflowID: 1,
		NodeRunID:  1,
		Worker: &cdn.SignatureWorker{
			WorkerID:      "1",
			FileName:      sdk.RandomString(10),
			RunResultType: string(sdk.WorkflowRunResultTypeArtifact),
			FilePerm:      0777,
			StepOrder:     0,
			WorkerName:    sdk.RandomString(10),
		},
	})
	require.NoError(t, err)
	_, err = s.loadOrCreateItemUnitBuffer(context.TODO(), it.ID, sdk.CDNTypeItemRunResult)
	require.NoError(t, err)

	require.NoError(t, os.Mkdir(fmt.Sprintf("%s/%s", tmpDir, string(sdk.CDNTypeItemRunResult)), 0755))

	file1Path := fmt.Sprintf("%s/%s/%s", tmpDir, string(sdk.CDNTypeItemRunResult), it.APIRefHash)
	t.Logf("Creating file %s", file1Path)
	content1 := []byte("I'm the real one")
	f1, err := os.Create(file1Path)
	require.NoError(t, err)
	defer f1.Close()
	_, err = f1.Write(content1)
	require.NoError(t, err)

	file2Path := fmt.Sprintf("%s/%s/%s", tmpDir, string(sdk.CDNTypeItemRunResult), "wronghash")
	t.Logf("Creating file %s", file2Path)
	content2 := []byte("I'm not the real one")
	f2, err := os.Create(file2Path)
	require.NoError(t, err)
	defer f2.Close()
	_, err = f2.Write(content2)
	require.NoError(t, err)

	vars := make(map[string]string)
	vars["id"] = s.Units.GetBuffer(sdk.CDNTypeItemRunResult).ID()
	vars["type"] = string(sdk.CDNTypeItemRunResult)
	uri := s.Router.GetRoute("POST", s.postAdminResyncBackendWithDatabaseHandler, vars) + "?dryRun=false"
	require.NotEmpty(t, uri)
	req := newRequest(t, "POST", uri, nil)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	cpt := 0
	for {
		_, err := os.Stat(file2Path)
		if os.IsNotExist(err) {
			break
		}
		if cpt >= 20 {
			t.FailNow()
		}
		cpt++
		time.Sleep(250 * time.Millisecond)
	}
	_, err = os.Stat(file2Path)
	require.True(t, os.IsNotExist(err))

	_, err = os.Stat(file1Path)
	require.NoError(t, err)

}
