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
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	EventLastUpdate = iota
)

// eventsBrokerSubscribe is the information needed to subscribe
type eventsBrokerSubscribe struct {
	UIID   string
	User   *sdk.User
	Events map[int]bool
	Queue  chan string
}

// lastUpdateBroker keeps connected client of the current route,
type eventsBroker struct {
	clients  map[string]eventsBrokerSubscribe
	messages chan sdk.Event
	mutex    *sync.Mutex
	dbFunc   func() *gorp.DbMap
	cache    cache.Store
}

//Init the eventsBroker
func (b *eventsBroker) Init(c context.Context) {
	// Start cache Subscription
	event.Subscribe(b.messages)

	// Start processing events
	go b.Start(c)
}

func (b *eventsBroker) UpdateUserPermissions(username string) {
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
		log.Error("eventsBroker.UpdateUserPermissions> Cannot load user permission:%s", err)
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
func (b *eventsBroker) Start(c context.Context) {
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
				log.Error("eventsBroker.CacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case receivedEvent := <-b.messages:
			bEvent, err := json.Marshal(receivedEvent)
			if err != nil {
				continue
			}

			b.mutex.Lock()
			for _, i := range b.clients {
				if i.User.Admin {
					i.Queue <- string(bEvent)
					continue
				}
				manageEvent(receivedEvent, string(bEvent), i)
			}

			b.mutex.Unlock()
		}
	}
}

func (b *eventsBroker) ServeHTTP() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return nil
		}

		uuid, errS := sessionstore.NewSessionKey()
		if errS != nil {
			return sdk.WrapError(errS, "eventsBroker.Serve> Cannot generate UUID")
		}

		user := getUser(ctx)
		if err := loadUserPermissions(b.dbFunc(), b.cache, user); err != nil {
			return sdk.WrapError(err, "eventsBroker.Serve Cannot load user permission")
		}

		messageChan := eventsBrokerSubscribe{
			UIID:   string(uuid),
			User:   user,
			Events: make(map[int]bool),
			Queue:  make(chan string, 10), // chan buffered, to avoid goroutine Start() wait on push in queue
		}
		messageChan.Events[EventLastUpdate] = true

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

		tick := time.NewTicker(time.Second)
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
			case <-tick.C:
				f.Flush()
			}
		}
		return nil
	}
}

func manageEvent(event sdk.Event, eventS string, subscriber eventsBrokerSubscribe) {
	if strings.HasPrefix(event.EventType, "EventProject") {
		if permission.ProjectPermission(event.ProjectKey, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
			return
		}
	}
	if strings.HasPrefix(event.EventType, "EventWorkflow") {
		if permission.WorkflowPermission(event.ProjectKey, event.WorkflowName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
			return
		}
	}
	if strings.HasPrefix(event.EventType, "EventApplication") {
		if permission.ApplicationPermission(event.ProjectKey, event.ApplicationName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
			return
		}
	}
	if strings.HasPrefix(event.EventType, "EventPipeline") {
		if permission.PipelinePermission(event.ProjectKey, event.PipelineName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
			return
		}
	}
	if strings.HasPrefix(event.EventType, "EventEnvironment") {
		if permission.EnvironmentPermission(event.ProjectKey, event.EnvironmentName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
			return
		}
	}
}
