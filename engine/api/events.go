package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"

	"github.com/tevino/abool"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// eventsBrokerSubscribe is the information needed to subscribe
type eventsBrokerSubscribe struct {
	UUID     string
	consumer *sdk.AuthConsumer
	isAlive  *abool.AtomicBool
	w        http.ResponseWriter
	mutex    sync.Mutex
}

// lastUpdateBroker keeps connected client of the current route,
type eventsBroker struct {
	clients          map[string]*eventsBrokerSubscribe
	messages         chan sdk.Event
	dbFunc           func() *gorp.DbMap
	cache            cache.Store
	router           *Router
	chanAddClient    chan (*eventsBrokerSubscribe)
	chanRemoveClient chan (string)
}

var handledEventErrors = []string{
	"index > windowEnd",
	"runtime error: index out of range",
	"runtime error: slice bounds out of range",
	"runtime error: invalid memory address or nil pointer dereference",
	"write: broken pipe",
	"write: connection reset by peer",
}

//Init the eventsBroker
func (b *eventsBroker) Init(ctx context.Context, panicCallback func(s string) (io.WriteCloser, error)) {
	// Start cache Subscription
	sdk.GoRoutine(ctx, "eventsBroker.Init.CacheSubscribe", func(ctx context.Context) {
		b.cacheSubscribe(ctx, b.messages, b.cache)
	}, panicCallback)

	sdk.GoRoutine(ctx, "eventsBroker.Init.Start", func(ctx context.Context) {
		b.Start(ctx, panicCallback)
	}, panicCallback)
}

func (b *eventsBroker) cacheSubscribe(c context.Context, cacheMsgChan chan<- sdk.Event, store cache.Store) {
	if cacheMsgChan == nil || store == nil {
		return
	}

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
				// don't print the error as we doesn't care
				continue
			}

			switch e.EventType {
			case "sdk.EventJob":
				continue
			}
			observability.Record(b.router.Background, SSEEvents, 1)
			cacheMsgChan <- e
		}
	}
}

// Start the broker
func (b *eventsBroker) Start(ctx context.Context, panicCallback func(s string) (io.WriteCloser, error)) {
	b.chanAddClient = make(chan (*eventsBrokerSubscribe))
	b.chanRemoveClient = make(chan (string))

	tickerMetrics := time.NewTicker(10 * time.Second)
	defer tickerMetrics.Stop()

	for {
		select {
		case <-tickerMetrics.C:
			observability.Record(b.router.Background, SSEClients, int64(len(b.clients)))

		case <-ctx.Done():
			if b.clients != nil {
				for uuid := range b.clients {
					delete(b.clients, uuid)
				}
				observability.Record(b.router.Background, SSEClients, 0)
			}
			if ctx.Err() != nil {
				log.Error("eventsBroker.Start> Exiting: %v", ctx.Err())
				return
			}

		case receivedEvent := <-b.messages:
			for i := range b.clients {
				c := b.clients[i]
				if c == nil {
					delete(b.clients, i)
					continue
				}

				// Send the event to the client sse within a goroutine
				s := "sse-" + b.clients[i].UUID
				sdk.GoRoutine(ctx, s,
					func(ctx context.Context) {
						if c.isAlive.IsSet() {
							log.Debug("eventsBroker> send data to %s", c.UUID)
							if err := c.Send(b.dbFunc(), receivedEvent); err != nil {
								b.chanRemoveClient <- c.UUID
								msg := fmt.Sprintf("%v", err)
								for _, s := range handledEventErrors {
									if strings.Contains(msg, s) {
										// do not log knowned error
										return
									}
								}
								log.Error("eventsBroker> unable to send event to %s: %v", c.UUID, err)
							}
						}
					}, panicCallback,
				)
			}

		case client := <-b.chanAddClient:
			b.clients[client.UUID] = client

		case uuid := <-b.chanRemoveClient:
			client, has := b.clients[uuid]
			if !has {
				continue
			}

			client.isAlive.UnSet()
			delete(b.clients, uuid)
		}
	}
}

func (b *eventsBroker) ServeHTTP() service.Handler {
	// This function may panic when the SSE ResponseWriter is closed, with following message
	// index > windowEnd
	// runtime error: index out of range
	// runtime error: slice bounds out of range
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)
		if !ok {
			return sdk.WrapError(fmt.Errorf("streaming unsupported"), "")
		}

		var client = eventsBrokerSubscribe{
			UUID:     sdk.UUID(),
			consumer: getAPIConsumer(ctx),
			isAlive:  abool.NewBool(true),
			w:        w,
		}

		// Add this client to the map of those that should receive updates
		b.chanAddClient <- &client

		// Set the headers related to event streaming.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		if _, err := w.Write([]byte(fmt.Sprintf("data: ACK: %s \n\n", client.UUID))); err != nil {
			return sdk.WrapError(err, "Unable to send ACK to client")
		}
		f.Flush()

		tick := time.NewTicker(time.Second)
		defer tick.Stop()

	leave:
		for {
			select {
			case <-ctx.Done():
				log.Debug("events.Http: context done")
				b.chanRemoveClient <- client.UUID
				break leave
			case <-r.Context().Done():
				log.Debug("events.Http: client disconnected")
				b.chanRemoveClient <- client.UUID
				break leave
			case <-tick.C:
				_ = client.Send(nil, sdk.Event{})
			}
		}

		return nil
	}
}

func (client *eventsBrokerSubscribe) manageEvent(db gorp.SqlExecutor, event sdk.Event) (bool, error) {
	var isSharedInfra = client.consumer.Groups.HasOneOf(group.SharedInfraGroup.ID)

	switch {
	case strings.HasPrefix(event.EventType, "sdk.EventProject"):
		if isSharedInfra || client.consumer.Maintainer() {
			return true, nil
		}

		perms, err := permission.LoadProjectMaxLevelPermission(context.Background(), db, []string{event.ProjectKey}, client.consumer.GetGroupIDs())
		if err != nil {
			return false, err
		}

		return perms.Level(event.ProjectKey) >= sdk.PermissionRead, nil

	case strings.HasPrefix(event.EventType, "sdk.EventWorkflow") || strings.HasPrefix(event.EventType, "sdk.EventRunWorkflow"):
		if isSharedInfra || client.consumer.Maintainer() {
			return true, nil
		}

		perms, err := permission.LoadWorkflowMaxLevelPermission(context.Background(), db, event.ProjectKey, []string{event.WorkflowName}, client.consumer.GetGroupIDs())
		if err != nil {
			return false, err
		}

		return perms.Level(event.WorkflowName) >= sdk.PermissionRead, nil

	case strings.HasPrefix(event.EventType, "sdk.EventBroadcast"):
		if event.ProjectKey == "" {
			return true, nil
		}

		if isSharedInfra || client.consumer.Maintainer() {
			return true, nil
		}

		perms, err := permission.LoadProjectMaxLevelPermission(context.Background(), db, []string{event.ProjectKey}, client.consumer.GetGroupIDs())
		if err != nil {
			return false, err
		}

		return perms.Level(event.ProjectKey) >= sdk.PermissionRead, nil
	default:
		return false, nil

	}
}

// Send an event to a client
func (client *eventsBrokerSubscribe) Send(db gorp.SqlExecutor, event sdk.Event) (err error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("eventsBrokerSubscribe.Send recovered %v", r)
		}
	}()

	if client == nil || client.w == nil {
		return nil
	}

	// Make sure that the writer supports flushing.
	f, ok := client.w.(http.Flusher)
	if !ok {
		return sdk.WrapError(fmt.Errorf("streaming unsupported"), "")
	}

	var buffer bytes.Buffer
	if event.EventType != "" {
		if ok, err := client.manageEvent(db, event); !ok {
			return err
		}

		msg, err := json.Marshal(event)
		if err != nil {
			return sdk.WrapError(err, "Unable to marshall event")
		}
		buffer.WriteString("data: ")
		buffer.Write(msg)
		buffer.WriteString("\n\n")
	} else {
		buffer.WriteString("")
	}

	if !client.isAlive.IsSet() {
		return nil
	}

	if _, err := client.w.Write(buffer.Bytes()); err != nil {
		return sdk.WrapError(err, "unable to write to client")
	}
	f.Flush()

	return nil
}
