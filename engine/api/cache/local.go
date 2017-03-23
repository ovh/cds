package cache

import (
	"container/list"
	"encoding/json"
	"strings"
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
func (s *LocalStore) Get(key string, value interface{}) bool {
	s.Mutex.Lock()
	b := s.Data[key]
	s.Mutex.Unlock()
	if b != nil && len(b) > 0 {
		if err := json.Unmarshal(b, value); err != nil {
			log.Warning("Cache> Cannot unmarshal %s :%s", key, err)
			return false
		}
		return true
	}
	return false
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
		go func(s *LocalStore, key string) {
			time.Sleep(time.Duration(ttl) * time.Second)
			s.Mutex.Lock()
			delete(s.Data, key)
			s.Mutex.Unlock()
		}(s, key)
	}
}

//Set a value in local store
func (s *LocalStore) Set(key string, value interface{}) {
	s.SetWithTTL(key, value, s.TTL)
}

//Delete a key from local store
func (s *LocalStore) Delete(key string) {
	s.Mutex.Lock()
	delete(s.Data, key)
	s.Mutex.Unlock()
}

//DeleteAll on locastore delete all the things
func (s *LocalStore) DeleteAll(key string) {
	for k := range s.Data {
		if key == k || (strings.HasSuffix(key, "*") && strings.HasPrefix(k, key[:len(key)-1])) {
			s.Mutex.Lock()
			delete(s.Data, k)
			s.Mutex.Unlock()
		}
	}
}

//Enqueue pushes to queue
func (s *LocalStore) Enqueue(queueName string, value interface{}) {
	s.Mutex.Lock()
	l := s.Queues[queueName]
	if l == nil {
		s.Queues[queueName] = &list.List{}
		l = s.Queues[queueName]
	}
	s.Mutex.Unlock()
	b, err := json.Marshal(value)
	if err != nil {
		return
	}
	s.Mutex.Lock()
	l.PushFront(b)
	s.Mutex.Unlock()
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
				s.Mutex.Lock()
				l.Remove(e)
				s.Mutex.Unlock()
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
