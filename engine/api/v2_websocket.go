package api

import (
	"context"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/websocket"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

type websocketV2Server struct {
	server     *websocket.Server
	mutex      sync.RWMutex
	clientData map[string]*websocketV2ClientData
}

func (s *websocketV2Server) AddClient(c websocket.Client, data *websocketV2ClientData) {
	s.server.AddClient(c)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.clientData[c.UUID()] = data
}

func (s *websocketV2Server) RemoveClient(uuid string) {
	s.server.RemoveClient(uuid)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.clientData, uuid)
}

func (s *websocketV2Server) GetClientData(uuid string) *websocketV2ClientData {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	data, ok := s.clientData[uuid]
	if !ok {
		return nil
	}
	return data
}

type webSocketV2Filters []sdk.WebsocketV2Filter

func (f webSocketV2Filters) HasOneKey(keys ...string) (found bool, needPostCheck bool) {
	for i := range keys {
		for _, filter := range f {
			if keys[i] == filter.Key() {
				found = true
				switch filter.Type {
				case sdk.WebsocketV2FilterTypeGlobal,
					sdk.WebsocketV2FilterTypeProjectRuns:
					needPostCheck = true
				}
				// If we found a filter that don't need post check we can return directly
				// If not we will check if another filter match the given keys, this will prevent from running post check if not needed
				if !needPostCheck {
					return
				}
			}
		}
	}
	return
}

func (f webSocketV2Filters) GetFirstByType(filterType sdk.WebsocketV2FilterType) *sdk.WebsocketV2Filter {
	for _, filter := range f {
		if filter.Type == filterType {
			return &filter
		}
	}
	return nil
}

type websocketV2ClientData struct {
	AuthConsumer sdk.AuthUserConsumer
	mutex        sync.Mutex
	filters      webSocketV2Filters
}

func (c *websocketV2ClientData) updateEventFilters(ctx context.Context, db gorp.SqlExecutor, cache cache.Store, msg []byte) error {
	var fs []sdk.WebsocketV2Filter

	var f sdk.WebsocketV2Filter
	if err := sdk.JSONUnmarshal(msg, &f); err == nil {
		fs = []sdk.WebsocketV2Filter{f}
	} else {
		if err := sdk.JSONUnmarshal(msg, &fs); err != nil {
			return sdk.WrapError(err, "cannot unmarshal websocket input message")
		}
	}

	var isMaintainer = c.AuthConsumer.Maintainer()

	// Check validity of given filters
	for _, f := range fs {
		if err := f.IsValid(); err != nil {
			return err
		}
		switch f.Type {
		case sdk.WebsocketV2FilterTypeProject,
			sdk.WebsocketV2FilterTypeProjectRuns:
			if isMaintainer {
				continue
			}
			hasRole, err := rbac.HasRoleOnProjectAndUserID(ctx, db, sdk.ProjectRoleRead, c.AuthConsumer.AuthConsumerUser.AuthentifiedUser.ID, f.ProjectKey)
			if err != nil {
				return err
			}
			if !hasRole {
				return sdk.WithStack(sdk.ErrForbidden)
			}
		}
	}

	// Update client filters
	c.mutex.Lock()
	c.filters = fs
	c.mutex.Unlock()

	log.Debug(ctx, "websocketV2ClientData.updateEventFilters> event filters updated for client %q with: %v", c.AuthConsumer.ID, fs)

	return nil
}

// For some event we need to check if it should be sent or not depending permissions or filters
func (c *websocketV2ClientData) eventPostCheck(ctx context.Context, db gorp.SqlExecutor, cache cache.Store, event sdk.FullEventV2) (result bool, err error) {
	log.Debug(ctx, "websocketV2ClientData.eventPostCheck> running eventPostCheck for event.Type: %q", string(event.Type))
	defer log.Debug(ctx, "websocketV2ClientData.eventPostCheck> result eventPostCheck for event.Type: %q is %t", string(event.Type), result)

	if event.Type == sdk.EventRunCrafted || event.Type == sdk.EventRunBuilding || event.Type == sdk.EventRunEnded || event.Type == sdk.EventRunRestart {
		filter := c.filters.GetFirstByType(sdk.WebsocketV2FilterTypeProjectRuns)
		if filter == nil {
			return false, nil
		}

		query, err := url.ParseQuery(filter.ProjectRunsParams)
		if err != nil {
			return false, sdk.WrapError(err, "cannot parse project_runs_params filter")
		}
		filters, offset, limit, sort := parseWorkflowRunsSearchV2Query(query)

		runs, err := workflow_v2.SearchRuns(ctx, db, filter.ProjectKey, filters, offset, limit, sort)
		if err != nil {
			return false, sdk.WrapError(err, "unable to search runs")
		}
		for i := range runs {
			if runs[i].ID == event.WorkflowRunID {
				return true, nil
			}
		}
		return false, nil
	}

	return false, nil
}

func (a *API) initWebsocketV2(pubSubKey string) error {
	log.Info(a.Router.Background, "Initializing WS V2 server")
	a.WSV2Server = &websocketV2Server{
		server:     websocket.NewServer(),
		clientData: make(map[string]*websocketV2ClientData),
	}
	tickerMetrics := time.NewTicker(10 * time.Second)
	defer tickerMetrics.Stop()
	a.GoRoutines.Run(a.Router.Background, "api.initWebsocketV2.WSV2Server", func(ctx context.Context) {
		for {
			select {
			case <-tickerMetrics.C:
				telemetry.Record(a.Router.Background, WebSocketV2Clients, int64(len(a.WSV2Server.server.ClientIDs())))
			case <-ctx.Done():
				telemetry.Record(a.Router.Background, WebSocketV2Clients, 0)
				return
			}
		}
	})

	log.Info(a.Router.Background, "Initializing WS V2 events broker")
	pubSub, err := a.Cache.Subscribe(pubSubKey)
	if err != nil {
		return sdk.WrapError(err, "unable to subscribe to %s", pubSubKey)
	}
	a.WSV2Broker = websocket.NewBroker()
	a.WSV2Broker.OnMessage(func(m []byte) {
		telemetry.Record(a.Router.Background, WebSocketV2Events, 1)

		var e sdk.FullEventV2
		if err := sdk.JSONUnmarshal(m, &e); err != nil {
			err = sdk.WrapError(err, "cannot parse event from WS broker")
			ctx := sdk.ContextWithStacktrace(context.TODO(), err)
			log.Warn(ctx, err.Error())
			return
		}

		a.websocketV2OnMessage(e)
	})
	a.WSV2Broker.Init(a.Router.Background, a.GoRoutines, pubSub)
	return nil
}

func (a *API) getWebsocketV2Handler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
		c, err := websocket.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			service.WriteError(ctx, w, r, sdk.NewErrorWithStack(err, sdk.ErrWebsocketUpgrade))
			return nil
		}
		defer c.Close()

		uc := getUserConsumer(ctx)

		wsClient := websocket.NewClient(c)
		wsClientData := &websocketV2ClientData{
			AuthConsumer: *uc,
		}

		wsClient.OnMessage(func(m []byte) {
			if err := wsClientData.updateEventFilters(ctx, a.mustDBWithCtx(ctx), a.Cache, m); err != nil {
				err = sdk.WithStack(err)
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Warn(ctx, err.Error())
				wsClient.Send(sdk.WebsocketV2Event{Status: "KO", Error: sdk.Cause(err).Error()})
			}
		})

		a.WSV2Server.AddClient(wsClient, wsClientData)
		defer a.WSV2Server.RemoveClient(wsClient.UUID())

		return wsClient.Listen(ctx, a.GoRoutines)
	}
}

func (a *API) websocketV2OnMessage(e sdk.FullEventV2) {
	eventKeys := a.websocketV2ComputeEventKeys(e)
	if len(eventKeys) == 0 {
		return
	}

	clientIDs := a.WSV2Server.server.ClientIDs()
	// Randomize the order of client to prevent the old client to always received new events in priority
	rand.Shuffle(len(clientIDs), func(i, j int) { clientIDs[i], clientIDs[j] = clientIDs[j], clientIDs[i] })

	for _, id := range clientIDs {
		// Copy idx for goroutine
		clientID := id

		// Send the event to the client websocket within a goroutine
		a.GoRoutines.Exec(context.Background(), "websocket-"+clientID, func(ctx context.Context) {
			c := a.WSV2Server.GetClientData(clientID)
			if c == nil {
				return
			}

			c.mutex.Lock()
			found, needPostCheck := c.filters.HasOneKey(eventKeys...)
			c.mutex.Unlock()

			log.Debug(ctx, "api.websocketV2OnMessage> check if event need to be sent to client %s for user %s with keys: %q return found: %t and needPostCheck: %t", clientID, c.AuthConsumer.GetUsername(), eventKeys, found, needPostCheck)

			if !found {
				return
			}

			if needPostCheck {
				allowed, err := c.eventPostCheck(ctx, a.mustDBWithCtx(ctx), a.Cache, e)
				if err != nil {
					err = sdk.WrapError(err, "unable to check event permission for client %s with consumer id: %s", clientID, c.AuthConsumer.ID)
					ctx = sdk.ContextWithStacktrace(ctx, err)
					log.Error(ctx, err.Error())
					return
				}
				if !allowed {
					return
				}
			}
			log.Debug(ctx, "api.websocketV2OnMessage> send data to client %s for user %s", clientID, c.AuthConsumer.GetUsername())
			if err := a.WSV2Server.server.SendToClient(clientID, sdk.WebsocketV2Event{
				Status: "OK",
				Event:  e,
			}); err != nil {
				log.Debug(ctx, "api.websocketV2OnMessage> can't send to client %s it will be removed: %+v", clientID, err)
				a.WSServer.RemoveClient(clientID)
			}
		})
	}
}

// This func will compute all the filter keys that match a given event.
func (a *API) websocketV2ComputeEventKeys(event sdk.FullEventV2) []string {
	// Compute required filter(s) key for given event
	var keys []string

	// Event that match project-runs filter
	if event.Type == sdk.EventRunCrafted || event.Type == sdk.EventRunBuilding || event.Type == sdk.EventRunEnded || event.Type == sdk.EventRunRestart {
		keys = append(keys, sdk.WebsocketV2Filter{
			Type:       sdk.WebsocketV2FilterTypeProjectRuns,
			ProjectKey: event.ProjectKey,
		}.Key())
	}

	return keys
}
