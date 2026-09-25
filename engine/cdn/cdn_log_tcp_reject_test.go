package cdn

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
)

func signLogForWorker(t *testing.T, key []byte, signature cdn.Signature) string {
	signer, err := jws.NewHMacSigner(key)
	require.NoError(t, err)
	sig, err := jws.Sign(signer, signature)
	require.NoError(t, err)
	return sig
}

// The key of a worker is cached by name only, for 20 minutes and without any invalidation. When
// a worker name is reused, the cached entry holds the key of the previous worker: without a
// refresh the whole job loses its logs until the entry expires.
func TestVerifyWorkerV2Log_RefreshStaleKey(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctx := context.TODO()

	workerName := sdk.RandomString(10)
	t.Cleanup(func() {
		runCache.Delete(workerV2CacheKey(workerName))
		runCache.Delete(fmt.Sprintf("workerv2-refresh-%s", workerName))
	})

	currentKey := []byte("the-key-of-the-current-worker-32")
	currentWorker := sdk.V2Worker{
		ID:         sdk.UUID(),
		Name:       workerName,
		JobRunID:   sdk.UUID(),
		PrivateKey: []byte(base64.StdEncoding.EncodeToString(currentKey)),
	}

	// The cache still holds the key of a previous worker that had the same name
	runCache.Set(workerV2CacheKey(workerName), sdk.V2Worker{
		ID:         sdk.UUID(),
		Name:       workerName,
		JobRunID:   sdk.UUID(),
		PrivateKey: []byte("the-key-of-the-previous-worker!!"),
	}, gocache.DefaultExpiration)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockClient := mock_cdsclient.NewMockInterface(ctrl)
	// The key must be fetched again exactly once: one call per rejected line would flood the API
	mockClient.EXPECT().
		V2WorkerGet(gomock.Any(), workerName, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name string, mods ...cdsclient.RequestModifier) (*sdk.V2Worker, error) {
			w := currentWorker
			return &w, nil
		}).
		Times(1)

	s := Service{}
	s.Client = mockClient

	unsafeSign := cdn.Signature{
		RunJobID: currentWorker.JobRunID,
		Worker:   &cdn.SignatureWorker{WorkerID: currentWorker.ID, WorkerName: workerName},
	}
	sig := signLogForWorker(t, currentKey, unsafeSign)

	var signature cdn.Signature
	workerData, err := s.verifyWorkerV2Log(ctx, unsafeSign, sig, &signature)
	require.NoError(t, err)
	require.Equal(t, currentWorker.ID, workerData.ID)
	require.Equal(t, currentWorker.JobRunID, signature.RunJobID)

	// A second line signed with an unknown key must not trigger another call to the API: the
	// refresh is rate limited, a job failing verification on every line would hit the API once
	// per line otherwise.
	runCache.Set(workerV2CacheKey(workerName), currentWorker, gocache.DefaultExpiration)
	badSig := signLogForWorker(t, []byte("a-key-that-nobody-else-knows-321"), unsafeSign)
	_, err = s.verifyWorkerV2Log(ctx, unsafeSign, badSig, &signature)
	require.Error(t, err)
}

// Same stale key window for v1 workers: their key is cached under 'worker-<name>' with the same
// 20 minutes lifetime, so a reused name rejects every line of the job just as it does in v2.
func TestVerifyWorkerLog_RefreshStaleKey(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctx := context.TODO()

	workerName := sdk.RandomString(10)
	t.Cleanup(func() {
		runCache.Delete(workerCacheKey(workerName))
		runCache.Delete(fmt.Sprintf("worker-refresh-%s", workerName))
	})

	var jobID int64 = 42
	currentKey := []byte("the-key-of-the-current-worker-32")
	currentWorker := sdk.Worker{
		ID:         sdk.UUID(),
		Name:       workerName,
		JobRunID:   &jobID,
		PrivateKey: []byte(base64.StdEncoding.EncodeToString(currentKey)),
	}

	// The cache still holds the key of a previous worker that had the same name
	var previousJobID int64 = 41
	runCache.Set(workerCacheKey(workerName), sdk.Worker{
		ID:         sdk.UUID(),
		Name:       workerName,
		JobRunID:   &previousJobID,
		PrivateKey: []byte("the-key-of-the-previous-worker!!"),
	}, gocache.DefaultExpiration)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockClient := mock_cdsclient.NewMockInterface(ctrl)
	// The key must be fetched again exactly once: one call per rejected line would flood the API
	mockClient.EXPECT().
		WorkerGet(gomock.Any(), workerName, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name string, mods ...cdsclient.RequestModifier) (*sdk.Worker, error) {
			w := currentWorker
			return &w, nil
		}).
		Times(1)

	s := Service{}
	s.Client = mockClient

	unsafeSign := cdn.Signature{
		JobID:  jobID,
		Worker: &cdn.SignatureWorker{WorkerID: currentWorker.ID, WorkerName: workerName},
	}
	sig := signLogForWorker(t, currentKey, unsafeSign)

	var signature cdn.Signature
	workerData, err := s.verifyWorkerLog(ctx, unsafeSign, sig, &signature)
	require.NoError(t, err)
	require.Equal(t, currentWorker.ID, workerData.ID)
	require.Equal(t, jobID, signature.JobID)

	// A second line signed with an unknown key must not trigger another call to the API: the
	// refresh is rate limited, a job failing verification on every line would hit the API once
	// per line otherwise.
	runCache.Set(workerCacheKey(workerName), currentWorker, gocache.DefaultExpiration)
	badSig := signLogForWorker(t, []byte("a-key-that-nobody-else-knows-321"), unsafeSign)
	_, err = s.verifyWorkerLog(ctx, unsafeSign, badSig, &signature)
	require.Error(t, err)
}

// The rejection reason must be recorded on the branch that rejects the line: cdn/tcp/errors
// alone cannot tell a log lost on a bad signature from a log lost on an unknown worker.
func TestHandleLogMessage_RecordRejectReason(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctx := context.TODO()

	s := Service{}
	s.GoRoutines = sdk.NewGoRoutines(ctx)
	require.NoError(t, s.initMetrics(ctx))
	require.NotNil(t, s.Metrics.tcpServerLogRejectedCount, "initMetrics must declare the rejected lines measure")

	tagReason := tag.MustNewKey("reason")
	v := &view.View{
		Name:        "test/" + s.Metrics.tcpServerLogRejectedCount.Name(),
		Description: s.Metrics.tcpServerLogRejectedCount.Description(),
		Measure:     s.Metrics.tcpServerLogRejectedCount,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{tagReason},
	}
	require.NoError(t, view.Register(v))
	t.Cleanup(func() { view.Unregister(v) })

	require.Error(t, s.handleLogMessage(ctx, []byte("this is not a gelf message")))
	require.Equal(t, map[string]int64{rejectReasonUnmarshalError: 1}, rejectedCountByReason(t, v.Name))

	require.Error(t, s.handleLogMessage(ctx, []byte(`{"version":"1.1","short_message":"no signature here"}`)))
	require.Equal(t, map[string]int64{
		rejectReasonUnmarshalError:   1,
		rejectReasonSignatureInvalid: 1,
	}, rejectedCountByReason(t, v.Name))
}

func rejectedCountByReason(t *testing.T, viewName string) map[string]int64 {
	rows, err := view.RetrieveData(viewName)
	require.NoError(t, err)
	res := make(map[string]int64, len(rows))
	for _, r := range rows {
		count, ok := r.Data.(*view.CountData)
		require.True(t, ok)
		for _, tg := range r.Tags {
			if tg.Key.Name() == "reason" {
				res[tg.Value] = count.Value
			}
		}
	}
	return res
}

// TestWorkerRejectReason pins the distinction the metric rests on. A 404 is an answer: the worker
// does not exist, and dropping its line is correct. A lookup that failed is not an answer, and a
// line dropped on it is a line lost on missing information. The two must not land in the same
// series, and the error the intake sees is the one getWorker builds, wrapped.
func TestWorkerRejectReason(t *testing.T) {
	notFound := sdk.WrapError(
		sdk.NewError(sdk.ErrNotFound, fmt.Errorf("worker does not exist")),
		"unable to get worker %s", "loving-worker")
	require.Equal(t, rejectReasonWorkerNotFound, workerRejectReason(notFound),
		"the api answered that the worker does not exist")

	lookupFailed := sdk.WrapError(
		fmt.Errorf(`request failed after 1 attempts: Transport Error: Get "https://api/worker/loving-worker?withKey=true": context deadline exceeded`),
		"unable to get worker %s", "loving-worker")
	require.Equal(t, rejectReasonWorkerLookupErr, workerRejectReason(lookupFailed),
		"the api never answered, which is not the same thing")
}

// TestWorkerLookupContext pins why the lookup does not derive its deadline from its caller: a
// derived context can only shorten one. A caller that has already spent its budget would make the
// lookup fail before its request leaves the process, and the intake reads that as a worker that
// does not exist.
func TestWorkerLookupContext(t *testing.T) {
	spent, cancelSpent := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancelSpent()
	<-spent.Done()
	require.Error(t, spent.Err(), "the caller budget must be exhausted for this test to mean anything")

	spent = context.WithValue(spent, cdslog.RequestID, "kept")

	ctx, cancel := workerLookupContext(spent)
	defer cancel()

	require.NoError(t, ctx.Err(), "the lookup must not start already out of time")
	deadline, ok := ctx.Deadline()
	require.True(t, ok, "the lookup must stay bounded")
	require.Greater(t, time.Until(deadline), workerLookupTimeout/2)
	require.Equal(t, "kept", ctx.Value(cdslog.RequestID), "detaching the deadline must not drop the log context")
}
