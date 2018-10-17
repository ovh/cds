package api

import (
	"context"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postPushCacheHandler() service.Handler {
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
			return sdk.WrapError(errO, "postPushCacheHandler>Cannot store cache")
		}

		return nil
	}
}

func (api *API) getPullCacheHandler() service.Handler {
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
				return sdk.WrapError(err, "Cannot fetch cache object")
			}
			w.Header().Add("Content-Type", "application/x-tar")
			w.Header().Add("Content-Disposition", "attachment; filename=\"cache.tar\"")
			http.Redirect(w, r, fURL, http.StatusMovedPermanently)
			return nil
		}

		ioread, err := objectstore.Fetch(&cacheObject)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "getPullCacheHandler> Cannot fetch artifact cache.tar : %v", err)
		}
		if _, err := io.Copy(w, ioread); err != nil {
			_ = ioread.Close()
			return sdk.WrapError(err, "Cannot stream artifact")
		}

		if err := ioread.Close(); err != nil {
			return sdk.WrapError(err, "Cannot close artifact")
		}
		return nil
	}
}

func (api *API) postPushCacheWithTempURLHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		tag := vars["tag"]

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.ErrInvalidName
		}

		store, ok := objectstore.Storage().(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.WrapError(sdk.ErrNotImplemented, "postPushCacheWithTempURLHandler> cast error")
		}

		cacheObject := sdk.Cache{
			Name:    "cache.tar",
			Project: projectKey,
			Tag:     tag,
		}

		url, key, errO := store.StoreURL(&cacheObject)
		if errO != nil {
			return sdk.WrapError(errO, "postPushCacheWithTempURLHandler>Cannot store cache")
		}
		cacheObject.TmpURL = url
		cacheObject.SecretKey = key

		return service.WriteJSON(w, cacheObject, http.StatusOK)
	}
}

func (api *API) getPullCacheWithTempURLHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		tag := vars["tag"]

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.ErrInvalidName
		}

		store, ok := objectstore.Storage().(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.WrapError(sdk.ErrNotImplemented, "getPullCacheWithTempURLHandler> cast error")
		}

		cacheObject := sdk.Cache{
			Name:    "cache.tar",
			Project: projectKey,
			Tag:     tag,
		}

		url, key, errF := store.FetchURL(&cacheObject)
		if errF != nil {
			return sdk.WrapError(errF, "getPullCacheWithTempURLHandler> Cannot get tmp URL")
		}
		cacheObject.TmpURL = url
		cacheObject.SecretKey = key

		return service.WriteJSON(w, cacheObject, http.StatusOK)
	}
}
