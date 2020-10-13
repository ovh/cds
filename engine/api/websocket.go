package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/websocket"
	"github.com/tevino/abool"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
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

type webSocketFilters []sdk.WebsocketFilter

func (f webSocketFilters) HasOneKey(keys ...string) (found bool, needCheckPermission bool) {
	for i := range keys {
		for _, filter := range f {
			if keys[i] == filter.Key() {
				found = true
				switch filter.Type {
				case sdk.WebsocketFilterTypeGlobal, sdk.WebsocketFilterTypeQueue, sdk.WebsocketFilterTypeTimeline, sdk.WebsocketFilterTypeDryRunRetentionWorkflow:
					needCheckPermission = true
				}
				// If we found a filter that don't need to check permission we can return directly
				// If not we will check if another filter match the given keys, this will prevent from checking permission if not needed
				if !needCheckPermission {
					return
				}
			}
		}
	}
	return
}

type websocketBroker struct {
	clients          map[string]*websocketClient
	cache            cache.Store
	dbFunc           func() *gorp.DbMap
	router           *Router
	messages         chan sdk.Event
	chanAddClient    chan *websocketClient
	chanRemoveClient chan string
	goRoutines       *sdk.GoRoutines
}

//Init the websocketBroker
func (b *websocketBroker) Init(ctx context.Context, panicCallback func(s string) (io.WriteCloser, error), goRoutines *sdk.GoRoutines) {
	// Start cache Subscription
	goRoutines.Run(ctx, "websocketBroker.Init.CacheSubscribe", func(ctx context.Context) {
		b.cacheSubscribe(ctx, b.messages, b.cache)
	}, panicCallback)

	goRoutines.Run(ctx, "websocketBroker.Init.Start", func(ctx context.Context) {
		b.Start(ctx, panicCallback, goRoutines)
	}, panicCallback)
}

// Start the broker
func (b *websocketBroker) Start(ctx context.Context, panicCallback func(s string) (io.WriteCloser, error), goRoutines *sdk.GoRoutines) {
	tickerMetrics := time.NewTicker(10 * time.Second)
	defer tickerMetrics.Stop()

	for {
		select {
		case <-tickerMetrics.C:
			telemetry.Record(b.router.Background, WebSocketClients, int64(len(b.clients)))
		case <-ctx.Done():
			if b.clients != nil {
				for uuid := range b.clients {
					delete(b.clients, uuid)
				}
				telemetry.Record(b.router.Background, WebSocketClients, 0)
			}
			return
		case receivedEvent := <-b.messages:
			eventKeys := b.computeEventKeys(receivedEvent)
			if len(eventKeys) == 0 {
				continue
			}

			// Randomize the order of client to prevent the old client to always received new events in priority
			clientIDs := make([]string, 0, len(b.clients))
			for i := range b.clients {
				clientIDs = append(clientIDs, i)
			}
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			r.Shuffle(len(clientIDs), func(i, j int) { clientIDs[i], clientIDs[j] = clientIDs[j], clientIDs[i] })

			for _, id := range clientIDs {
				c := b.clients[id]
				if c == nil {
					delete(b.clients, id)
					continue
				}

				// Send the event to the client websocket within a goroutine
				s := "websocket-" + c.UUID
				goRoutines.Exec(ctx, s, func(ctx context.Context) {
					found, needCheckPermission := c.filters.HasOneKey(eventKeys...)
					if !found {
						return
					}
					if needCheckPermission {
						allowed, err := c.checkEventPermission(ctx, b.dbFunc(), receivedEvent)
						if err != nil {
							err = sdk.WrapError(err, "unable to check event permission for client %s with consumer id: %s", c.UUID, c.AuthConsumer.ID)
							log.ErrorWithFields(ctx, log.Fields{
								"stack_trace": fmt.Sprintf("%+v", err),
							}, "%s", err)
							return
						}
						if !allowed {
							return
						}
					}
					if c.isAlive.IsSet() {
						log.Debug("send data to %s", c.AuthConsumer.GetUsername())
						if err := c.send(ctx, receivedEvent); err != nil {
							log.Debug("can't send to client %s it will be removed: %+v", c.UUID, err)
							b.chanRemoveClient <- c.UUID
						}
					}
				}, panicCallback)
			}

		case client := <-b.chanAddClient:
			log.Debug("add new websocket client %s for consumer %s", client.UUID, client.AuthConsumer.GetUsername())
			b.clients[client.UUID] = client

		case uuid := <-b.chanRemoveClient:
			log.Debug("remove websocket client %s", uuid)
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

			telemetry.Record(b.router.Background, WebSocketEvents, 1)
			cacheMsgChan <- e
		}
	}
}

func (b *websocketBroker) ServeHTTP() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			service.WriteError(ctx, w, r, sdk.WithStack(sdk.ErrWebsocketUpgrade))
			return nil
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
		defer func() {
			close(client.inMessageChan)
			b.chanRemoveClient <- client.UUID
		}()

		b.goRoutines.Exec(ctx, fmt.Sprintf("readUpdateFilterChan-%s-%s", client.AuthConsumer.GetUsername(), client.UUID), func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					log.Debug("events.Http: context done")
					return
				case m, more := <-client.inMessageChan:
					if !more {
						return
					}
					if err := client.updateEventFilters(ctx, b.dbFunc(), m); err != nil {
						err = sdk.WithStack(err)
						log.WarningWithFields(ctx, log.Fields{
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
					log.WarningWithFields(ctx, log.Fields{
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
	var isHatchery = c.AuthConsumer.Service != nil && c.AuthConsumer.Service.Type == sdk.TypeHatchery
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
		case sdk.WebsocketFilterTypeWorkflow, sdk.WebsocketFilterTypeAscodeEvent, sdk.WebsocketFilterTypeDryRunRetentionWorkflow:
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
	c.filters = fs
	c.mutex.Unlock()

	return nil
}

// This func will compute all the filter keys that match a given event.
func (b *websocketBroker) computeEventKeys(event sdk.Event) []string {
	// Compute required filter(s) key for given event
	var keys []string

	// Event that match global filter
	if event.EventType == fmt.Sprintf("%T", sdk.EventMaintenance{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type: sdk.WebsocketFilterTypeGlobal,
		}.Key())
	}
	if strings.HasPrefix(event.EventType, "sdk.EventBroadcast") {
		keys = append(keys, sdk.WebsocketFilter{
			Type: sdk.WebsocketFilterTypeGlobal,
		}.Key())
	}
	// Event that match project filter
	if strings.HasPrefix(event.EventType, "sdk.EventProject") {
		keys = append(keys, sdk.WebsocketFilter{
			Type:       sdk.WebsocketFilterTypeProject,
			ProjectKey: event.ProjectKey,
		}.Key())
	}
	// Event that match Purge Filter
	if strings.HasPrefix(event.EventType, "sdk.EventRetentionWorkflowDryRun") {
		keys = append(keys, sdk.WebsocketFilter{
			Type:         sdk.WebsocketFilterTypeDryRunRetentionWorkflow,
			ProjectKey:   event.ProjectKey,
			WorkflowName: event.WorkflowName,
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
	if event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflow{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type:              sdk.WebsocketFilterTypeWorkflowRun,
			ProjectKey:        event.ProjectKey,
			WorkflowName:      event.WorkflowName,
			WorkflowRunNumber: event.WorkflowRunNum,
		}.Key())
	}
	if event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}) {
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
		keys = append(keys, sdk.WebsocketFilter{
			Type: sdk.WebsocketFilterTypeQueue,
		}.Key())
	}
	// Event that match operation filter
	if event.EventType == fmt.Sprintf("%T", sdk.EventOperation{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type:          sdk.WebsocketFilterTypeOperation,
			ProjectKey:    event.ProjectKey,
			OperationUUID: event.OperationUUID,
		}.Key())
	}
	// Event that match timeline filter
	if event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflow{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type: sdk.WebsocketFilterTypeTimeline,
		}.Key())
	}
	// Event that match as code event filter
	if event.EventType == fmt.Sprintf("%T", sdk.EventAsCodeEvent{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type:         sdk.WebsocketFilterTypeAscodeEvent,
			ProjectKey:   event.ProjectKey,
			WorkflowName: event.WorkflowName,
		}.Key())
	}

	return keys
}

// We need to check permission for some kind of events, when permission can't be verified at filter subscription.
func (c *websocketClient) checkEventPermission(ctx context.Context, db gorp.SqlExecutor, event sdk.Event) (bool, error) {
	var isMaintainer = c.AuthConsumer.Maintainer() || c.AuthConsumer.Admin()
	var isHatchery = c.AuthConsumer.Service != nil && c.AuthConsumer.Service.Type == sdk.TypeHatchery
	var isHatcheryWithGroups = isHatchery && len(c.AuthConsumer.GroupIDs) > 0

	if strings.HasPrefix(event.EventType, "sdk.EventRetentionWorkflowDryRun") {
		if event.Username == c.AuthConsumer.AuthentifiedUser.Username {
			return true, nil
		}
	}

	if strings.HasPrefix(event.EventType, "sdk.EventBroadcast") {
		if event.ProjectKey == "" {
			return true, nil
		}
		if isMaintainer && !isHatcheryWithGroups {
			return true, nil
		}
		perms, err := permission.LoadProjectMaxLevelPermission(ctx, db, []string{event.ProjectKey}, c.AuthConsumer.GetGroupIDs())
		if err != nil {
			return false, err
		}
		return perms.Level(event.ProjectKey) >= sdk.PermissionRead, nil
	}
	if event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflow{}) || event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflowJob{}) {
		// We need to check the permission on project here
		if isMaintainer && !isHatcheryWithGroups {
			return true, nil
		}
		// We search permission from database to allow events for project created after websocket init to be retuned.
		// As the AuthConsumer group list is not updated, events for project where a group will be added after websocket
		// init will not be returned until socket reconnection.
		perms, err := permission.LoadWorkflowMaxLevelPermission(ctx, db, event.ProjectKey, []string{event.WorkflowName}, c.AuthConsumer.GetGroupIDs())
		if err != nil {
			return false, err
		}
		return perms.Level(event.WorkflowName) >= sdk.PermissionRead, nil
	}

	return true, nil
}

// Send an event to a client
func (c *websocketClient) send(ctx context.Context, event sdk.Event) (err error) {
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
