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
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/service"
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
	clients  map[string]eventsBrokerSubscribe
	messages chan sdk.Event
	mutex    *sync.Mutex
	dbFunc   func() *gorp.DbMap
	cache    cache.Store
	router   *Router
}

// AddClient add a client to the client map
func (b *eventsBroker) addClient(ctx context.Context, client eventsBrokerSubscribe) {
	b.mutex.Lock()
	b.clients[client.UUID] = client
	b.mutex.Unlock()
	go observability.Record(ctx, b.router.Stats.SSEClients, 1)
}

// CleanAll cleans all clients
func (b *eventsBroker) cleanAll() {
	b.mutex.Lock()
	if b.clients != nil {
		defer observability.Record(b.router.Background, b.router.Stats.SSEClients, -1*int64(len(b.clients)))
		for c, v := range b.clients {
			close(v.Queue)
			delete(b.clients, c)
		}
	}
	b.mutex.Unlock()
}

func (b *eventsBroker) disconnectClient(ctx context.Context, uuid string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	client, has := b.clients[uuid]
	if !has {
		return
	}

	close(client.Queue)
	delete(b.clients, uuid)

	go observability.Record(ctx, b.router.Stats.SSEClients, -1)
}

//Init the eventsBroker
func (b *eventsBroker) Init(ctx context.Context) {
	// Start cache Subscription
	sdk.GoRoutine(ctx, "eventsBroker.Init.CacheSubscribe", func(ctx context.Context) {
		b.cacheSubscribe(ctx, b.messages, b.cache)
	})

	sdk.GoRoutine(ctx, "eventsBroker.Init.Start", func(ctx context.Context) {
		b.Start(ctx)
	})
}

func (b *eventsBroker) cacheSubscribe(c context.Context, cacheMsgChan chan<- sdk.Event, store cache.Store) {
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
			observability.Record(c, b.router.Stats.SSEEvents, 1)
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

func (b *eventsBroker) ServeHTTP() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)
		if !ok {
			return sdk.WrapError(fmt.Errorf("streaming unsupported"), "")
		}

		uuid := sdk.UUID()
		client := eventsBrokerSubscribe{
			UUID:  uuid,
			User:  getUser(ctx),
			Queue: make(chan sdk.Event, 10), // chan buffered, to avoid goroutine Start() wait on push in queue
		}

		// Add this client to the map of those that should receive updates
		b.addClient(ctx, client)

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
				b.disconnectClient(ctx, client.UUID)
				break leave
			case <-r.Context().Done():
				log.Info("events.Http: client disconnected")
				b.disconnectClient(ctx, client.UUID)
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
	// Create a slice of clients with a mutex
	b.mutex.Lock()
	clients := make([]eventsBrokerSubscribe, len(b.clients))
	var i int
	for _, client := range b.clients {
		clients[i] = client
		i++
	}
	b.mutex.Unlock()
	// Then iterate over it outside the mutex
	for _, c := range clients {
		log.Debug("send data to %s", c.UUID)
		c.Queue <- receivedEvent
	}
}

func (s *eventsBrokerSubscribe) manageEvent(event sdk.Event) bool {
	var isSharedInfra bool
	for _, g := range s.User.Groups {
		if g.ID == group.SharedInfraGroup.ID {
			isSharedInfra = true
			break
		}
	}

	if strings.HasPrefix(event.EventType, "sdk.EventProject") {
		if s.User.Admin || isSharedInfra || permission.ProjectPermission(event.ProjectKey, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventWorkflow") || strings.HasPrefix(event.EventType, "sdk.EventRunWorkflow") {
		if s.User.Admin || isSharedInfra || permission.WorkflowPermission(event.ProjectKey, event.WorkflowName, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventApplication") {
		if s.User.Admin || isSharedInfra || permission.ApplicationPermission(event.ProjectKey, event.ApplicationName, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventPipeline") {
		if s.User.Admin || isSharedInfra || permission.PipelinePermission(event.ProjectKey, event.PipelineName, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventEnvironment") {
		if s.User.Admin || isSharedInfra || permission.EnvironmentPermission(event.ProjectKey, event.EnvironmentName, s.User) >= permission.PermissionRead {
			return true
		}
		return false
	}
	if strings.HasPrefix(event.EventType, "sdk.EventBroadcast") {
		if s.User.Admin || isSharedInfra || event.ProjectKey == "" || permission.AccessToProject(event.ProjectKey, s.User, permission.PermissionRead) {
			return true
		}
		return false
	}
	return false
}
