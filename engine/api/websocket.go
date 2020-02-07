package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/websocket"
	"github.com/tevino/abool"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var upgrader = websocket.Upgrader{} // use default options

type websocketClient struct {
	UUID             string
	AuthConsumer     *sdk.AuthConsumer
	isAlive          *abool.AtomicBool
	con              *websocket.Conn
	mutex            sync.Mutex
	filter           sdk.WebsocketFilter
	updateFilterChan chan sdk.WebsocketFilter
}

type websocketBroker struct {
	clients          map[string]*websocketClient
	cache            cache.Store
	dbFunc           func() *gorp.DbMap
	router           *Router
	messages         chan sdk.Event
	chanAddClient    chan *websocketClient
	chanRemoveClient chan string
}

//Init the websocketBroker
func (b *websocketBroker) Init(ctx context.Context, panicCallback func(s string) (io.WriteCloser, error)) {
	// Start cache Subscription
	sdk.GoRoutine(ctx, "websocketBroker.Init.CacheSubscribe", func(ctx context.Context) {
		b.cacheSubscribe(ctx, b.messages, b.cache)
	}, panicCallback)

	sdk.GoRoutine(ctx, "websocketBroker.Init.Start", func(ctx context.Context) {
		b.Start(ctx, panicCallback)
	}, panicCallback)
}

// Start the broker
func (b *websocketBroker) Start(ctx context.Context, panicCallback func(s string) (io.WriteCloser, error)) {
	tickerMetrics := time.NewTicker(10 * time.Second)
	defer tickerMetrics.Stop()

	for {
		select {
		case <-tickerMetrics.C:
			observability.Record(b.router.Background, WebSocketClients, int64(len(b.clients)))
		case <-ctx.Done():
			if b.clients != nil {
				for uuid := range b.clients {
					delete(b.clients, uuid)
				}
				observability.Record(b.router.Background, WebSocketClients, 0)
			}
			if ctx.Err() != nil {
				log.Error(ctx, "websocketBroker.Start> Exiting: %v", ctx.Err())
				return
			}

		case receivedEvent := <-b.messages:
			for i := range b.clients {
				c := b.clients[i]
				if c == nil {
					delete(b.clients, i)
					continue
				}

				// Send the event to the client websocket within a goroutine
				s := "websocket-" + b.clients[i].UUID
				sdk.GoRoutine(ctx, s,
					func(ctx context.Context) {
						if c.isAlive.IsSet() {
							log.Debug("send data to %s", c.AuthConsumer.GetUsername())
							if err := c.send(ctx, receivedEvent); err != nil {
								b.chanRemoveClient <- c.UUID
								log.Error(ctx, "websocketBroker.Start> unable to send event to %s: %v", c.AuthConsumer.GetUsername(), err)
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

func (b *websocketBroker) cacheSubscribe(ctx context.Context, cacheMsgChan chan<- sdk.Event, store cache.Store) {
	if cacheMsgChan == nil {
		return
	}

	pubSub, err := store.Subscribe("events_pubsub")
	if err != nil {
		log.Error(ctx, "websocketBroker.cacheSubscribe> unable to subscribe to events_pubsub")
	}
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "websocketBroker.cacheSubscribe> Exiting: %v", ctx.Err())
				return
			}
		case <-tick.C:
			msg, err := store.GetMessageFromSubscription(ctx, pubSub)
			if err != nil {
				log.Warning(ctx, "websocketBroker.cacheSubscribe> Cannot get message %s: %s", msg, err)
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
			observability.Record(b.router.Background, WebSocketEvents, 1)
			cacheMsgChan <- e
		}
	}
}

func (b *websocketBroker) ServeHTTP() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Warning(ctx, "websocket> upgrade: %v", err)
			return err
		}
		defer c.Close()

		client := websocketClient{
			UUID:             sdk.UUID(),
			AuthConsumer:     getAPIConsumer(ctx),
			isAlive:          abool.NewBool(true),
			con:              c,
			updateFilterChan: make(chan sdk.WebsocketFilter, 10),
		}
		b.chanAddClient <- &client

		sdk.GoRoutine(ctx, fmt.Sprintf("readUpdateFilterChan-%s-%s", client.AuthConsumer.GetUsername(), client.UUID), func(ctx context.Context) {
			client.readUpdateFilterChan(ctx, b.dbFunc())
		})

		if err := c.WriteJSON(sdk.Event{
			EventType:    "sdk.EventProject",
			ProjectKey:   "foo",
			WorkflowName: "bar",
		}); err != nil {
			log.Error(ctx, "websocketClient.Send > unable to write json: %v", err)
		}
		if err := c.WriteJSON(sdk.Event{
			EventType:    "sdk.EventProject",
			ProjectKey:   "foo",
			WorkflowName: "bar",
		}); err != nil {
			log.Error(ctx, "websocketClient.Send > unable to write json: %v", err)
		}

		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			var msg sdk.WebsocketFilter
			_, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Warning(ctx, "websocket error: %v", err)
				}
				log.Debug("%s disconnected", client.AuthConsumer.GetUsername())
				break
			}

			if err := json.Unmarshal(message, &msg); err != nil {
				log.Warning(ctx, "websocket.readJSON: %v", err)
			}

			// Send message to client
			client.updateFilterChan <- msg
		}
		return nil
	}
}

func (c *websocketClient) readUpdateFilterChan(ctx context.Context, db *gorp.DbMap) {
	for {
		select {
		case <-ctx.Done():
			log.Debug("events.Http: context done")
			return
		case m := <-c.updateFilterChan:
			if err := c.updateEventFilter(ctx, db, m); err != nil {
				log.Error(ctx, "websocketClient.read: unable to update event filter: %v", err)
				msg := sdk.WebsocketEvent{
					Status: "KO",
					Error:  sdk.Cause(err).Error(),
				}
				_ = c.con.WriteJSON(msg)
				continue
			}
		}
	}
}

func (c *websocketClient) updateEventFilter(ctx context.Context, db gorp.SqlExecutor, m sdk.WebsocketFilter) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// Subscribe to project
	if (m.ProjectKey != "" && m.WorkflowName == "") || (m.ProjectKey != "" && m.Operation != "") {
		perms, err := permission.LoadProjectMaxLevelPermission(ctx, db, []string{m.ProjectKey}, getAPIConsumer(ctx).GetGroupIDs())
		if err != nil {
			return err
		}
		maxLevelPermission := perms.Level(m.ProjectKey)
		if maxLevelPermission < sdk.PermissionRead && !isMaintainer(ctx) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		c.filter = m
	}

	// Subscribe to workflow
	if m.ProjectKey != "" && m.WorkflowName != "" {
		perms, err := permission.LoadWorkflowMaxLevelPermission(ctx, db, m.ProjectKey, []string{m.WorkflowName}, getAPIConsumer(ctx).GetGroupIDs())
		if err != nil {
			return err
		}
		maxLevelPermission := perms.Level(m.WorkflowName)
		if maxLevelPermission < sdk.PermissionRead && !isMaintainer(ctx) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		c.filter = m
	}

	c.filter.Queue = m.Queue

	return nil
}

// Send an event to a client
func (c *websocketClient) send(ctx context.Context, event sdk.Event) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("websocketClient.Send recovered %v", r)
		}
	}()

	if c == nil || c.con == nil || !c.isAlive.IsSet() {
		return nil
	}

	if event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflowJob{}) && c.filter.Queue {
		// Do not check anything else

		// OPERATION EVENT
	} else if event.EventType == fmt.Sprintf("%T", sdk.Operation{}) && c.filter.Operation == event.OperationUUID && c.filter.ProjectKey == event.ProjectKey {
		// Do not check anything else
	} else {
		switch {
		// PROJECT EVENT
		case strings.HasPrefix(event.EventType, "sdk.EventProject"):
			if event.ProjectKey != c.filter.ProjectKey {
				return nil
			}
			// APPLICATION EVENT
		case strings.HasPrefix(event.EventType, "sdk.EventApplication"):
			if event.ProjectKey != c.filter.ProjectKey || event.ApplicationName != c.filter.ApplicationName {
				return nil
			}
			// PIPELINE EVENT
		case strings.HasPrefix(event.EventType, "sdk.EventPipeline"):
			if event.ProjectKey != c.filter.ProjectKey || event.PipelineName != c.filter.PipelineName {
				return nil
			}
			// ENVIRONMENT EVENT
		case strings.HasPrefix(event.EventType, "sdk.EventEnvironment"):
			if event.ProjectKey != c.filter.ProjectKey || event.EnvironmentName != c.filter.EnvironmentName {
				return nil
			}
			// WORKFLOW EVENT
		case strings.HasPrefix(event.EventType, "sdk.EventWorkflow"):
			if event.ProjectKey != c.filter.ProjectKey || event.WorkflowName != c.filter.WorkflowName {
				return nil
			}
		case strings.HasPrefix(event.EventType, "sdk.EventRunWorkflow"):
			// WORKFLOW RUN EVENT
			if event.ProjectKey != c.filter.ProjectKey || event.WorkflowName != c.filter.WorkflowName {
				return nil
			}

			if c.filter.WorkflowRunNumber != 0 && event.WorkflowRunNum != c.filter.WorkflowRunNumber {
				return nil
			}
			// WORKFLOW NODE RUN EVENT
			if c.filter.WorkflowNodeRunID != 0 && event.WorkflowNodeRunID != c.filter.WorkflowNodeRunID {
				return nil
			}
		default:
			return nil
		}
	}

	msg := sdk.WebsocketEvent{
		Status: "OK",
		Event:  event,
	}
	if err := c.con.WriteJSON(msg); err != nil {
		log.Error(ctx, "websocketClient.Send > unable to write json: %v", err)
	}
	return nil
}
