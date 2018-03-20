package sessionstore

import (
	"context"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

//Status for session store
var Status sdk.MonitoringStatusLine

//Get is a factory
func Get(c context.Context, s cache.Store, ttl int) (Store, error) {
	Status = sdk.MonitoringStatusLine{Component: "Sessions-Store", Value: "OK", Status: sdk.MonitoringStatusOK}

	return &sessionstore{
		cache: s,
		ttl:   ttl,
	}, nil
}
