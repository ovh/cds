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
<<<<<<< HEAD
		projectKey := vars[permProjectKey]
=======
>>>>>>> wip
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
			Project: vars["permProjectKey"],
			Tag:     tag,
		}

		storageDriver, err := api.getStorageDriver(vars["permProjectKey"], vars["integrationName"])
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
		vars := mux.Vars(r)
<<<<<<< HEAD
		projectKey := vars[permProjectKey]
=======
>>>>>>> wip
		tag := vars["tag"]

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.ErrInvalidName
		}

		cacheObject := sdk.Cache{
			Project: vars["permProjectKey"],
			Name:    "cache.tar",
			Tag:     tag,
		}

		storageDriver, err := api.getStorageDriver(vars["permProjectKey"], vars["integrationName"])
		if err != nil {
			return err
		}

		s, temporaryURLSupported := storageDriver.(objectstore.DriverWithRedirect)
		if storageDriver.TemporaryURLSupported() && temporaryURLSupported { // with temp URL
			fURL, _, err := s.FetchURL(&cacheObject)
			if err != nil {
				return sdk.WrapError(err, "Cannot fetch cache object")
			}
			w.Header().Add("Content-Type", "application/x-tar")
			w.Header().Add("Content-Disposition", "attachment; filename=\"cache.tar\"")
			http.Redirect(w, r, fURL, http.StatusMovedPermanently)
			return nil
		}

		ioread, err := storageDriver.Fetch(&cacheObject)
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
<<<<<<< HEAD
		projectKey := vars[permProjectKey]
=======
>>>>>>> wip
		tag := vars["tag"]

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.ErrInvalidName
		}

		storageDriver, err := api.getStorageDriver(vars["permProjectKey"], vars["integrationName"])
		if err != nil {
			return err
		}

		store, ok := storageDriver.(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.WrapError(sdk.ErrNotImplemented, "postPushCacheWithTempURLHandler> cast error")
		}

		cacheObject := sdk.Cache{
			Name:    "cache.tar",
			Project: vars["permProjectKey"],
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
		tag := vars["tag"]

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(tag) {
			return sdk.ErrInvalidName
		}

		storageDriver, err := api.getStorageDriver(vars[permProjectKey], vars["integrationName"])
		if err != nil {
			return err
		}

		store, ok := storageDriver.(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.WrapError(sdk.ErrNotImplemented, "getPullCacheWithTempURLHandler> cast error")
		}

		cacheObject := sdk.Cache{
			Name:    "cache.tar",
			Project: vars["permProjectKey"],
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
