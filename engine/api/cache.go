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

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.ErrInvalidName
		}

		if r.Body == nil {
			return sdk.ErrWrongRequest
		}
		defer r.Body.Close()

		cacheObject := sdk.Cache{
			Name:    "cache.tar",
			Project: projectKey,
			Tag:     tag,
		}

		_, errO := objectstore.StoreArtifact(&cacheObject, r.Body)
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
			Name:    "cache.tar",
			Tag:     tag,
		}

		fURL, err := objectstore.FetchTempURL(&cacheObject)
		if err != nil {
			return sdk.WrapError(err, "getPullCacheHandler> Cannot fetch cache object")
		}

		w.Header().Add("Content-Type", "application/x-tar")
		w.Header().Add("Content-Disposition", "attachment; filename=\"cache.tar\"")
		http.Redirect(w, r, fURL, http.StatusMovedPermanently)
		return nil
	}
}
