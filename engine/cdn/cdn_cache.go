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

func (s *Service) getWorkerV2(ctx context.Context, workerName string, opts GetWorkerOptions) (sdk.V2Worker, error) {
	workerKey := fmt.Sprintf("workerv2-%s", workerName)

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
func (s *Service) getWorker(ctx context.Context, workerName string, opts GetWorkerOptions) (sdk.Worker, error) {
	workerKey := fmt.Sprintf("worker-%s", workerName)

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
