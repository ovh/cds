package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	clients  map[string]lastUpdateBrokerSubscribe
	messages chan string
	mutex    *sync.Mutex
	dbFunc   func() *gorp.DbMap
	cache    cache.Store
}

//Init the lastUpdateBroker
func (b *lastUpdateBroker) Init(c context.Context) {
	// Start cache Subscription
	go CacheSubscribe(c, b.messages, b.cache)

	// Start processing events
	go b.Start(c)
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

func (b *lastUpdateBroker) UpdateUserPermissions(username string) {
	var user *sdk.User

	// get the user
	b.mutex.Lock()
	for _, c := range b.clients {
		if c.User.Username == username {
			user = c.User
			break
		}
	}
	b.mutex.Unlock()

	if user == nil {
		return
	}
	// load permission without being in the mutex lock
	if err := loadUserPermissions(b.dbFunc(), b.cache, user); err != nil {
		log.Error("lastUpdate.UpdateUserPermissions> Cannot load user permission:%s", err)
	}

	// then, relock map and update user
	b.mutex.Lock()
	for _, c := range b.clients {
		if c.User.Username == username {
			c.User = user
		}
	}
	b.mutex.Unlock()
}

// Start the broker
func (b *lastUpdateBroker) Start(c context.Context) {
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
		case msg := <-b.messages:
			var lastModif sdk.LastModification
			if err := json.Unmarshal([]byte(msg), &lastModif); err != nil {
				log.Warning("lastUpdate.CacheSubscribe> Cannot unmarshal message: %s", msg)
				continue
			}

			//Receive new message
			b.mutex.Lock()
			for _, i := range b.clients {
				if i.User.Admin {
					i.Queue <- msg
					continue
				}
				switch strings.Split(lastModif.Type, ".")[0] {
				case "project":
					if permission.ProjectPermission(lastModif.Key, i.User) >= permission.PermissionRead {
						i.Queue <- msg
						continue
					}
				case "application":
					if permission.ApplicationPermission(lastModif.Key, lastModif.Name, i.User) >= permission.PermissionRead {
						i.Queue <- msg
						continue
					}
				case "pipeline":
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

		user := getUser(ctx)
		if err := loadUserPermissions(b.dbFunc(), b.cache, user); err != nil {
			return sdk.WrapError(err, "lastUpdate.CacheSubscribe> Cannot load user permission")
		}

		messageChan := lastUpdateBrokerSubscribe{
			UIID:  string(uuid),
			User:  user,
			Queue: make(chan string, 10), // chan buffered, to avoid goroutine Start() wait on push in queue
		}

		// Add this client to the map of those that should receive updates
		b.mutex.Lock()
		b.clients[string(uuid)] = messageChan
		b.mutex.Unlock()

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
			case <-ctx.Done():
				b.mutex.Lock()
				delete(b.clients, messageChan.UIID)
				b.mutex.Unlock()
				break leave
			case <-w.(http.CloseNotifier).CloseNotify():
				b.mutex.Lock()
				delete(b.clients, messageChan.UIID)
				b.mutex.Unlock()
				break leave
			case msg := <-messageChan.Queue:
				w.Write([]byte("data: "))
				w.Write([]byte(msg))
				w.Write([]byte("\n\n"))
				f.Flush()
			default:
				f.Flush()
			}
		}
		return nil
	}
}
