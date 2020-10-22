package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/websocket"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

type websocketServer struct {
	server     *websocket.Server
	mutex      sync.RWMutex
	clientData map[string]*websocketClientData
}

func (s *websocketServer) AddClient(c websocket.Client, data *websocketClientData) {
	s.server.AddClient(c)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.clientData[c.UUID()] = data
}

func (s *websocketServer) RemoveClient(uuid string) {
	s.server.RemoveClient(uuid)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.clientData, uuid)
}

func (s *websocketServer) GetClientData(uuid string) *websocketClientData {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	data, ok := s.clientData[uuid]
	if !ok {
		return nil
	}
	return data
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

type websocketClientData struct {
	AuthConsumer sdk.AuthConsumer
	mutex        sync.Mutex
	filters      webSocketFilters
}

func (c *websocketClientData) updateEventFilters(ctx context.Context, db gorp.SqlExecutor, msg []byte) error {
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

// We need to check permission for some kind of events, when permission can't be verified at filter subscription.
func (c *websocketClientData) checkEventPermission(ctx context.Context, db gorp.SqlExecutor, event sdk.Event) (bool, error) {
	var isMaintainer = c.AuthConsumer.Maintainer() || c.AuthConsumer.Admin()
	var isHatchery = c.AuthConsumer.Service != nil && c.AuthConsumer.Service.Type == sdk.TypeHatchery
	var isHatcheryWithGroups = isHatchery && len(c.AuthConsumer.GroupIDs) > 0

	if strings.HasPrefix(event.EventType, "sdk.EventRetentionWorkflowDryRun") {
		return event.Username == c.AuthConsumer.AuthentifiedUser.Username, nil
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

func (a *API) initWebsocket(pubSubKey string) error {
	log.Info(a.Router.Background, "Initializing WS server")
	a.WSServer = &websocketServer{
		server:     websocket.NewServer(),
		clientData: make(map[string]*websocketClientData),
	}
	tickerMetrics := time.NewTicker(10 * time.Second)
	defer tickerMetrics.Stop()
	a.GoRoutines.Run(a.Router.Background, "api.InitRouter.WSServer", func(ctx context.Context) {
		for {
			select {
			case <-tickerMetrics.C:
				telemetry.Record(a.Router.Background, WebSocketClients, int64(len(a.WSServer.server.ClientIDs())))
			case <-ctx.Done():
				telemetry.Record(a.Router.Background, WebSocketClients, 0)
				return
			}
		}
	})

	log.Info(a.Router.Background, "Initializing WS events broker")
	pubSub, err := a.Cache.Subscribe(pubSubKey)
	if err != nil {
		return sdk.WrapError(err, "unable to subscribe to %s", pubSubKey)
	}
	a.WSBroker = websocket.NewBroker()
	a.WSBroker.OnMessage(func(m []byte) {
		telemetry.Record(a.Router.Background, WebSocketEvents, 1)

		var e sdk.Event
		if err := json.Unmarshal(m, &e); err != nil {
			err = sdk.WrapError(err, "cannot parse event from WS broker")
			log.WarningWithFields(a.Router.Background, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			return
		}

		a.websocketOnMessage(e)
	})
	a.WSBroker.Init(a.Router.Background, a.GoRoutines, pubSub, a.PanicDump())
	return nil
}

func (a *API) getWebsocketHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
		c, err := websocket.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			service.WriteError(ctx, w, r, sdk.NewErrorWithStack(err, sdk.ErrWebsocketUpgrade))
			return nil
		}
		defer c.Close()

		wsClient := websocket.NewClient(c)
		wsClientData := &websocketClientData{
			AuthConsumer: *getAPIConsumer(ctx),
		}
		wsClient.OnMessage(func(m []byte) {
			if err := wsClientData.updateEventFilters(ctx, a.mustDBWithCtx(ctx), m); err != nil {
				err = sdk.WithStack(err)
				log.WarningWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				wsClient.Send(sdk.WebsocketEvent{Status: "KO", Error: sdk.Cause(err).Error()})
			}
		})

		a.WSServer.AddClient(wsClient, wsClientData)
		defer a.WSServer.RemoveClient(wsClient.UUID())

		return wsClient.Listen(ctx, a.GoRoutines)
	}
}

func (a *API) websocketOnMessage(e sdk.Event) {
	eventKeys := a.websocketComputeEventKeys(e)
	if len(eventKeys) == 0 {
		return
	}

	// Randomize the order of client to prevent the old client to always received new events in priority
	clientIDs := a.WSServer.server.ClientIDs()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(clientIDs), func(i, j int) { clientIDs[i], clientIDs[j] = clientIDs[j], clientIDs[i] })

	for _, id := range clientIDs {
		// Copy idx for goroutine
		clientID := id

		// Send the event to the client websocket within a goroutine
		a.GoRoutines.Exec(context.Background(), "websocket-"+clientID, func(ctx context.Context) {
			c := a.WSServer.GetClientData(clientID)
			if c == nil {
				return
			}

			found, needCheckPermission := c.filters.HasOneKey(eventKeys...)
			if !found {
				return
			}
			if needCheckPermission {
				allowed, err := c.checkEventPermission(ctx, a.mustDBWithCtx(ctx), e)
				if err != nil {
					err = sdk.WrapError(err, "unable to check event permission for client %s with consumer id: %s", clientID, c.AuthConsumer.ID)
					log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
					return
				}
				if !allowed {
					return
				}
			}
			log.Debug("api.websocketOnMessage> send data to client %s for user %s", clientID, c.AuthConsumer.GetUsername())
			if err := a.WSServer.server.SendToClient(clientID, sdk.WebsocketEvent{
				Status: "OK",
				Event:  e,
			}); err != nil {
				log.Debug("websocketOnMessage> can't send to client %s it will be removed: %+v", clientID, err)
				a.WSServer.RemoveClient(clientID)
			}
		}, a.PanicDump())
	}
}

// This func will compute all the filter keys that match a given event.
func (a *API) websocketComputeEventKeys(event sdk.Event) []string {
	// Compute required filter(s) key for given event
	var keys []string

	// Event that match global filter
	if event.EventType == fmt.Sprintf("%T", sdk.EventFake{}) {
		keys = append(keys, sdk.WebsocketFilter{
			Type: sdk.WebsocketFilterTypeGlobal,
		}.Key())
	}
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
