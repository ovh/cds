package cdn

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	gocache "github.com/patrickmn/go-cache"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	runCache = gocache.New(20*time.Minute, 20*time.Minute)
)

// workerKeyRefreshTTL bounds how often the key of a given worker can be fetched again from the
// API after a signature verification failure: a job whose every line fails verification would
// otherwise hit the API once per log line.
const workerKeyRefreshTTL = time.Minute

// workerLookupTimeout bounds a whole lookup of a worker on the API, retries included.
const workerLookupTimeout = 4 * time.Second

// workerLookupAttemptTimeout bounds one attempt of that lookup. It is what makes the retries
// below real: the client already retries twice, but its attempts share a single context and it
// stops as soon as that context is done, so one slow attempt spends the whole budget and no
// retry ever happens. A blip shorter than an attempt is absorbed; a slow attempt is abandoned
// in time for the next one.
const workerLookupAttemptTimeout = time.Second

// workerLookupAttempts is how many times a lookup is tried before giving up.
const workerLookupAttempts = 3

// workerLookupBackoff separates two attempts, so that a blip has time to pass.
const workerLookupBackoff = 200 * time.Millisecond

// workerLookupFailureTTL remembers, briefly, that looking a worker up just failed. The intake
// handles the messages of a connection one at a time, so without this every line of a job whose
// worker cannot be looked up would pay the full budget in turn and stall the connection for
// minutes. The lines are dropped either way; this only stops them from being dropped slowly.
const workerLookupFailureTTL = 5 * time.Second

// workerLookupContext gives a lookup a deadline of its own instead of whatever the caller happens
// to carry. Deriving from the caller would change nothing: a derived context can only shorten a
// deadline, never lift one, so a lookup entered with an exhausted budget fails before its request
// leaves the process. The intake then reads that failure as "this worker does not exist" and drops
// the log line, which is how a caller running out of time turns into logs lost for good.
//
// The detachment is bounded by workerLookupTimeout and the result is cached, so at worst the
// lookup outlives its caller by that long.
func workerLookupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), workerLookupTimeout)
}

func workerLookupFailureKey(workerName string) string {
	return fmt.Sprintf("worker-lookup-failed-%s", workerName)
}

// retryWorkerLookup runs fetch until it succeeds, until the API answers that the worker does not
// exist, or until the attempts or the overall budget run out. Only a lookup that failed without an
// answer is worth trying again: a 404 is an answer, and repeating it would just double the load.
//
// A lookup that ends without an answer is remembered for workerLookupFailureTTL, so the next lines
// of the same job fail immediately instead of each paying the budget again.
func retryWorkerLookup(ctx context.Context, workerName string, fetch func(context.Context) error) error {
	if _, failedRecently := runCache.Get(workerLookupFailureKey(workerName)); failedRecently {
		return sdk.NewErrorFrom(sdk.ErrServiceUnavailable, "worker %s could not be looked up recently", workerName)
	}

	lookupCtx, cancel := workerLookupContext(ctx)
	defer cancel()

	var err error
	for attempt := 0; attempt < workerLookupAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.NewTimer(workerLookupBackoff)
			select {
			case <-lookupCtx.Done():
			case <-backoff.C:
			}
			backoff.Stop()
			if lookupCtx.Err() != nil {
				break
			}
		}

		attemptCtx, cancelAttempt := context.WithTimeout(lookupCtx, workerLookupAttemptTimeout)
		err = fetch(attemptCtx)
		cancelAttempt()

		if err == nil {
			return nil
		}
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if lookupCtx.Err() != nil {
			break
		}
	}

	runCache.Set(workerLookupFailureKey(workerName), true, workerLookupFailureTTL)
	return err
}

func workerV2CacheKey(workerName string) string {
	return fmt.Sprintf("workerv2-%s", workerName)
}

// refreshWorkerV2Key evicts the cached key of the worker and fetches it again from the API. It
// returns false without calling the API when the key has already been refreshed recently.
func (s *Service) refreshWorkerV2Key(ctx context.Context, workerName string) (sdk.V2Worker, bool, error) {
	refreshKey := fmt.Sprintf("workerv2-refresh-%s", workerName)
	if _, ok := runCache.Get(refreshKey); ok {
		return sdk.V2Worker{}, false, nil
	}
	runCache.Set(refreshKey, true, workerKeyRefreshTTL)
	runCache.Delete(workerV2CacheKey(workerName))

	w, err := s.getWorkerV2(ctx, workerName, GetWorkerOptions{NeedPrivateKey: true})
	if err != nil {
		return sdk.V2Worker{}, false, err
	}
	return w, true, nil
}

func (s *Service) getWorkerV2(ctx context.Context, workerName string, opts GetWorkerOptions) (sdk.V2Worker, error) {
	workerKey := workerV2CacheKey(workerName)

	// Get worker from cache
	cacheData, ok := runCache.Get(workerKey)
	if ok {
		w, ok := cacheData.(sdk.V2Worker)
		if ok && (!opts.NeedPrivateKey || len(w.PrivateKey) > 0) {
			return w, nil
		}
	}

	// Get worker from API
	var w *sdk.V2Worker
	err := retryWorkerLookup(ctx, workerName, func(attemptCtx context.Context) error {
		var fetchErr error
		w, fetchErr = s.Client.V2WorkerGet(attemptCtx, workerName, cdsclient.WithQueryParameter("withKey", "true"))
		return fetchErr
	})
	if err != nil {
		return sdk.V2Worker{}, sdk.WrapError(err, "unable to get worker %s", workerName)
	}

	privateKeyDecoded, err := base64.StdEncoding.DecodeString(string(w.PrivateKey))
	if err != nil {
		return sdk.V2Worker{}, sdk.WithStack(err)
	}
	w.PrivateKey = privateKeyDecoded
	runCache.Set(workerKey, *w, gocache.DefaultExpiration)

	return *w, nil
}
func workerCacheKey(workerName string) string {
	return fmt.Sprintf("worker-%s", workerName)
}

// refreshWorkerKey is the v1 twin of refreshWorkerV2Key: same name keyed cache, same 20 minutes
// lifetime, so same need to fetch the key again once before rejecting a whole job.
func (s *Service) refreshWorkerKey(ctx context.Context, workerName string) (sdk.Worker, bool, error) {
	refreshKey := fmt.Sprintf("worker-refresh-%s", workerName)
	if _, ok := runCache.Get(refreshKey); ok {
		return sdk.Worker{}, false, nil
	}
	runCache.Set(refreshKey, true, workerKeyRefreshTTL)
	runCache.Delete(workerCacheKey(workerName))

	w, err := s.getWorker(ctx, workerName, GetWorkerOptions{NeedPrivateKey: true})
	if err != nil {
		return sdk.Worker{}, false, err
	}
	return w, true, nil
}

func (s *Service) getWorker(ctx context.Context, workerName string, opts GetWorkerOptions) (sdk.Worker, error) {
	workerKey := workerCacheKey(workerName)

	// Get worker from cache
	cacheData, ok := runCache.Get(workerKey)
	if ok {
		w, ok := cacheData.(sdk.Worker)
		if ok && (!opts.NeedPrivateKey || len(w.PrivateKey) > 0) {
			return w, nil
		}
	}

	// Get worker from API
	var w *sdk.Worker
	err := retryWorkerLookup(ctx, workerName, func(attemptCtx context.Context) error {
		var fetchErr error
		w, fetchErr = s.Client.WorkerGet(attemptCtx, workerName, cdsclient.WithQueryParameter("withKey", "true"))
		return fetchErr
	})
	if err != nil {
		return sdk.Worker{}, sdk.WrapError(err, "unable to get worker %s", workerName)
	}
	privateKeyDecoded, err := base64.StdEncoding.DecodeString(string(w.PrivateKey))
	if err != nil {
		return sdk.Worker{}, sdk.WithStack(err)
	}
	w.PrivateKey = privateKeyDecoded
	runCache.Set(workerKey, *w, gocache.DefaultExpiration)

	return *w, nil
}
