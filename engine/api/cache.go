package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
)

func (api *API) postPushCacheHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		tag := vars["tag"]

		if r.Body == nil {
			return sdk.ErrWrongRequest
		}

		cacheObject := sdk.Cache{
			Name:    "cache.tar.gz",
			Project: projectKey,
			Tag:     tag,
		}

		_, errO := objectstore.StoreArtifact(&cacheObject, r.Body)
		defer r.Body.Close()
		if errO != nil {
			return sdk.WrapError(errO, "SaveFile>Cannot store cache")
		}

		return nil
	}
}

func (api *API) getPullCacheHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		tag := vars["tag"]

		cacheObject := sdk.Cache{
			Project: projectKey,
			Name:    "cache.tar.gz",
			Tag:     tag,
		}

		fURL, err := objectstore.FetchTempURL(&cacheObject)
		if err != nil {
			return sdk.WrapError(err, "pullCacheHandler> Cannot fetch cache object")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", "attachment; filename=\"cache.tar.gz\"")
		http.Redirect(w, r, fURL, http.StatusMovedPermanently)
		return nil
	}
}
