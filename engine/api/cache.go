package api

import (
	"context"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
)

func (api *API) postPushCacheHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
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

		_, errO := objectstore.Store(&cacheObject, r.Body)
		if errO != nil {
			return sdk.WrapError(errO, "SaveFile>Cannot store cache")
		}

		return nil
	}
}

func (api *API) getPullCacheHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		tag := vars["tag"]

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.ErrInvalidName
		}

		cacheObject := sdk.Cache{
			Project: projectKey,
			Name:    "cache.tar",
			Tag:     tag,
		}

		if objectstore.Instance().TemporaryURLSupported {
			fURL, err := objectstore.FetchTempURL(&cacheObject)
			if err != nil {
				return sdk.WrapError(err, "getPullCacheHandler> Cannot fetch cache object")
			}
			w.Header().Add("Content-Type", "application/x-tar")
			w.Header().Add("Content-Disposition", "attachment; filename=\"cache.tar\"")
			http.Redirect(w, r, fURL, http.StatusMovedPermanently)
			return nil
		}

		ioread, err := objectstore.Fetch(&cacheObject)
		if err != nil {
			return sdk.WrapError(err, "getPullCacheHandler> Cannot fetch artifact cache.tar")
		}
		if _, err := io.Copy(w, ioread); err != nil {
			_ = ioread.Close()
			return sdk.WrapError(err, "getPullCacheHandler> Cannot stream artifact")
		}

		if err := ioread.Close(); err != nil {
			return sdk.WrapError(err, "getPullCacheHandler> Cannot close artifact")
		}
		return nil
	}
}
