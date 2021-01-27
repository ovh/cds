package cdn

import (
	"context"
	"encoding/json"
	"github.com/ovh/cds/engine/cdn/item"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetItemsArtefactHandler(t *testing.T) {
	s, db := newTestService(t)
	s.Cfg.EnableLogProcessing = true

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
			WorkerID:     "1",
			WorkerName:   "workername",
			ArtifactName: "myartifact",
		},
	}

	item1 := sdk.CDNItem{
		Type:   sdk.CDNTypeItemArtifact,
		Status: sdk.CDNStatusItemCompleted,
		APIRef: sdk.NewCDNArtifactApiRef(workerSignature),
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

	vars := map[string]string{
		"type": string(sdk.CDNTypeItemArtifact),
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
	require.Equal(t, item1.ID, results[0].ID)
}
