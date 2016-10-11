package sessionstore

import (
	"sync"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/log"
)

//Status for session store
var Status string

//Get is a factory
func Get(mode, redisHost, redisPassword string, ttl int) (Store, error) {
	log.Notice("SessionStore> Intializing store (%s)\n", mode)
	switch mode {
	case "redis":
		Status = "Redis "
		r, err := NewRedis(redisHost, redisPassword, ttl)
		if err != nil {
			log.Critical("sessionstore.factory> unable to connect to redis %s : %s", redisHost, err)
			Status += "KO"
		} else {
			Status = "OK"
		}
		return r, err
	default:
		Status = "In Memory"
		return &InMemory{
			lock: &sync.Mutex{},
			data: map[SessionKey]cache.Store{},
			ttl:  ttl * 60,
		}, nil
	}
}
