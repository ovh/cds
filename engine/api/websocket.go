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
	"github.com/sirupsen/logrus"
	"github.com/tevino/abool"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type websocketClient struct {
	UUID          string
	AuthConsumer  *sdk.AuthConsumer
	isAlive       *abool.AtomicBool
	con           *websocket.Conn
	mutex         sync.Mutex
	filters       webSocketFilters
	inMessageChan chan []byte
}

type webSocketFilters map[string]struct{}

func (f webSocketFilters) HasOneKey(keys ...string) bool {
	for i := range keys {
		if _, ok := f[keys[i]]; ok {
			return true
		}
	}
	return false
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
			return
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
							if err := c.send(ctx, b.dbFunc(), receivedEvent); err != nil {
								b.chanRemoveClient <- c.UUID
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
			return
		case <-tick.C:
			if ctx.Err() != nil {
				continue
			}

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

			observability.Record(b.router.Background, WebSocketEvents, 1)
			cacheMsgChan <- e
		}
	}
}

func (b *websocketBroker) ServeHTTP() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return sdk.WithStack(err)
		}
		defer c.Close()

		client := websocketClient{
			UUID:          sdk.UUID(),
			AuthConsumer:  getAPIConsumer(ctx),
			isAlive:       abool.NewBool(true),
			con:           c,
			inMessageChan: make(chan []byte, 10),
		}
		b.chanAddClient <- &client

		sdk.GoRoutine(ctx, fmt.Sprintf("readUpdateFilterChan-%s-%s", client.AuthConsumer.GetUsername(), client.UUID), func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					log.Debug("events.Http: context done")
					return
				case m := <-client.inMessageChan:
					if err := client.updateEventFilters(ctx, b.dbFunc(), m); err != nil {
						err = sdk.WithStack(err)
						log.WarningWithFields(ctx, logrus.Fields{
							"stack_trace": fmt.Sprintf("%+v", err),
						}, "%s", err)
						msg := sdk.WebsocketEvent{
							Status: "KO",
							Error:  sdk.Cause(err).Error(),
						}
						_ = client.con.WriteJSON(msg)
					}
				}
			}
		})

		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			_, msg, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					err = sdk.WrapError(err, "websocket error occured")
					log.WarningWithFields(ctx, logrus.Fields{
						"stack_trace": fmt.Sprintf("%+v", err),
					}, "%s", err)
				}
				log.Debug("%s disconnected", client.AuthConsumer.GetUsername())
				break
			}

			client.inMessageChan <- msg
		}
		return nil
	}
}

func (c *websocketClient) updateEventFilters(ctx context.Context, db gorp.SqlExecutor, msg []byte) error {
	var fs []sdk.WebsocketFilter

	var f sdk.WebsocketFilter
	if err := json.Unmarshal(msg, &f); err == nil {
		fs = []sdk.WebsocketFilter{f}
	} else {
		if err := json.Unmarshal(msg, &fs); err != nil {
			return sdk.WrapError(err, "cannot unmarshal websocket input message")
		}
	}

	var isMaintainer = c.AuthConsumer.Maintainer() || c.AuthConsumer.Admin()
	var isHatchery = c.AuthConsumer.Service != nil && c.AuthConsumer.Service.Type == services.TypeHatchery
	var isHatcheryWithGroups = isHatchery && len(c.AuthConsumer.GroupIDs) > 0

	// Check validity of given filters
	for _, f := range fs {
		if err := f.IsValid(); err != nil {
			return err
		}
		switch f.Type {
		case sdk.WebsocketFilterTypeProject,
			sdk.WebsocketFilterTypeApplication,
			sdk.WebsocketFilterTypePipeline,
			sdk.WebsocketFilterTypeEnvironment,
			sdk.WebsocketFilterTypeOperation:
			if isMaintainer && !isHatcheryWithGroups {
				continue
			}
			perms, err := permission.LoadProjectMaxLevelPermission(ctx, db, []string{f.ProjectKey}, c.AuthConsumer.GetGroupIDs())
			if err != nil {
				return err
			}
			maxLevelPermission := perms.Level(f.ProjectKey)
			if maxLevelPermission < sdk.PermissionRead {
				return sdk.WithStack(sdk.ErrForbidden)
			}
		case sdk.WebsocketFilterTypeWorkflow:
			if isMaintainer && !isHatcheryWithGroups {
				continue
			}
			perms, err := permission.LoadWorkflowMaxLevelPermission(ctx, db, f.ProjectKey, []string{f.WorkflowName}, c.AuthConsumer.GetGroupIDs())
			if err != nil {
				return err
			}
			maxLevelPermission := perms.Level(f.WorkflowName)
			if maxLevelPermission < sdk.PermissionRead {
				return sdk.WithStack(sdk.ErrForbidden)
			}
		}
	}

	// Update client filters
	c.mutex.Lock()
	c.filters = make(webSocketFilters)
	for i := range fs {
		c.filters[fs[i].Key()] = struct{}{}
	}
	c.mutex.Unlock()

	return nil
}

// Send an event to a client
func (c *websocketClient) send(ctx context.Context, db gorp.SqlExecutor, event sdk.Event) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	defer func() {
		if r := recover(); r != nil {
			err = sdk.WithStack(fmt.Errorf("websocketClient.Send recovered %v", r))
		}
	}()

	if c == nil || c.con == nil || !c.isAlive.IsSet() {
		return sdk.WithStack(fmt.Errorf("client deconnected"))
	}

	var isMaintainer = c.AuthConsumer.Maintainer() || c.AuthConsumer.Admin()
	var isHatchery = c.AuthConsumer.Service != nil && c.AuthConsumer.Service.Type == services.TypeHatchery
	var isHatcheryWithGroups = isHatchery && len(c.AuthConsumer.GroupIDs) > 0

	// Compute required filter(s) key for given event
	var keys []string

	// Event that match global filter
	if event.EventType == fmt.Sprintf("%T", sdk.EventMaintenance{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type: sdk.WebsocketFilterTypeGlobal,
		}.Key())
	}
	if strings.HasPrefix(event.EventType, "sdk.EventBroadcast") {
		var allowed bool
		if event.ProjectKey == "" {
			allowed = true
		} else {
			if isMaintainer && !isHatcheryWithGroups {
				allowed = true
			} else {
				perms, err := permission.LoadProjectMaxLevelPermission(context.Background(), db, []string{event.ProjectKey}, c.AuthConsumer.GetGroupIDs())
				if err != nil {
					return err
				}
				allowed = perms.Level(event.ProjectKey) >= sdk.PermissionRead
			}
		}
		if allowed {
			keys = append(keys, sdk.WebsocketFilter{
				Type: sdk.WebsocketFilterTypeGlobal,
			}.Key())
		}
	}
	// Event that match project filter
	if strings.HasPrefix(event.EventType, "sdk.EventProject") || event.EventType == fmt.Sprintf("%T", sdk.EventAsCodeEvent{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type:       sdk.WebsocketFilterTypeProject,
			ProjectKey: event.ProjectKey,
		}.Key())
	}
	// Event that match workflow filter
	if strings.HasPrefix(event.EventType, "sdk.EventWorkflow") || event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflow{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type:         sdk.WebsocketFilterTypeWorkflow,
			ProjectKey:   event.ProjectKey,
			WorkflowName: event.WorkflowName,
		}.Key())
	}
	// Event that match workflow run filter
	if event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflowJob{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type:              sdk.WebsocketFilterTypeWorkflowRun,
			ProjectKey:        event.ProjectKey,
			WorkflowName:      event.WorkflowName,
			WorkflowRunNumber: event.WorkflowRunNum,
		}.Key())
	}
	// Event that match workflow node run filter
	if event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type:              sdk.WebsocketFilterTypeWorkflowNodeRun,
			ProjectKey:        event.ProjectKey,
			WorkflowName:      event.WorkflowName,
			WorkflowNodeRunID: event.WorkflowNodeRunID,
		}.Key())
	}
	// Event that match pipeline filter
	if strings.HasPrefix(event.EventType, "sdk.EventPipeline") {
		keys = append(keys, sdk.WebsocketFilter{
			Type:         sdk.WebsocketFilterTypePipeline,
			ProjectKey:   event.ProjectKey,
			PipelineName: event.PipelineName,
		}.Key())
	}
	// Event that match application filter
	if strings.HasPrefix(event.EventType, "sdk.EventApplication") {
		keys = append(keys, sdk.WebsocketFilter{
			Type:            sdk.WebsocketFilterTypeApplication,
			ProjectKey:      event.ProjectKey,
			ApplicationName: event.ApplicationName,
		}.Key())
	}
	// Event that match environment filter
	if strings.HasPrefix(event.EventType, "sdk.EventEnvironment") {
		keys = append(keys, sdk.WebsocketFilter{
			Type:            sdk.WebsocketFilterTypeEnvironment,
			ProjectKey:      event.ProjectKey,
			EnvironmentName: event.EnvironmentName,
		}.Key())
	}
	// Event that match queue filter
	if event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflowJob{}) {
		// We need to check the permission on project here
		var allowed bool
		if isMaintainer && !isHatcheryWithGroups {
			allowed = true
		} else {
			// We search permission from database to allow events for project created after websocket init to be retuned.
			// As the AuthConsumer group list is not updated, events for project where a group will be added after websocket
			// init will not be returned until socket reconnection.
			perms, err := permission.LoadWorkflowMaxLevelPermission(context.Background(), db, event.ProjectKey, []string{event.WorkflowName}, c.AuthConsumer.GetGroupIDs())
			if err != nil {
				return err
			}
			allowed = perms.Level(event.WorkflowName) >= sdk.PermissionRead
		}
		if allowed {
			keys = append(keys, sdk.WebsocketFilter{
				Type: sdk.WebsocketFilterTypeQueue,
			}.Key())
		}
	}
	// Event that match operation filter
	if event.EventType == fmt.Sprintf("%T", sdk.Operation{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type:          sdk.WebsocketFilterTypeOperation,
			ProjectKey:    event.ProjectKey,
			OperationUUID: event.OperationUUID,
		}.Key())
	}
	if len(keys) == 0 || !c.filters.HasOneKey(keys...) {
		return nil
	}

	msg := sdk.WebsocketEvent{
		Status: "OK",
		Event:  event,
	}
	if err := c.con.WriteJSON(msg); err != nil {
		// ErrCloseSent is returned when the application writes a message to the connection after sending a close message.
		if err == websocket.ErrCloseSent {
			return sdk.WithStack(err)
		}
		if strings.Contains(err.Error(), "use of closed network connection") {
			return sdk.WithStack(err)
		}
		log.Error(ctx, "websocketClient.Send > unable to write json: %v", err)
	}
	return nil
}
