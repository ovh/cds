package api

import (
	"bytes"
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

// eventsBrokerSubscribe is the information needed to subscribe
type eventsBrokerSubscribe struct {
	UUID  string
	User  *sdk.User
	Queue chan sdk.Event
}

// lastUpdateBroker keeps connected client of the current route,
type eventsBroker struct {
	clients           map[string]eventsBrokerSubscribe
	messages          chan sdk.Event
	mutex             *sync.Mutex
	disconnected      map[string]bool
	disconnectedMutex *sync.Mutex
	dbFunc            func() *gorp.DbMap
	cache             cache.Store
}

// AddClient add a client to the client map
func (b *eventsBroker) addClient(client eventsBrokerSubscribe) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.clients[client.UUID] = client
}

// CleanAll cleans all clients
func (b *eventsBroker) cleanAll() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.clients != nil {
		for c, v := range b.clients {
			close(v.Queue)
			delete(b.clients, c)
		}
	}
}

func (b *eventsBroker) disconnectClient(uuid string) {
	b.disconnectedMutex.Lock()
	defer b.disconnectedMutex.Unlock()
	b.disconnected[uuid] = true
}

//Init the eventsBroker
func (b *eventsBroker) Init(c context.Context) {
	// Start cache Subscription
	subscribeFunc := func() {
		cacheSubscribe(c, b.messages, b.cache)
	}
	sdk.GoRoutine("eventsBroker.Init.CacheSubscribe", subscribeFunc)

	startFunc := func() {
		b.Start(c)
	}
	sdk.GoRoutine("eventsBroker.Init.Start", startFunc)
}

func cacheSubscribe(c context.Context, cacheMsgChan chan<- sdk.Event, store cache.Store) {
	pubSub := store.Subscribe("events_pubsub")
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("events.cacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case <-tick.C:
			msg, err := store.GetMessageFromSubscription(c, pubSub)
			if err != nil {
				log.Warning("events.cacheSubscribe> Cannot get message %s: %s", msg, err)
				continue
			}
			var e sdk.Event
			if err := json.Unmarshal([]byte(msg), &e); err != nil {
				log.Warning("events.cacheSubscribe> Cannot unmarshal event %s: %s", msg, err)
				continue
			}

			switch e.EventType {
			case "sdk.EventPipelineBuild", "sdk.EventJob":
				continue
			}
			cacheMsgChan <- e
		}
	}
}

// Start the broker
func (b *eventsBroker) Start(c context.Context) {
	for {
		select {
		case <-c.Done():
			b.cleanAll()
			if c.Err() != nil {
				log.Error("eventsBroker.Start> Exiting: %v", c.Err())
				return
			}
		case receivedEvent := <-b.messages:
			b.manageEvent(receivedEvent)
		}
	}
}

func (b *eventsBroker) ServeHTTP() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)
		if !ok {
			return sdk.WrapError(fmt.Errorf("streaming unsupported"), "")
		}

		uuidSK, errS := sessionstore.NewSessionKey()
		if errS != nil {
			return sdk.WrapError(errS, "eventsBroker.Serve> Cannot generate UUID")
		}
		uuid := string(uuidSK)
		user := getUser(ctx)
		if err := loadUserPermissions(b.dbFunc(), b.cache, user); err != nil {
			return sdk.WrapError(err, "eventsBroker.Serve Cannot load user permission")
		}

		client := eventsBrokerSubscribe{
			UUID:  uuid,
			User:  user,
			Queue: make(chan sdk.Event, 10), // chan buffered, to avoid goroutine Start() wait on push in queue
		}

		// Add this client to the map of those that should receive updates
		b.addClient(client)

		// Set the headers related to event streaming.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		if _, err := w.Write([]byte(fmt.Sprintf("data: ACK: %s \n\n", uuid))); err != nil {
			return sdk.WrapError(err, "events.write> Unable to send ACK to client")
		}
		f.Flush()

		tick := time.NewTicker(time.Second)
		defer tick.Stop()

	leave:
		for {
			select {
			case <-ctx.Done():
				log.Info("events.Http: context done")
				b.disconnectClient(client.UUID)
				break leave
			case <-r.Context().Done():
				log.Info("events.Http: client disconnected")
				b.disconnectClient(client.UUID)
				break leave
			case event := <-client.Queue:
				if ok := client.manageEvent(event); !ok {
					continue
				}

				msg, errJ := json.Marshal(event)
				if errJ != nil {
					log.Warning("sendevent> Unavble to marshall event: %v", errJ)
					continue
				}

				var buffer bytes.Buffer
				buffer.WriteString("data: ")
				buffer.Write(msg)
				buffer.WriteString("\n\n")

				if _, err := w.Write(buffer.Bytes()); err != nil {
					return sdk.WrapError(err, "events.write> Unable to write to client")
				}
				f.Flush()
			case <-tick.C:
				if _, err := w.Write([]byte("")); err != nil {
					return sdk.WrapError(err, "events.write> Unable to ping client")
				}
				f.Flush()
			}
		}

		return nil
	}
}

func (b *eventsBroker) manageEvent(receivedEvent sdk.Event) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, i := range b.clients {
		if b.canSend(i) {
			i.Queue <- receivedEvent
		}
	}
}

// canSend Test if client is connected. If not, close channel and remove client from map
func (b *eventsBroker) canSend(client eventsBrokerSubscribe) bool {
	b.disconnectedMutex.Lock()
	defer b.disconnectedMutex.Unlock()
	if _, ok := b.disconnected[client.UUID]; !ok {
		return true
	}
	close(client.Queue)
	delete(b.clients, client.UUID)
	return false
}

func (s *eventsBrokerSubscribe) manageEvent(event sdk.Event) bool {
	if strings.HasPrefix(event.EventType, "sdk.EventProject") {
		if s.User.Admin || permission.ProjectPermission(event.ProjectKey, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventWorkflow") || strings.HasPrefix(event.EventType, "sdk.EventRunWorkflow") {
		if s.User.Admin || permission.WorkflowPermission(event.ProjectKey, event.WorkflowName, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventApplication") {
		if s.User.Admin || permission.ApplicationPermission(event.ProjectKey, event.ApplicationName, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventPipeline") {
		if s.User.Admin || permission.PipelinePermission(event.ProjectKey, event.PipelineName, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventEnvironment") {
		if s.User.Admin || permission.EnvironmentPermission(event.ProjectKey, event.EnvironmentName, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventBroadcast") {
		if s.User.Admin || event.ProjectKey == "" || permission.AccessToProject(event.ProjectKey, s.User, permission.PermissionRead) {
			return true
		}
		return false
	}
	return false
}
