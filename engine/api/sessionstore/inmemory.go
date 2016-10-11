package sessionstore

import (
	"sync"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//InMemory is a local in memory sessionstore, for dev purpose
type InMemory struct {
	lock *sync.Mutex
	data map[SessionKey]cache.Store
	ttl  int
}

//New creates a new session
func (s *InMemory) New(k SessionKey) (SessionKey, error) {
	if k == "" {
		var e error
		k, e = NewSessionKey()
		if e != nil {
			return "", e
		}
	}
	cache := &cache.LocalStore{
		Mutex: &sync.Mutex{},
		Data:  map[string][]byte{},
		TTL:   s.ttl,
	}
	s.lock.Lock()
	s.data[k] = cache
	s.lock.Unlock()

	go func(k SessionKey) {
		time.Sleep(time.Duration(s.ttl) * time.Minute)
		log.Notice("session> delete session %s after %d minutes", k, s.ttl)
		s.lock.Lock()
		delete(s.data, k)
		s.lock.Unlock()
	}(k)

	return k, nil
}

//Exists check if session exists
func (s *InMemory) Exists(key SessionKey) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, exists := s.data[key]
	return exists, nil
}

//Set set a value in session with a key
func (s *InMemory) Set(session SessionKey, k string, data interface{}) error {
	if b, _ := s.Exists(session); !b {
		return sdk.ErrSessionNotFound
	}
	s.data[session].Set(k, data)
	return nil
}

//Get returns the value corresponding to key for the session
func (s *InMemory) Get(session SessionKey, k string, data interface{}) error {
	if b, _ := s.Exists(session); !b {
		return sdk.ErrSessionNotFound
	}
	s.data[session].Get(k, data)
	return nil
}
