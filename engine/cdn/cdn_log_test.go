package cdn

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
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
			WorkerID:  "abcdef-123456",
			StepOrder: 0,
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

	chanMessages := s.handleConnectionChannel(context.TODO())
	require.NoError(t, s.handleLogMessage(context.TODO(), chanMessages, []byte(message)))
	close(chanMessages)

	time.Sleep(100 * time.Millisecond)

	logs, err := workflow.LoadLogs(s.Db, dbj.ID)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, "[ALERT] this is my message\n", logs[0].Val)
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

	logCache.Set(fmt.Sprintf("hatchery-key-%d", signature.Service.HatcheryID), &key.PublicKey, gocache.DefaultExpiration)
	logCache.Set(fmt.Sprintf("service-worker-%s", signature.Service.WorkerName), true, gocache.DefaultExpiration)

	signatureField, err := jws.Sign(sign, signature)
	require.NoError(t, err)

	message := `{"level": 1, "version": "1", "short": "this", "_facility": "fa", "_file": "file",
	"host": "host", "_line":1, "_pid": 1, "_prefix": "prefix", "full_message": "this is my service message", "_Signature": "%s"}`
	message = fmt.Sprintf(message, signatureField)

	chanMessages := s.handleConnectionChannel(context.TODO())
	require.NoError(t, s.handleLogMessage(context.TODO(), chanMessages, []byte(message)))
	close(chanMessages)

	logs, err := workflow.LoadServiceLog(db, dbj.ID, signature.Service.RequirementName)
	require.NoError(t, err)
	require.Equal(t, "this is my service message\n", logs.Val)
}
