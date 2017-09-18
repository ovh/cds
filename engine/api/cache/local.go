package cache

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ovh/cds/sdk/log"
)

//LocalStore is a in memory cache for dev purpose only
type LocalStore struct {
	mutex  *sync.Mutex
	Data   map[string][]byte
	Queues map[string]*list.List
	TTL    int
}

// NewLocalStore returns a new localstore
func NewLocalStore() *LocalStore {
	return &LocalStore{
		mutex:  &sync.Mutex{},
		Data:   map[string][]byte{},
		Queues: map[string]*list.List{},
	}
}

//Get a key from local store
func (s *LocalStore) Get(key string, value interface{}) bool {
	s.mutex.Lock()
	b := s.Data[key]
	s.mutex.Unlock()

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
	s.mutex.Lock()
	s.Data[key] = b
	s.mutex.Unlock()

	if ttl > 0 {
		go func(s *LocalStore, key string) {
			time.Sleep(time.Duration(ttl) * time.Second)
			delete(s.Data, key)
		}(s, key)
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
	for k := range s.Data {
		if key == k || (strings.HasSuffix(key, "*") && strings.HasPrefix(k, key[:len(key)-1])) {
			delete(s.Data, k)
		}
	}
}

//Enqueue pushes to queue
func (s *LocalStore) Enqueue(queueName string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	l := s.Queues[queueName]
	if l == nil {
		s.Queues[queueName] = &list.List{}
		l = s.Queues[queueName]
	}
	b, err := json.Marshal(value)
	if err != nil {
		return
	}

	log.Debug("[%p] Enqueueing to %s :%s", s, queueName, string(b))
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
				s.mutex.Lock()
				l.Remove(e)
				s.mutex.Unlock()
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
	close(elemChan)
	return
}

//QueueLen returns the length of a queue
func (s *LocalStore) QueueLen(queueName string) int {
	l := s.Queues[queueName]
	if l == nil {
		return 0
	}
	return l.Len()
}

//DequeueWithContext gets from queue This is blocking while there is nothing in the queue, it can be cancelled with a context.Context
func (s *LocalStore) DequeueWithContext(c context.Context, queueName string, value interface{}) {
	log.Debug("[%p] DequeueWithContext from %s", s, queueName)
	l := s.Queues[queueName]
	if l == nil {
		s.mutex.Lock()
		s.Queues[queueName] = &list.List{}
		l = s.Queues[queueName]
		s.mutex.Unlock()
	}

	elemChan := make(chan *list.Element)
	var once sync.Once
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond).C
		for {
			select {
			case <-ticker:
				e := l.Back()
				if e != nil {
					s.mutex.Lock()
					l.Remove(e)
					s.mutex.Unlock()
					elemChan <- e
					return
				}
			case <-c.Done():
				once.Do(func() {
					close(elemChan)
				})
				return
			}
		}
	}()

	e := <-elemChan
	if e != nil {
		b, ok := e.Value.([]byte)
		if !ok {
			return
		}
		json.Unmarshal(b, value)
	}

	once.Do(func() {
		close(elemChan)
	})
	return
}

// LocalPubSub local subscriber
type LocalPubSub struct {
	queueName string
}

// Unsubscribe a subscriber
func (s *LocalPubSub) Unsubscribe(channels ...string) error {
	return nil
}

// Publish a msg in a queue
func (s *LocalStore) Publish(channel string, value interface{}) {
	s.mutex.Lock()
	l := s.Queues[channel]
	if l == nil {
		s.Queues[channel] = &list.List{}
		l = s.Queues[channel]
	}
	s.mutex.Unlock()
	b, err := json.Marshal(value)
	if err != nil {
		return
	}
	s.mutex.Lock()
	l.PushBack(b)
	s.mutex.Unlock()
}

// Subscribe to a channel
func (s *LocalStore) Subscribe(channel string) PubSub {
	return &LocalPubSub{
		queueName: channel,
	}
}

// GetMessageFromSubscription from a queue
func (s *LocalStore) GetMessageFromSubscription(c context.Context, pb PubSub) (string, error) {
	lps, ok := pb.(*LocalPubSub)
	if !ok {
		return "", fmt.Errorf("GetMessage> PubSub is not a LocalPubSub. Got %T", pb)
	}
	var msg string
	s.DequeueWithContext(c, lps.queueName, &msg)
	return msg, nil
}

// Status returns the status of the local cache
func (s *LocalStore) Status() string {
	return "OK (local)"
}
