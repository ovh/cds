package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// lastUpdateBrokerSubscribe is the information need to to subscribe
type lastUpdateBrokerSubscribe struct {
	UIID  string
	User  *sdk.User
	Queue chan string
}

// lastUpdateBroker keeps connected client of the current route,
type lastUpdateBroker struct {
	clients    map[string]*lastUpdateBrokerSubscribe
	newClients chan *lastUpdateBrokerSubscribe
	messages   chan string
	mutex      *sync.Mutex
}

//Init the lastUpdateBroker
func (b *lastUpdateBroker) Init(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) {
	// Start cache Subscription
	go CacheSubscribe(c, b.messages, store)

	// Start processing events
	go b.Start(c, DBFunc, store)
}

// CacheSubscribe subscribe to a channel and push received message in a channel
func CacheSubscribe(c context.Context, cacheMsgChan chan<- string, store cache.Store) {
	pubSub := store.Subscribe("lastUpdates")
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("lastUpdate.CacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case <-tick.C:
			msg, err := store.GetMessageFromSubscription(c, pubSub)
			if err != nil {
				log.Warning("lastUpdate.CacheSubscribe> Cannot get message %s: %s", msg, err)
				time.Sleep(5 * time.Second)
				continue
			}
			cacheMsgChan <- msg
		}
	}
}

// Start the broker
func (b *lastUpdateBroker) Start(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) {
	for {
		select {
		case <-c.Done():
			// Close all channels
			b.mutex.Lock()
			for c := range b.clients {
				delete(b.clients, c)
			}
			b.mutex.Unlock()
			if c.Err() != nil {
				log.Error("lastUpdate.CacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case s := <-b.newClients:
			// Register new client
			b.mutex.Lock()
			b.clients[s.UIID] = s
			b.mutex.Unlock()
		case msg := <-b.messages:
			var lastModif sdk.LastModification
			if err := json.Unmarshal([]byte(msg), &lastModif); err != nil {
				log.Warning("lastUpdate.CacheSubscribe> Cannot unmarshal message: %s", msg)
				continue
			}

			//Receive new message
			b.mutex.Lock()
			for _, i := range b.clients {
				if err := loadUserPermissions(DBFunc(), store, i.User); err != nil {
					log.Warning("lastUpdate.CacheSubscribe> Cannot load user permission: %s", err)
					continue
				}

				if i.User.Admin {
					i.Queue <- msg
					continue
				}

				switch lastModif.Type {
				case sdk.ProjectLastModificationType:
					if permission.ProjectPermission(lastModif.Key, i.User) >= permission.PermissionRead {
						i.Queue <- msg
						continue
					}
				case sdk.ApplicationLastModificationType:
					if permission.ApplicationPermission(lastModif.Key, lastModif.Name, i.User) >= permission.PermissionRead {
						i.Queue <- msg
						continue
					}
				case sdk.PipelineLastModificationType:
					if permission.PipelinePermission(lastModif.Key, lastModif.Name, i.User) >= permission.PermissionRead {
						i.Queue <- msg
						continue
					}
				}

			}
			b.mutex.Unlock()
		}
	}
}

func (b *lastUpdateBroker) ServeHTTP() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return nil
		}

		uuid, errS := sessionstore.NewSessionKey()
		if errS != nil {
			return sdk.WrapError(errS, "lastUpdateBroker.Serve> Cannot generate UUID")
		}
		messageChan := &lastUpdateBrokerSubscribe{
			UIID:  string(uuid),
			User:  getUser(ctx),
			Queue: make(chan string),
		}

		// Add this client to the map of those that should receive updates
		b.newClients <- messageChan

		// Set the headers related to event streaming.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		fmt.Fprint(w, "data: ACK\n\n")
		f.Flush()

	leave:
		for {
			select {
			case <-w.(http.CloseNotifier).CloseNotify():
				b.mutex.Lock()
				delete(b.clients, messageChan.UIID)
				b.mutex.Unlock()
				break leave
			case msg := <-messageChan.Queue:
				fmt.Fprintf(w, "data: %s\n\n", msg)
				f.Flush()
			}
		}
		return nil
	}
}
