package sessionstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Redis is a redis client
type Redis struct {
	ttl       int
	store     *cache.RedisStore
	nbSession int
}

// Status return status of the session store
func (s *Redis) Status() (string, string, bool, error) {
	_, _, ok, err := s.store.Status()
	return "Redis", fmt.Sprintf("%d sessions", s.nbSession), ok, err
}

//Keep redis in good health and remove HSet for expired session
func (s *Redis) vacuumCleaner(c context.Context) {
	tick := time.NewTicker(5 * time.Minute).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting sessionstore.vacuumCleaner: %v", c.Err())
				return
			}
		case <-tick:
			keys, err := s.store.Client.Keys("session:*:data").Result()
			s.nbSession = len(keys)
			if err != nil {
				log.Error("RedisSessionStore> Unable to get keys in store : %s", err)
			}
			for _, k := range keys {
				sessionKey := strings.Replace(k, ":data", "", -1)
				sessionExist, err := s.store.Client.Exists(sessionKey).Result()
				if err != nil {
					log.Warning("RedisSessionStore> Unable to get key %s from store : %s", sessionKey, err)
				}
				if sessionExist == 0 {
					if err := s.store.Client.Del(k).Err(); err != nil {
						log.Error("RedisSessionStore> Unable to clear session %s from store : %s", sessionKey, err)
					}
				}
			}
		}
	}
}

//NewRedis creates a ready to use redisstore
func NewRedis(c context.Context, redisHost, redisPassword string, ttl int) (*Redis, error) {
	r, err := cache.NewRedisStore(redisHost, redisPassword, ttl*60)
	if err != nil {
		return nil, err
	}
	log.Info("Redis> Store ready")
	redisStore := &Redis{ttl * 1440, r, 0}
	go redisStore.vacuumCleaner(c)
	return redisStore, nil
}

//New creates a new session
func (s *Redis) New(k SessionKey) (SessionKey, error) {
	var token SessionKey
	var err error
	if k != "" {
		token = k
	} else {
		token, err = NewSessionKey()
	}

	if err != nil {
		log.Error("Redis> unable to generate session key : %s", err)
		return "", err
	}
	key := cache.Key("session", string(token))
	//Store in redis
	if err := s.store.Client.Set(key, 1, time.Duration(s.ttl)*time.Minute).Err(); err != nil {
		log.Error("Redis> unable create redis session %s : %s", key, err)
		return "", err
	}
	return token, nil
}

//Exists check if session exists
func (s *Redis) Exists(token SessionKey) (bool, error) {
	key := cache.Key("session", string(token))
	exists, err := s.store.Client.Exists(key).Result()
	if err != nil {
		log.Warning("Redis> unable check session exist %s : %s", key, err)
		return false, err
	}
	if exists == 1 {
		if err := s.store.Client.Expire(key, time.Duration(s.ttl)*time.Minute).Err(); err != nil {
			log.Warning("Redis> unable to update session expire %s : %s", key, err)
		}
	} else {
		log.Debug("Session %s invalid", key)
	}

	return exists == 1, nil
}

//Set set a value in session with a key
func (s *Redis) Set(token SessionKey, f string, data interface{}) error {
	if b, _ := s.Exists(token); !b {
		return sdk.ErrSessionNotFound
	}
	key := cache.Key("session", string(token), "data")

	b, err := json.Marshal(data)
	if err != nil {
		log.Warning("Redis> error marshal %s %s", key, f)
	}

	if err := s.store.Client.HSet(key, f, string(b)).Err(); err != nil {
		log.Warning("Redis> unable create redis session %s : %s", key, err)
		return err
	}
	return nil
}

//Get returns the value corresponding to key for the session
func (s *Redis) Get(token SessionKey, f string, data interface{}) error {
	if b, _ := s.Exists(token); !b {
		return sdk.ErrSessionNotFound
	}

	key := cache.Key("session", string(token), "data")
	sval, err := s.store.Client.HGet(key, f).Result()
	if err != nil {
		log.Warning("Redis> unable to get %s %s", key, f)
		return err
	}

	if sval != "" {
		if err := json.Unmarshal([]byte(sval), data); err != nil {
			log.Warning("Redis> Cannot unmarshal %s :%s", key, err)
			return err
		}
	}

	return nil
}

//Delete delete a session
func (s *Redis) Delete(token SessionKey) error {
	key := cache.Key("session", string(token))
	if err := s.store.Client.Del(key).Err(); err != nil {
		return err
	}
	keyData := cache.Key("session", string(token), "data")
	if err := s.store.Client.Del(keyData).Err(); err != nil {
		return err
	}
	return nil
}
