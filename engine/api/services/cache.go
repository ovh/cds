package services

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type event struct {
	f func(s sdk.Service)
	s sdk.Service
}

type iCache struct {
	dbConnFactory *database.DBConnectionFactory
	mutex         sync.RWMutex
	data          map[string][]sdk.Service
	chanEvent     chan event
}

var internalCache iCache

// Initialize the service package
func Initialize(c context.Context, dbF *database.DBConnectionFactory, goRoutines *sdk.GoRoutines, panicCallback func(s string) (io.WriteCloser, error)) {
	internalCache = iCache{
		chanEvent:     make(chan event),
		data:          make(map[string][]sdk.Service),
		dbConnFactory: dbF,
		mutex:         sync.RWMutex{},
	}
	goRoutines.Run(c, "service.internalCache.doUpdateData", internalCache.doUpdateData, panicCallback)
	goRoutines.Run(c, "service.internalCache.doListenDatabase", internalCache.doListenDatabase, panicCallback)
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
	if c == nil {
		return nil, false
	}
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
		log.Error(ctx, "Unable to %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			break

		case err := <-chanErr:
			log.Error(ctx, "doListenDatabase> %v", err)
			listener := c.dbConnFactory.NewListener(time.Second, 10*time.Second, eventCallback)
			if err := listener.Listen("events"); err != nil {
				chanErr <- err
			}

		case n := <-listener.Notify:
			e := map[string]interface{}{}
			if err := json.Unmarshal([]byte(n.Extra), &e); err != nil {
				log.Warning(ctx, "unable to unmarshal received event: %v", err)
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

			dataAsObject, ok := data.(map[string]interface{})
			if !ok {
				continue
			}

			name := dataAsObject["name"].(string)
			db := database.DBMap(gorpmapping.Mapper, c.dbConnFactory.DB())

			switch action {
			case "UPDATE", "INSERT":
				srv, err := LoadByName(ctx, db, name)
				if err != nil {
					log.Error(ctx, "unable to find service %s: %v", name, err)
					continue
				}
				c.chanEvent <- event{c.updateCache, *srv}
			case "DELETE":
				c.chanEvent <- event{c.removeFromCache, sdk.Service{
					CanonicalService: sdk.CanonicalService{
						Name: name,
					},
				}}
			}

		case <-time.After(90 * time.Second):
			log.Debug("Received no events for 90 seconds, checking connection")
			go func() {
				listener.Ping() // nolint
			}()
		}
	}
}
