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
	Queue chan string
}

// lastUpdateBroker keeps connected client of the current route,
type eventsBroker struct {
	clients  map[string]eventsBrokerSubscribe
	messages chan sdk.Event
	mutex    *sync.Mutex
	dbFunc   func() *gorp.DbMap
	cache    cache.Store
}

// AddClient add a client to the client map
func (b *eventsBroker) addClient(uuid string, messageChan eventsBrokerSubscribe) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.clients[uuid] = messageChan
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

// CleanClient cleans a client
func (b *eventsBroker) cleanClient(client eventsBrokerSubscribe) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Close channel
	close(client.Queue)
	// Delete client from map
	delete(b.clients, client.UUID)
}

func (b *eventsBroker) setUser(user *sdk.User) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, c := range b.clients {
		if c.User.Username == user.Username {
			c.User = user
			break
		}
	}
}

func (b *eventsBroker) getUser(username string) *sdk.User {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, c := range b.clients {
		if c.User.Username == username {
			return c.User
		}
	}
	return nil
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

func (b *eventsBroker) UpdateUserPermissions(username string) {
	var user *sdk.User

	user = b.getUser(username)

	if user == nil {
		return
	}
	// load permission without being in the mutex lock
	if err := loadUserPermissions(b.dbFunc(), b.cache, user); err != nil {
		log.Error("eventsBroker.UpdateUserPermissions> Cannot load user permission:%s", err)
	}

	// then, relock map and update user
	b.setUser(user)

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
			bEvent, err := json.Marshal(receivedEvent)
			if err != nil {
				log.Warning("eventsBroker.Start> Unable to marshal event: %+v", receivedEvent)
				continue
			}
			b.manageEvent(receivedEvent, string(bEvent))
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

		messageChan := eventsBrokerSubscribe{
			UUID:  uuid,
			User:  user,
			Queue: make(chan string, 10), // chan buffered, to avoid goroutine Start() wait on push in queue
		}

		// Add this client to the map of those that should receive updates
		b.addClient(uuid, messageChan)

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
				b.cleanClient(messageChan)
				break leave
			case <-r.Context().Done():
				log.Info("events.Http: client disconnected")
				b.cleanClient(messageChan)
				break leave
			case msg := <-messageChan.Queue:
				var buffer bytes.Buffer
				buffer.WriteString("data: ")
				buffer.WriteString(msg)
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

func (b *eventsBroker) manageEvent(receivedEvent sdk.Event, eventS string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, i := range b.clients {
		if i.Queue != nil {
			b.handleEvent(receivedEvent, eventS, i)
		} else {
			log.Warning("eventsBroker.manageEvent > Queue is null for client %+v/%s", i.User, i.UUID)
		}

	}
}

func (b *eventsBroker) handleEvent(event sdk.Event, eventS string, subscriber eventsBrokerSubscribe) {
	if strings.HasPrefix(event.EventType, "sdk.EventProject") {
		if subscriber.User.Admin || permission.ProjectPermission(event.ProjectKey, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventWorkflow") || strings.HasPrefix(event.EventType, "sdk.EventRunWorkflow") {
		if subscriber.User.Admin || permission.WorkflowPermission(event.ProjectKey, event.WorkflowName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventApplication") {
		if subscriber.User.Admin || permission.ApplicationPermission(event.ProjectKey, event.ApplicationName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventPipeline") {
		if subscriber.User.Admin || permission.PipelinePermission(event.ProjectKey, event.PipelineName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventEnvironment") {
		if subscriber.User.Admin || permission.EnvironmentPermission(event.ProjectKey, event.EnvironmentName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventBroadcast") {
		if subscriber.User.Admin || event.ProjectKey == "" || permission.AccessToProject(event.ProjectKey, subscriber.User, permission.PermissionRead) {
			subscriber.Queue <- eventS
		}
		return
	}
}
