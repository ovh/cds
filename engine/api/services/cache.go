package services

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type event struct {
	f func(s sdk.Service)
	s sdk.Service
}

type eventFunc func(s sdk.Service)

type iCache struct {
	dbConnFactory *database.DBConnectionFactory
	mutex         *sync.RWMutex
	data          map[string][]sdk.Service
	chanEvent     chan event
}

var internalCache iCache

// Initialize the service package
func Initialize(c context.Context, dbF *database.DBConnectionFactory, panicCallback func(s string) (io.WriteCloser, error)) {
	internalCache = iCache{
		chanEvent:     make(chan event),
		data:          make(map[string][]sdk.Service),
		dbConnFactory: dbF,
		mutex:         new(sync.RWMutex),
	}
	sdk.GoRoutine(c, "service.internalCache.doUpdateData", internalCache.doUpdateData, panicCallback)
	sdk.GoRoutine(c, "service.internalCache.doListenDatabase", internalCache.doListenDatabase, panicCallback)
}

func (c *iCache) updateCache(s sdk.Service) {
	ss, ok := c.data[s.Type]
	indexToUpdate := -1
	if !ok {
		ss = make([]sdk.Service, 0, 1)
	} else {
		for i, sub := range ss {
			if sub.Name == s.Name {
				indexToUpdate = i
				break
			}
		}
	}
	if indexToUpdate == -1 {
		ss = append(ss, s)
	} else {
		ss[indexToUpdate] = s
	}
	c.data[s.Type] = ss
}

func (c *iCache) removeFromCache(s sdk.Service) {
	ss, ok := c.data[s.Type]
	if !ok || len(ss) == 0 {
		return
	}
	indexToSplit := 0
	for i, sub := range ss {
		if sub.Name == s.Name {
			indexToSplit = i
			break
		}
	}
	ss = append(ss[:indexToSplit], ss[indexToSplit+1:]...)
	c.data[s.Type] = ss
}

func (c *iCache) getFromCache(s string) ([]sdk.Service, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	ss, ok := c.data[s]
	return ss, ok
}

func (c *iCache) doUpdateData(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			break
		case e, has := <-c.chanEvent:
			log.Debug("service.doUpdateData> updating service cache")
			if e.f != nil {
				c.mutex.Lock()
				e.f(e.s)
				c.mutex.Unlock()
			}
			if !has {
				break
			}
		}
	}
}

func (c *iCache) doListenDatabase(ctx context.Context) {

	chanErr := make(chan error)
	eventCallback := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			chanErr <- err
		}
	}

	listener := c.dbConnFactory.NewListener(time.Second, 10*time.Second, eventCallback)
	if err := listener.Listen("events"); err != nil {
		log.Error("Unable to ", err)
	}

	for {
		select {
		case <-ctx.Done():
			break

		case err := <-chanErr:
			log.Error("doListenDatabase> %v", err)
			listener := c.dbConnFactory.NewListener(time.Second, 10*time.Second, eventCallback)
			if err := listener.Listen("events"); err != nil {
				chanErr <- err
			}

		case n := <-listener.Notify:
			log.Debug("Received data from channel [%s]", n.Channel)
			// Prepare notification payload for pretty print

			e := map[string]interface{}{}
			if err := json.Unmarshal([]byte(n.Extra), &e); err != nil {
				log.Warning("unable to unmarshal received event: %v", err)
				continue
			}

			iTableName, has := e["table"]
			if !has {
				continue
			}
			table, ok := iTableName.(string)
			if !ok || table != "services" {
				continue
			}

			iAction, has := e["action"]
			if !has {
				continue
			}
			action, ok := iAction.(string)
			if !ok {
				continue
			}

			data, has := e["data"]
			if !has {
				continue
			}

			jsonData, _ := json.Marshal(data) // this cannot append
			var srv sdk.Service
			if err := json.Unmarshal(jsonData, &srv); err != nil {
				log.Warning("unable to unmarshal received event data: %v", err)
				continue
			}

			log.Debug("handling %s on (%d)%s", action, srv.ID, srv.Name)

			switch action {
			case "UPDATE", "INSERT":
				c.chanEvent <- event{c.updateCache, srv}
			case "DELETE":
				c.chanEvent <- event{c.removeFromCache, srv}
			}

		case <-time.After(90 * time.Second):
			log.Debug("Received no events for 90 seconds, checking connection")
			go func() {
				listener.Ping()
			}()
		}
	}
}
