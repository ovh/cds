package api

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

type contextKey int

const (
	contextUserAuthentified contextKey = iota
	contextProvider
	contextAPIConsumer
	contextJWT
	contextJWTRaw
	contextScope
	contextWorkflowTemplate
)

// ContextValues retuns auth values of a context
func ContextValues(ctx context.Context) map[interface{}]interface{} {
	return map[interface{}]interface{}{
		//contextHatchery: ctx.Value(contextHatchery),
		//contextService:  ctx.Value(contextService),
		//contextWorker:   ctx.Value(contextWorker),
		//contextUser:     ctx.Value(contextUser),
	}
}

//GetWorker returns the worker instance from its id
func GetWorker(db *gorp.DbMap, Store cache.Store, workerID, workerName string) (*sdk.Worker, error) {
	// Load worker
	var w = &sdk.Worker{}

	key := cache.Key("worker", workerID)
	b := Store.Get(key, w)
	if !b || w.JobRunID == 0 {
		var err error
		w, err = worker.LoadByID(db, workerID)
		if err != nil {
			return nil, fmt.Errorf("cannot load worker '%s': %s", workerName, err)
		}
		Store.Set(key, w)
	}
	return w, nil
}

//GetService returns the service instance from its hash
func GetService(db *gorp.DbMap, Store cache.Store, hash string) (*sdk.Service, error) {
	//Load the service from the cache
	//TODO: this should be embeded in the repository layer
	var srv = &sdk.Service{}
	key := cache.Key("services", hash)
	// Else load it from DB
	if !Store.Get(key, srv) {
		var err error
		srv, err = services.FindByHash(db, hash)
		if err != nil {
			return nil, fmt.Errorf("cannot load service: %s", err)
		}
		srv.Uptodate = srv.Version == sdk.VERSION
		Store.Set(key, srv)
	}
	return srv, nil
}
