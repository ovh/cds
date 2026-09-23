package cdn

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/mitchellh/hashstructure"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook/graylog"
	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

func TestStoreTruncatedLogs(t *testing.T) {
	t.SkipNow()
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.Factory = log.NewTestingWrapper(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}
	s.GoRoutines = sdk.NewGoRoutines(context.TODO())

	ctx, ccl := context.WithCancel(context.TODO())
	t.Cleanup(ccl)
	cdnUnits := newRunningStorageUnits(t, m, db.DbMap, ctx, cache)
	s.Units = cdnUnits

	hm := handledMessage{
		Msg: graylog.Message{
			Full: "Bim bam boum",
		},
		IsTerminated: false,
		Signature: cdn.Signature{
			ProjectKey:   sdk.RandomString(10),
			WorkflowID:   1,
			WorkflowName: "MyWorklow",
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
	apiRef := &sdk.CDNLogAPIRef{
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

	it := sdk.CDNItem{
		Status:     sdk.CDNStatusItemIncoming,
		APIRefHash: strconv.FormatUint(hashRef, 10),
		APIRef:     apiRef,
		Type:       sdk.CDNTypeItemStepLog,
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &it))
	defer func() {
		_ = item.DeleteByID(db, it.ID)

	}()
	content := buildMessage(hm)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content)
	require.NoError(t, err)

	hm.IsTerminated = true
	hm.Msg.Full = "End of step"

	content = buildMessage(hm)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content)
	require.NoError(t, err)

	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB)
	require.Equal(t, sdk.CDNStatusItemCompleted, itemDB.Status)
	require.NotEmpty(t, itemDB.Hash)
	require.NotEmpty(t, itemDB.MD5)
	require.NotZero(t, itemDB.Size)

	unit, err := storage.LoadUnitByName(context.TODO(), m, db, s.Units.LogsBuffer().Name())
	require.NoError(t, err)
	require.NotNil(t, unit)

	itemUnit, err := storage.LoadItemUnitByUnit(context.TODO(), m, db, unit.ID, itemDB.ID)
	require.NoError(t, err)
	require.NotNil(t, itemUnit)

	_, lineCount, rc, _, err := s.getItemLogValue(ctx, sdk.CDNTypeItemStepLog, strconv.FormatUint(hashRef, 10), getItemLogOptions{
		format: sdk.CDNReaderFormatText,
		from:   0,
		size:   10000,
		sort:   1,
	})
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rc)
	require.NoError(t, err)

	require.Equal(t, "Bim bam boum\n...truncated\n", buf.String())
	require.Equal(t, int64(2), lineCount)
}

func TestReadTCPMessagesDropsOversizedMessage(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	globalRateLimit = NewRateLimiter(ctx, 1024*1024*1024, 1024)

	s := Service{}
	s.Cfg.Log.StepMaxSize = 1024
	s.Cfg.Log.StepLinesRateLimit = 1000

	client, server := net.Pipe()
	var received []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.readTCPMessages(ctx, server, func(_ context.Context, msg []byte) error {
			received = append(received, string(msg))
			return nil
		})
	}()

	// An oversized message must be dropped whole, the following one must still be handled
	_, err := client.Write(append(bytes.Repeat([]byte("x"), 3*1024), 0))
	require.NoError(t, err)
	_, err = client.Write(append([]byte("small message"), 0))
	require.NoError(t, err)
	require.NoError(t, client.Close())
	<-done

	require.Equal(t, []string{"small message"}, received)
}

// End to end through the real read loop, with a frame built exactly as a worker builds it and
// the production step size: the oversized message must be discarded whole, the next one must
// still be delivered, and the warning must name the job rather than only the tcp peer.
//
// This is the test that actually proves the tail works. The identification depends on the gelf
// marshalling putting the extra fields AFTER the multi megabyte full_message, so it can only be
// trusted when exercised on a real frame that overflows the buffer for real.
func TestReadTCPMessages_NamesTheSenderOfADroppedOversizedMessage(t *testing.T) {
	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	logger.SetLevel(logrus.DebugLevel)
	log.Factory = log.NewLogrusWrapper(logger)
	t.Cleanup(func() { log.Factory = log.NewTestingWrapper(t) })

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	globalRateLimit = NewRateLimiter(ctx, 1024*1024*1024, 1024)

	s := Service{}
	s.Cfg.Log.StepMaxSize = 3000000 // the value used in production
	s.Cfg.Log.StepLinesRateLimit = 100000

	key, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)
	signer, err := jws.NewHMacSigner(key)
	require.NoError(t, err)
	token, err := jws.Sign(signer, cdn.Signature{
		ProjectKey: "THEKEY", WorkflowName: "the-workflow", RunJobID: "the-run-job-id", JobName: "the-job",
		Worker: &cdn.SignatureWorker{WorkerName: "the-worker", StepName: "the-step"},
	})
	require.NoError(t, err)

	marshal := func(full string) []byte {
		m := graylog.Message{
			Version: "1.1", Host: "a-host", Short: "short", Full: full,
			Extra: map[string]interface{}{"_" + cdslog.ExtraFieldSignature: token},
		}
		b, err := json.Marshal(&m)
		require.NoError(t, err)
		return b
	}

	// 6MB of content, twice the budget: the signature sits megabytes past the point where the
	// buffer is thrown away, so only the tail can still carry it.
	oversized := marshal(strings.Repeat("x", 6*1024*1024))
	require.Greater(t, len(oversized), 6*1024*1024)
	small := marshal("a normal line")

	client, server := net.Pipe()
	var received [][]byte
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.readTCPMessages(ctx, server, func(_ context.Context, msg []byte) error {
			received = append(received, append([]byte{}, msg...))
			return nil
		})
	}()

	_, err = client.Write(append(oversized, 0))
	require.NoError(t, err)
	_, err = client.Write(append(small, 0))
	require.NoError(t, err)
	require.NoError(t, client.Close())
	<-done

	require.Len(t, received, 1, "the oversized message must be dropped, the next one delivered")
	require.Equal(t, small, received[0])

	out := logs.String()
	require.Contains(t, out, "message dropped")
	require.Contains(t, out, "project=THEKEY")
	require.Contains(t, out, "workflow=the-workflow")
	require.Contains(t, out, "run_job_id=the-run-job-id")
	require.Contains(t, out, "worker=the-worker")
	require.Contains(t, out, "step=the-step")
	require.NotContains(t, out, "unidentified sender")
}
