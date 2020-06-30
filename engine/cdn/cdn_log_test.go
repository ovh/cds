package cdn

import (
	"context"
	"encoding/base64"
	"fmt"
	"gopkg.in/h2non/gock.v1"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	gocache "github.com/patrickmn/go-cache"

	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/require"
)

func TestWorkerLog(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	// Create worker private key
	key, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)

	// Create worker signer
	sign, err := jws.NewHMacSigner(key)
	require.NoError(t, err)

	// Create cdn service
	s := Service{
		Db:    db,
		Cache: cache,
	}

	// Create run job
	jobRun := sdk.WorkflowNodeJobRun{
		Start:             time.Now(),
		WorkflowNodeRunID: 1,
		Status:            sdk.StatusBuilding,
	}
	dbj := new(workflow.JobRun)
	require.NoError(t, dbj.ToJobRun(&jobRun))
	require.NoError(t, db.Insert(dbj))

	signature := log.Signature{
		Worker: &log.SignatureWorker{
			WorkerID:   "abcdef-123456",
			StepOrder:  0,
			WorkerName: "myworker",
		},
		JobID:     dbj.ID,
		NodeRunID: jobRun.WorkflowNodeRunID,
		Timestamp: time.Now().UnixNano(),
	}
	logCache.Set(fmt.Sprintf("worker-%s", signature.Worker.WorkerID), sdk.Worker{
		JobRunID:   &signature.JobID,
		PrivateKey: key,
	}, gocache.DefaultExpiration)

	signatureField, err := jws.Sign(sign, signature)
	require.NoError(t, err)

	message := `{"level": 1, "version": "1", "short": "this", "_facility": "fa", "_file": "file",
	"host": "host", "_line":1, "_pid": 1, "_prefix": "prefix", "full_message": "this is my message", "_Signature": "%s"}`
	message = fmt.Sprintf(message, signatureField)

	s.Client = cdsclient.New(cdsclient.Config{
		Host: "http://lolcat.host",
	})

	w := sdk.Worker{
		Name:       signature.Worker.WorkerName,
		ID:         signature.Worker.WorkerID,
		PrivateKey: []byte(base64.StdEncoding.EncodeToString(key)),
		JobRunID:   &signature.JobID,
	}
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())

	//gock.New("http://lolcat.host").Post(fmt.Sprintf("/queue/workflows/%d/log", dbj.ID)).Reply(200)
	gock.New("http://lolcat.host").AddMatcher(func(r *http.Request, rr *gock.Request) (bool, error) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.String(), "http://lolcat.host/worker/myworker") {
			return true, nil
		}
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.String(), fmt.Sprintf("/queue/workflows/%d/log", dbj.ID)) {
			return true, nil
		}
		return false, nil
	}).Reply(200).JSON(w)

	chanMessages := s.handleConnectionChannel(context.TODO())

	require.NoError(t, s.handleLogMessage(context.TODO(), chanMessages, []byte(message)))
	close(chanMessages)

	for i := range chanMessages {
		t.Logf("%s", i.m.Full)
	}

	require.NoError(t, err)
	require.True(t, gock.IsDone())
}

func TestServiceLog(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	// Create hatchery private key
	key, err := jws.NewRandomRSAKey()
	require.NoError(t, err)

	// Create worker signer
	sign, err := jws.NewSigner(key)
	require.NoError(t, err)

	// Create cdn service
	s := Service{
		Db:    db,
		Cache: cache,
	}
	s.Client = cdsclient.New(cdsclient.Config{
		Host: "http://lolcat.host",
	})
	gock.InterceptClient(s.Client.(cdsclient.Raw).HTTPClient())

	// Create run job
	jobRun := sdk.WorkflowNodeJobRun{
		Start:             time.Now(),
		WorkflowNodeRunID: 1,
		Status:            sdk.StatusBuilding,
	}
	dbj := new(workflow.JobRun)
	require.NoError(t, dbj.ToJobRun(&jobRun))
	require.NoError(t, db.Insert(dbj))

	signature := log.Signature{
		Service: &log.SignatureService{
			WorkerName:      "my-worker-name",
			HatcheryID:      1,
			HatcheryName:    "my-hatchery-name",
			RequirementID:   1,
			RequirementName: "service-1",
		},
		JobID:     dbj.ID,
		NodeRunID: jobRun.WorkflowNodeRunID,
		Timestamp: time.Now().UnixNano(),
	}

	// Create worker private key
	wKey, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)
	w := sdk.Worker{
		Name:       signature.Service.WorkerName,
		HatcheryID: signature.Service.HatcheryID,
		PrivateKey: []byte(base64.StdEncoding.EncodeToString(wKey)),
		JobRunID:   &signature.JobID,
	}

	logCache.Set(fmt.Sprintf("hatchery-key-%d", signature.Service.HatcheryID), &key.PublicKey, gocache.DefaultExpiration)
	logCache.Set(fmt.Sprintf("worker-%s", signature.Service.WorkerName), w, gocache.DefaultExpiration)

	signatureField, err := jws.Sign(sign, signature)
	require.NoError(t, err)

	message := `{"level": 1, "version": "1", "short": "this", "_facility": "fa", "_file": "file",
	"host": "host", "_line":1, "_pid": 1, "_prefix": "prefix", "full_message": "this is my service message", "_Signature": "%s"}`
	message = fmt.Sprintf(message, signatureField)

	gock.New("http://lolcat.host").Post("/queue/workflows/log/service").Reply(200)

	require.NoError(t, s.handleLogMessage(context.TODO(), nil, []byte(message)))

	require.True(t, gock.IsDone())

}
