package sessionstore

import (
	"context"

	"github.com/ovh/cds/sdk/log"
)

//Status for session store
var Status string

//Get is a factory
func Get(c context.Context, redisHost, redisPassword string, ttl int) (Store, error) {
	r, err := NewRedis(c, redisHost, redisPassword, ttl)
	if err != nil {
		log.Error("sessionstore.factory> unable to connect to redis %s : %s", redisHost, err)
		Status += "KO"
	} else {
		Status = "OK"
	}
	return r, err
}
