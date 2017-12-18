package sessionstore

import (
	"context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Status for session store
var Status sdk.MonitoringStatusLine

//Get is a factory
func Get(c context.Context, redisHost, redisPassword string, ttl int) (Store, error) {
	r, err := NewRedis(c, redisHost, redisPassword, ttl)
	if err != nil {
		log.Error("sessionstore.factory> unable to connect to redis %s : %s", redisHost, err)
		Status = sdk.MonitoringStatusLine{Component: "Sessions-Store", Value: "KO", Status: sdk.MonitoringStatusAlert}
	} else {
		Status = sdk.MonitoringStatusLine{Component: "Sessions-Store", Value: "OK", Status: sdk.MonitoringStatusOK}
	}
	return r, err
}
