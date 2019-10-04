package sessionstore

import (
	"crypto/rand"
	"fmt"
	"io"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
)

// NewSessionKey generates a random UUID according to RFC 4122
func NewSessionKey() (SessionKey, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return SessionKey(fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])), nil
}

//SessionKey is the session ID
type SessionKey string

//Store is the session store interface
type Store interface {
	New(SessionKey) (SessionKey, error)
	Exists(SessionKey) (bool, error)
	Get(SessionKey, string, interface{}) error
	Set(SessionKey, string, interface{}) error
	Delete(SessionKey) error
}

//Options is a struct to switch from in memory to redis session store
type Options struct {
	Cache cache.Store
	TTL   int
}

type sessionstore struct {
	cache cache.Store
	ttl   int
}

var cacheSessionStore = cache.Key("api:users:session")

func (s *sessionstore) New(session SessionKey) (SessionKey, error) {
	if session == "" {
		var err error
		session, err = NewSessionKey()
		if err != nil {
			return session, err
		}
	}

	k := cache.Key(cacheSessionStore, string(session))
	if err := s.cache.SetWithTTL(k, 1, s.ttl); err != nil {
		log.Error("cannot SetWithTTL: %s: %v", k, err)
	}
	return session, nil
}

func (s *sessionstore) Exists(session SessionKey) (bool, error) {
	k := cache.Key(cacheSessionStore, string(session))
	var sval int
	exist, err := s.cache.Get(k, &sval)
	if err != nil {
		log.Error("cannot get from redis %s: %v", k, err)
	}

	if exist {
		if err := s.cache.SetWithTTL(k, 1, s.ttl); err != nil {
			log.Error("cannot SetWithTTL: %s: %v", k, err)
		}
	}
	return exist, nil
}

func (s *sessionstore) Get(session SessionKey, subkey string, i interface{}) error {
	k := cache.Key(cacheSessionStore, string(session))
	exist, err := s.Exists(session)
	if err != nil {
		return err
	}

	if !exist {
		return fmt.Errorf("session does not exist")
	}

	ks := cache.Key(k, subkey)
	_, err = s.cache.Get(ks, i)
	if err != nil {
		log.Error("cannot get from cache %s: %v", ks, err)
	}
	if err := s.cache.SetWithTTL(ks, i, s.ttl); err != nil {
		log.Error("cannot SetWithTTL: %s: %v", ks, err)
	}
	return nil
}

func (s *sessionstore) Set(session SessionKey, subkey string, i interface{}) error {
	k := cache.Key(cacheSessionStore, string(session))
	exist, err := s.Exists(session)
	if err != nil {
		return err
	}

	if !exist {
		return fmt.Errorf("session does not exist")
	}

	ks := cache.Key(k, subkey)
	if err := s.cache.SetWithTTL(ks, i, s.ttl); err != nil {
		log.Error("cannot SetWithTTL: %s: %v", ks, err)
	}

	return nil
}

func (s *sessionstore) Delete(session SessionKey) error {
	k := cache.Key(cacheSessionStore, string(session))
	ks := cache.Key(cacheSessionStore, string(session), "*")
	if err := s.cache.DeleteAll(k); err != nil {
		log.Error("unable to cache deleteAll %s: %v", k, err)
	}
	if err := s.cache.DeleteAll(ks); err != nil {
		log.Error("unable to cache deleteAll %s: %v", ks, err)
	}
	return nil
}
