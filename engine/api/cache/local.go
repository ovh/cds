package cache

import (
	"container/list"
	"encoding/json"
	"sync"
	"time"

	"github.com/ovh/cds/engine/log"
)

var s Store

//LocalStore is a in memory cache for dev purpose only
type LocalStore struct {
	Mutex  *sync.Mutex
	Data   map[string][]byte
	Queues map[string]*list.List
	TTL    int
}

//Get a key from local store
func (s *LocalStore) Get(key string, value interface{}) {
	s.Mutex.Lock()
	b := s.Data[key]
	s.Mutex.Unlock()
	if b != nil && len(b) > 0 {
		if err := json.Unmarshal(b, value); err != nil {
			log.Warning("Cache> Cannot unmarshal %s :%s", key, err)
		}
	}
}

//SetWithTTL a value in local store with a specific ttl (in seconds): (0 for eternity)
func (s *LocalStore) SetWithTTL(key string, value interface{}, ttl int) {
	b, err := json.Marshal(value)
	if err != nil {
		log.Warning("Error caching %s", key)
	}
	s.Mutex.Lock()
	s.Data[key] = b
	s.Mutex.Unlock()

	if ttl > 0 {
		go func(key string) {
			time.Sleep(time.Duration(ttl) * time.Second)
			log.Debug("Cache> Delete %s from cache after %d seconds", key, s.TTL)
			s.Mutex.Lock()
			delete(s.Data, key)
			s.Mutex.Unlock()
		}(key)
	}
}

//Set a value in local store
func (s *LocalStore) Set(key string, value interface{}) {
	s.SetWithTTL(key, value, s.TTL)
}

//Delete a key from local store
func (s *LocalStore) Delete(key string) {
	delete(s.Data, key)
}

//DeleteAll on locastore delete all the things
func (s *LocalStore) DeleteAll(key string) {
	s.Data = map[string][]byte{}
}

//Enqueue pushes to queue
func (s *LocalStore) Enqueue(queueName string, value interface{}) {
	l := s.Queues[queueName]
	if l == nil {
		s.Queues[queueName] = &list.List{}
		l = s.Queues[queueName]
	}
	b, err := json.Marshal(value)
	if err != nil {
		return
	}
	l.PushFront(b)
}

//Dequeue gets from queue This is blocking while there is nothing in the queue
func (s *LocalStore) Dequeue(queueName string, value interface{}) {
	l := s.Queues[queueName]
	if l == nil {
		s.Queues[queueName] = &list.List{}
		l = s.Queues[queueName]
	}

	elemChan := make(chan *list.Element)
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			e := l.Back()
			if e != nil {
				elemChan <- e
				return
			}
		}
	}()

	e := <-elemChan
	b, ok := e.Value.([]byte)
	if !ok {
		return
	}
	json.Unmarshal(b, value)
	return
}
