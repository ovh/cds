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
		if _, isWorker := api.isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)
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
			Project: vars[permProjectKey],
			Tag:     tag,
		}

		storageDriver, err := objectstore.GetDriver(ctx, api.mustDB(), api.SharedStorage, vars[permProjectKey], vars["integrationName"])
		if err != nil {
			return err
		}

		if _, err := storageDriver.Store(&cacheObject, r.Body); err != nil {
			return sdk.WrapError(err, "postPushCacheHandler>Cannot store cache")
		}

		return nil
	}
}

func (api *API) getPullCacheHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if _, isWorker := api.isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)
		tag := vars["tag"]

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.ErrInvalidName
		}

		cacheObject := sdk.Cache{
			Project: vars[permProjectKey],
			Name:    "cache.tar",
			Tag:     tag,
		}

		storageDriver, err := objectstore.GetDriver(ctx, api.mustDB(), api.SharedStorage, vars[permProjectKey], vars["integrationName"])
		if err != nil {
			return err
		}

		s, temporaryURLSupported := storageDriver.(objectstore.DriverWithRedirect)
		if storageDriver.TemporaryURLSupported() && temporaryURLSupported { // with temp URL
			fURL, _, err := s.FetchURL(&cacheObject)
			if err != nil {
				return sdk.WrapError(err, "cannot fetch cache object")
			}
			w.Header().Add("Content-Type", "application/x-tar")
			w.Header().Add("Content-Disposition", "attachment; filename=\"cache.tar\"")
			http.Redirect(w, r, fURL, http.StatusMovedPermanently)
			return nil
		}

		ioread, err := storageDriver.Fetch(ctx, &cacheObject)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNotFound, "cannot fetch artifact cache.tar"))
		}
		if _, err := io.Copy(w, ioread); err != nil {
			_ = ioread.Close()
			return sdk.WrapError(err, "cannot stream artifact")
		}

		if err := ioread.Close(); err != nil {
			return sdk.WrapError(err, "cannot close artifact")
		}
		return nil
	}
}

func (api *API) postPushCacheWithTempURLHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if _, isWorker := api.isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)
		tag := vars["tag"]

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.WithStack(sdk.ErrInvalidName)
		}

		storageDriver, err := objectstore.GetDriver(ctx, api.mustDB(), api.SharedStorage, vars[permProjectKey], vars["integrationName"])
		if err != nil {
			return err
		}

		store, ok := storageDriver.(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.WrapError(sdk.ErrNotImplemented, "cast error")
		}

		cacheObject := sdk.Cache{
			Name:    "cache.tar",
			Project: vars[permProjectKey],
			Tag:     tag,
		}

		url, key, err := store.StoreURL(&cacheObject, "application/tar")
		if err != nil {
			return sdk.WrapError(err, "cannot store cache")
		}
		cacheObject.TmpURL = url
		cacheObject.SecretKey = key

		return service.WriteJSON(w, cacheObject, http.StatusOK)
	}
}

func (api *API) getPullCacheWithTempURLHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if _, isWorker := api.isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)
		tag := vars["tag"]

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.WithStack(sdk.ErrInvalidName)
		}

		storageDriver, err := objectstore.GetDriver(ctx, api.mustDB(), api.SharedStorage, vars[permProjectKey], vars["integrationName"])
		if err != nil {
			return err
		}

		store, ok := storageDriver.(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.WrapError(sdk.ErrNotImplemented, "cast error")
		}

		cacheObject := sdk.Cache{
			Name:    "cache.tar",
			Project: vars[permProjectKey],
			Tag:     tag,
		}

		url, key, err := store.FetchURL(&cacheObject)
		if err != nil {
			return sdk.WrapError(err, "cannot get tmp URL")
		}
		cacheObject.TmpURL = url
		cacheObject.SecretKey = key

		return service.WriteJSON(w, cacheObject, http.StatusOK)
	}
}
