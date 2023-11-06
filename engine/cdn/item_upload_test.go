package cdn

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ovh/symmecrypt/keyloader"

	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/jws"
)

func TestPostUploadHandler(t *testing.T) {
	s, db := newTestService(t)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	cdntest.ClearSyncRedisSet(t, s.Cache, "local_storage")

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
				Redis: &storage.RedisBufferConfiguration{
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

	// Mock cds client
	s.Client = cdsclient.New(cdsclient.Config{Host: "http://lolcat.api", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())
	t.Cleanup(gock.OffAll)

	// Create worker
	workerKey, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)
	jobRunID := int64(1)
	worker := sdk.Worker{
		ID:         "1",
		Name:       "myworker",
		JobRunID:   &jobRunID,
		PrivateKey: []byte(base64.StdEncoding.EncodeToString(workerKey)),
	}

	gock.New("http://lolcat.api").Post("/queue/workflows/1/run/results/check").Reply(http.StatusNoContent)
	gock.New("http://lolcat.api").Post("/queue/workflows/1/run/results").Reply(http.StatusNoContent)
	gock.New("http://lolcat.api").Get("/worker/myworker").MatchParam("withKey", "true").Reply(200).JSON(worker)

	workerSignature := cdn.Signature{
		Timestamp:    time.Now().Unix(),
		ProjectKey:   "projKey",
		WorkflowID:   1,
		JobID:        1,
		JobName:      "my job",
		RunID:        1,
		WorkflowName: "myworkflow",
		Worker: &cdn.SignatureWorker{
			WorkerID:      worker.ID,
			WorkerName:    worker.Name,
			FileName:      "myartifact",
			RunResultType: string(sdk.WorkflowRunResultTypeArtifact),
		},
	}
	signer, err := jws.NewHMacSigner(workerKey)
	require.NoError(t, err)

	signature, err := jws.Sign(signer, workerSignature)
	require.NoError(t, err)

	// Create artifact
	fileContent := []byte("Hi, I am foo.")
	myartifact, errF := os.Create(path.Join(os.TempDir(), "myartifact"))
	defer os.RemoveAll(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, errF)
	_, errW := myartifact.Write(fileContent)
	require.NoError(t, errW)

	errClose := myartifact.Close()
	require.NoError(t, errClose)

	f, err := os.Open(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, err)
	defer f.Close()
	h := md5.New()
	_, err = io.Copy(h, f)
	require.NoError(t, err)
	md5Sum := hex.EncodeToString(h.Sum(nil))
	require.NoError(t, f.Close())

	f, err = os.Open(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, err)
	defer f.Close()
	hasher := sha512.New()
	_, err = io.Copy(hasher, f)
	require.NoError(t, err)
	sha := hex.EncodeToString(hasher.Sum(nil))
	require.NoError(t, f.Close())

	vars := map[string]string{}
	uri := s.Router.GetRoute("POST", s.postUploadHandler, vars)
	require.NotEmpty(t, uri)

	moreHeaders := map[string]string{
		"X-CDS-WORKER-SIGNATURE": signature,
	}
	f, err = os.Open(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, err)
	req := assets.NewUploadFileRequest(t, "POST", uri, f, moreHeaders)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	for _, r := range gock.Pending() {
		t.Logf("PENDING: %s \n", r.Request().URLStruct.String())
	}
	f.Close()
	require.Equal(t, 204, rec.Code)
	require.True(t, gock.IsDone())

	its, err := item.LoadAll(ctx, s.Mapper, db, 1, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	require.Equal(t, 1, len(its))
	require.Equal(t, sha, its[0].Hash)
	require.Equal(t, md5Sum, its[0].MD5)
	require.Equal(t, int64(len(fileContent)), its[0].Size)
	require.Equal(t, sdk.CDNStatusItemCompleted, its[0].Status)

	// Read from Local Buffer
	localBuffer := s.Units.FileBuffer()
	iuBuffer, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, db, localBuffer.ID(), its[0].ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	reader, err := localBuffer.NewReader(ctx, *iuBuffer)
	require.NoError(t, err)
	buf := &bytes.Buffer{}
	err = localBuffer.Read(*iuBuffer, reader, buf)
	require.NoError(t, err)
	require.Equal(t, string(fileContent), buf.String())

	// Sync in local storage
	unit, err := storage.LoadUnitByName(ctx, s.Mapper, db, s.Units.Storages[0].Name())
	require.NoError(t, err)
	// Waiting FS sync
	cpt := 0
	var iuID string
	for {
		ids, err := storage.LoadAllItemUnitsIDsByItemIDsAndUnitID(db, unit.ID, []string{its[0].ID})
		require.NoError(t, err)
		if len(ids) == 1 {
			iuID = ids[0]
			break
		}
		if cpt == 10 {
			t.Logf("No sync in FS")
			t.Fail()
			return
		}
		if len(ids) != 1 {
			cpt++
			time.Sleep(500 * time.Millisecond)
			continue
		}
	}

	iu, err := storage.LoadItemUnitByID(ctx, s.Mapper, db, iuID, gorpmapper.GetOptions.WithDecryption)
	buf = &bytes.Buffer{}

	uiRead, err := s.Units.Storages[0].NewReader(ctx, *iu)
	defer uiRead.Close()
	require.NoError(t, err)

	require.NoError(t, s.Units.Storages[0].Read(*iu, uiRead, buf))
	uiRead.Close()

	require.Equal(t, string(fileContent), buf.String())
}

func TestPostUploadHandler_WorkflowV2(t *testing.T) {
	s, db := newTestService(t)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)
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
				Redis: &storage.RedisBufferConfiguration{
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

	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	cdntest.ClearSyncRedisSet(t, s.Cache, "local_storage")

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	mockCDSClient := mock_cdsclient.NewMockInterface(ctrl)
	s.Client = mockCDSClient

	workerKey, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)

	mockCDSClient.EXPECT().V2WorkerGet(gomock.Any(), "wk.Name()", gomock.Any()).Return(
		&sdk.V2Worker{
			Name:       "wk.Name()",
			PrivateKey: []byte(base64.StdEncoding.EncodeToString(workerKey)),
		}, nil,
	)

	mockCDSClient.EXPECT().V2QueueJobRunResultGet(gomock.Any(), "region", "wk.currentJobV2.runJob.ID", "runResult.ID").Return(
		&sdk.V2WorkflowRunResult{
			Status: sdk.V2WorkflowRunResultStatusPending,
		}, nil,
	)

	vars := map[string]string{}
	uri := s.Router.GetRoute("POST", s.postUploadHandler, vars)
	require.NotEmpty(t, uri)

	sig := cdn.Signature{
		JobName:       "wk.currentJobV2.runJob.Job.Name",
		RunJobID:      "wk.currentJobV2.runJob.ID",
		Region:        "region",
		ProjectKey:    "wk.currentJobV2.runJob.ProjectKey",
		WorkflowName:  "wk.currentJobV2.runJob.WorkflowName",
		WorkflowRunID: "wk.currentJobV2.runJob.WorkflowRunID",
		RunNumber:     1,
		RunAttempt:    0,
		Timestamp:     time.Now().UnixNano(),
		Worker: &cdn.SignatureWorker{
			WorkerID:      "wk.id",
			WorkerName:    "wk.Name()",
			RunResultID:   "runResult.ID",
			RunResultName: "runResult.Name()",
			RunResultType: "runResult.Typ()",
		},
	}

	signer, err := jws.NewHMacSigner(workerKey)
	require.NoError(t, err)

	signature, err := jws.Sign(signer, sig)
	require.NoError(t, err)

	moreHeaders := map[string]string{
		"X-CDS-WORKER-SIGNATURE": signature,
	}

	req := assets.NewUploadFileRequest(t, "POST", uri, bytes.NewBufferString("foo bar"), moreHeaders)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
}
