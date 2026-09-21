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
	w, err := s.Client.V2WorkerGet(ctx, workerName, cdsclient.WithQueryParameter("withKey", "true"))
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
	w, err := s.Client.WorkerGet(ctx, workerName, cdsclient.WithQueryParameter("withKey", "true"))
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
