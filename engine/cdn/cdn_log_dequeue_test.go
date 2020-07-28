package cdn

import (
	"context"
	"github.com/mitchellh/hashstructure"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestStoreNewStepLog(t *testing.T) {
	m := gorpmapper.New()
	index.Init(m)
	db, cache := test.SetupPGWithMapper(t, m, sdk.TypeCDN)

	// Create cdn service
	s := Service{
		Db:     db.DbMap,
		Cache:  cache,
		Mapper: m,
	}

	hm := handledMessage{
		Msg:    hook.Message{},
		Status: "Building",
		Line:   1,
		Signature: log.Signature{
			ProjectKey:   "KEY",
			WorkflowID:   1,
			WorkflowName: "MyWorklow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &log.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}
	err := s.storeStepLogs(context.TODO(), hm)
	require.NoError(t, err)

	apiRef := index.ApiRef{
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
	hashRef, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)
	item, err := index.LoadItemByApiRefHashAndType(context.TODO(), s.Mapper, db, strconv.FormatUint(hashRef, 10), index.TypeItemStepLog)
	require.NoError(t, err)
	require.NotNil(t, item)
	require.Equal(t, index.StatusItemIncoming, item.Status)

}
