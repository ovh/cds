package cdn

import (
	"context"
	"encoding/json"

	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/authentication"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log/hook"
)

func TestMarkItemToDeleteHandler(t *testing.T) {
	s, db := newTestService(t)
	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)

	item1 := sdk.CDNItem{
		ID:   sdk.UUID(),
		Type: sdk.CDNTypeItemStepLog,
		APIRef: &sdk.CDNLogAPIRef{
			RunID:      1,
			WorkflowID: 1,
		},
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item1))
	item2 := sdk.CDNItem{
		ID:   sdk.UUID(),
		Type: sdk.CDNTypeItemStepLog,
		APIRef: &sdk.CDNLogAPIRef{
			RunID:      2,
			WorkflowID: 2,
		},
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item2))

	item3 := sdk.CDNItem{
		ID:   sdk.UUID(),
		Type: sdk.CDNTypeItemStepLog,
		APIRef: &sdk.CDNLogAPIRef{
			RunID:      3,
			WorkflowID: 2,
		},
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item3))

	vars := map[string]string{}
	uri := s.Router.GetRoute("POST", s.bulkDeleteItemsHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, "POST", uri, sdk.CDNMarkDelete{RunID: 2})

	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	item3DB, err := item.LoadByID(context.TODO(), s.Mapper, db, item3.ID)
	require.NoError(t, err)
	require.False(t, item3DB.ToDelete)

	item2DB, err := item.LoadByID(context.TODO(), s.Mapper, db, item2.ID)
	require.NoError(t, err)
	require.True(t, item2DB.ToDelete)

	item1DB, err := item.LoadByID(context.TODO(), s.Mapper, db, item1.ID)
	require.NoError(t, err)
	require.False(t, item1DB.ToDelete)

	// Test deleteItemHandler
	vars = map[string]string{
		"apiRef": item3.APIRefHash,
		"type":   string(item3.Type),
	}
	uri = s.Router.GetRoute("DELETE", s.deleteItemHandler, vars)
	req = newRequest(t, "DELETE", uri, nil)
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	item3DB, err = item.LoadByID(context.TODO(), s.Mapper, db, item3.ID)
	require.NoError(t, err)
	require.True(t, item3DB.ToDelete)

}

func TestGetItemLogsDownloadHandler(t *testing.T) {
	projectKey := sdk.RandomString(10)
	// Create cdn service with need storage and test item
	s, db := newTestService(t)
	s.Client = cdsclient.New(cdsclient.Config{Host: "http://lolcat.api", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())
	gock.New("http://lolcat.api").Get("/project/" + projectKey + "/workflows/1/type/step-log/access").Reply(http.StatusOK).JSON(nil)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, s.Mapper, db.DbMap, ctx, s.Cache)

	hm := handledMessage{
		Msg: hook.Message{
			Full: "this is a message",
		},
		IsTerminated: sdk.StatusTerminated,
		Signature: cdn.Signature{
			ProjectKey:   projectKey,
			WorkflowID:   1,
			WorkflowName: "MyWorkflow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &cdn.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}

	content := buildMessage(hm)
	err := s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content)
	require.NoError(t, err)

	signer, err := authentication.NewSigner("cdn-test", test.SigningKey)
	require.NoError(t, err)
	s.Common.ParsedAPIPublicKey = signer.GetVerifyKey()
	authSessionJWTClaims := sdk.AuthSessionJWTClaims{
		ID: sdk.UUID(),
		StandardClaims: jwt.StandardClaims{
			Issuer:    "test",
			Subject:   sdk.UUID(),
			Id:        sdk.UUID(),
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute).Unix(),
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, authSessionJWTClaims)
	jwtTokenRaw, err := signer.SignJWT(jwtToken)
	require.NoError(t, err)

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey:     hm.Signature.ProjectKey,
		WorkflowName:   hm.Signature.WorkflowName,
		WorkflowID:     hm.Signature.WorkflowID,
		RunID:          hm.Signature.RunID,
		NodeRunName:    hm.Signature.NodeRunName,
		NodeRunID:      hm.Signature.NodeRunID,
		NodeRunJobName: hm.Signature.JobName,
		NodeRunJobID:   hm.Signature.JobID,
		StepName:       hm.Signature.Worker.StepName,
		StepOrder:      hm.Signature.Worker.StepOrder,
	}
	apiRefHashU, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)
	apiRefHash := strconv.FormatUint(apiRefHashU, 10)

	uri := s.Router.GetRoute("GET", s.getItemDownloadHandler, map[string]string{
		"type":   string(sdk.CDNTypeItemStepLog),
		"apiRef": apiRefHash,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtTokenRaw, "GET", uri, nil)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	assert.Equal(t, "this is a message\n", string(rec.Body.Bytes()))

	uri = s.Router.GetRoute("GET", s.getItemHandler, map[string]string{
		"type":   string(sdk.CDNTypeItemStepLog),
		"apiRef": apiRefHash,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtTokenRaw, "GET", uri, nil)
	req.URL.Query().Add("withDecryption", "true")
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	t.Log(rec.Body.String())

}

func TestGetItemArtifactDownloadHandler(t *testing.T) {
	projectKey := sdk.RandomString(10)

	// Create cdn service with need storage and test item
	s, db := newTestService(t)

	cdntest.ClearItem(t, context.Background(), s.Mapper, db)

	s.Client = cdsclient.New(cdsclient.Config{Host: "http://lolcat.api", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())
	t.Cleanup(gock.OffAll)
	gock.New("http://lolcat.api").Get("/project/" + projectKey + "/workflows/1/type/run-result/access").Reply(http.StatusOK).JSON(nil)

	gock.New("http://lolcat.api").Post("/queue/workflows/3/run/results/check").Reply(http.StatusNoContent)
	gock.New("http://lolcat.api").Post("/queue/workflows/3/run/results").Reply(http.StatusNoContent)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, s.Mapper, db.DbMap, ctx, s.Cache)

	fileContent := []byte("Hi, I am foo.")
	myartifact, errF := os.Create(path.Join(os.TempDir(), "myartifact"))
	defer os.RemoveAll(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, errF)
	_, errW := myartifact.Write(fileContent)
	require.NoError(t, errW)
	myartifact.Close()

	f, err := os.Open(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, err)

	sig := cdn.Signature{
		ProjectKey:   projectKey,
		WorkflowName: "WfName",
		WorkflowID:   1,
		RunID:        2,
		JobID:        3,
		JobName:      "JobDownload",
		Worker: &cdn.SignatureWorker{
			FileName:      "myartifact",
			WorkerName:    "myworker",
			RunResultType: string(sdk.WorkflowRunResultTypeArtifact),
		},
	}

	require.NoError(t, s.storeFile(ctx, sig, f, StoreFileOptions{}))

	signer, err := authentication.NewSigner("cdn-test", test.SigningKey)
	require.NoError(t, err)
	s.Common.ParsedAPIPublicKey = signer.GetVerifyKey()
	authSessionJWTClaims := sdk.AuthSessionJWTClaims{
		ID: sdk.UUID(),
		StandardClaims: jwt.StandardClaims{
			Issuer:    "test",
			Subject:   sdk.UUID(),
			Id:        sdk.UUID(),
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute).Unix(),
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, authSessionJWTClaims)
	jwtTokenRaw, err := signer.SignJWT(jwtToken)
	require.NoError(t, err)

	apiRef := sdk.NewCDNRunResultApiRef(sig)
	refhash, err := apiRef.ToHash()
	require.NoError(t, err)

	uri := s.Router.GetRoute("GET", s.getItemDownloadHandler, map[string]string{
		"type":   string(sdk.CDNTypeItemRunResult),
		"apiRef": refhash,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtTokenRaw, "GET", uri, nil)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	assert.Equal(t, string(fileContent), string(rec.Body.Bytes()))

	its, err := item.LoadAll(ctx, s.Mapper, db, 1, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	// Sync inbackend
	// Sync in local storage
	unit, err := storage.LoadUnitByName(ctx, s.Mapper, db, s.Units.Storages[0].Name())
	require.NoError(t, err)
	// Waiting FS sync
	cpt := 0
	for {
		ids, err := storage.LoadAllItemUnitsIDsByItemIDsAndUnitID(db, unit.ID, []string{its[0].ID})
		require.NoError(t, err)
		if len(ids) == 1 {
			break
		}
		if cpt == 20 {
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

	// Delete from buffer
	iuInBuffers, err := storage.LoadAllItemUnitsIDsByItemIDsAndUnitID(db, s.Units.FileBuffer().ID(), []string{its[0].ID})
	require.NoError(t, err)
	require.Equal(t, 1, len(iuInBuffers))
	iuBuffer, err := storage.LoadItemUnitByID(ctx, s.Mapper, db, iuInBuffers[0])
	require.NoError(t, err)
	require.NoError(t, storage.DeleteItemUnit(s.Mapper, db, iuBuffer))

	// Download again
	uri = s.Router.GetRoute("GET", s.getItemDownloadHandler, map[string]string{
		"type":   string(sdk.CDNTypeItemRunResult),
		"apiRef": refhash,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtTokenRaw, "GET", uri, nil)
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	assert.Equal(t, string(fileContent), string(rec.Body.Bytes()))
	for _, r := range gock.Pending() {
		t.Logf("Pending call: %s", r.Request().URLStruct.String())
	}
	assert.True(t, gock.IsDone())
}

func TestGetItemsArtefactHandler(t *testing.T) {
	s, db := newTestService(t)

	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)
	cdntest.ClearSyncRedisSet(t, s.Cache, "local_storage")

	workerSignature := cdn.Signature{
		Timestamp:    time.Now().Unix(),
		ProjectKey:   "projKey",
		WorkflowID:   1,
		JobID:        1,
		JobName:      "my job",
		RunID:        1,
		WorkflowName: "my workflow",
		Worker: &cdn.SignatureWorker{
			WorkerID:      "1",
			WorkerName:    "workername",
			FileName:      "myartifact",
			RunResultType: string(sdk.WorkflowRunResultTypeArtifact),
		},
	}

	item1 := sdk.CDNItem{
		Type:   sdk.CDNTypeItemRunResult,
		Status: sdk.CDNStatusItemCompleted,
		APIRef: sdk.NewCDNRunResultApiRef(workerSignature),
	}
	refhash, err := item1.APIRef.ToHash()
	require.NoError(t, err)
	item1.APIRefHash = refhash
	require.NoError(t, item.Insert(context.Background(), s.Mapper, db, &item1))

	item2 := sdk.CDNItem{
		Type:   sdk.CDNTypeItemStepLog,
		Status: sdk.CDNStatusItemCompleted,
		APIRef: sdk.NewCDNLogApiRef(workerSignature),
	}
	refhashLog, err := item2.APIRef.ToHash()
	require.NoError(t, err)
	item2.APIRefHash = refhashLog
	require.NoError(t, item.Insert(context.Background(), s.Mapper, db, &item2))

	workerSignature.JobID += 1
	item3 := sdk.CDNItem{
		Type:   sdk.CDNTypeItemRunResult,
		Status: sdk.CDNStatusItemCompleted,
		APIRef: sdk.NewCDNRunResultApiRef(workerSignature),
	}
	refhash, err = item3.APIRef.ToHash()
	require.NoError(t, err)
	item3.APIRefHash = refhash
	require.NoError(t, item.Insert(context.Background(), s.Mapper, db, &item3))

	vars := map[string]string{
		"type": string(sdk.CDNTypeItemRunResult),
	}
	uri := s.Router.GetRoute("GET", s.getItemsHandler, vars)
	require.NotEmpty(t, uri)
	req := newRequest(t, "GET", uri, nil)
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 400, rec.Code)

	req = newRequest(t, "GET", uri+"?runid=1", nil)
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var results []sdk.CDNItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))

	require.Equal(t, 1, len(results))
	require.Equal(t, item3.ID, results[0].ID)
}
