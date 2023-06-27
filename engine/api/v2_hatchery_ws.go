package api

import (
	"context"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/region"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/websocket"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (api *API) getHatcheryWebsocketHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHatchery),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			c, err := websocket.Upgrader.Upgrade(w, r, nil)
			if err != nil {
				service.WriteError(ctx, w, r, sdk.NewErrorWithStack(err, sdk.ErrWebsocketUpgrade))
				return nil
			}
			defer c.Close() // nolint

			hatchConsumer := getHatcheryConsumer(ctx)
			hatch, err := hatchery.LoadHatcheryByID(ctx, api.mustDB(), hatchConsumer.AuthConsumerHatchery.HatcheryID)
			if err != nil {
				return err
			}
			permission, err := rbac.LoadRBACByHatcheryID(ctx, api.mustDB(), hatchConsumer.AuthConsumerHatchery.HatcheryID)
			if err != nil {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "no permission found for this hatchery")
			}

			filter := sdk.WebsocketHatcheryFilter{}
			for _, hatchPerm := range permission.Hatcheries {
				if hatchPerm.HatcheryID == hatchConsumer.AuthConsumerHatchery.HatcheryID {
					reg, err := region.LoadRegionByID(ctx, api.mustDB(), hatchPerm.RegionID)
					if err != nil {
						return err
					}
					filter.ModelType = hatch.ModelType
					filter.Region = reg.Name
					break
				}
			}

			wsClient := websocket.NewClient(c)
			wsClientData := &websocketHatcheryData{
				AuthConsumer: *getHatcheryConsumer(ctx),
				filter:       filter,
			}
			api.WSHatcheryServer.AddClient(wsClient, wsClientData)
			defer api.WSHatcheryServer.RemoveClient(wsClient.UUID())
			return wsClient.Listen(ctx, api.GoRoutines)
		}

}

type websocketHatcheryServer struct {
	server     *websocket.Server
	mutex      sync.RWMutex
	clientData map[string]*websocketHatcheryData
}

type websocketHatcheryData struct {
	AuthConsumer sdk.AuthHatcheryConsumer
	mutex        sync.Mutex
	filter       sdk.WebsocketHatcheryFilter
}

func (api *API) initHatcheryWebsocket(pubSubKey string) error {
	log.Info(api.Router.Background, "Initializing hatchery WS server")
	api.WSHatcheryServer = &websocketHatcheryServer{
		server:     websocket.NewServer(),
		clientData: make(map[string]*websocketHatcheryData),
	}
	tickerMetrics := time.NewTicker(10 * time.Second)
	defer tickerMetrics.Stop()
	api.GoRoutines.Run(api.Router.Background, "api.InitRouter.WSHatcheryServer", func(ctx context.Context) {
		for {
			select {
			case <-tickerMetrics.C:
				telemetry.Record(api.Router.Background, WebSocketHatcheryClients, int64(len(api.WSHatcheryServer.server.ClientIDs())))
			case <-ctx.Done():
				telemetry.Record(api.Router.Background, WebSocketHatcheryClients, 0)
				return
			}
		}
	})

	log.Info(api.Router.Background, "Initializing WS events broker")
	pubSub, err := api.Cache.Subscribe(pubSubKey)
	if err != nil {
		return sdk.WrapError(err, "unable to subscribe to %s", pubSubKey)
	}
	api.WSHatcheryBroker = websocket.NewBroker()
	api.WSHatcheryBroker.OnMessage(func(m []byte) {
		telemetry.Record(api.Router.Background, WebSocketEvents, 1)
		var e sdk.WebsocketJobQueueEvent
		if err := sdk.JSONUnmarshal(m, &e); err != nil {
			err = sdk.WrapError(err, "cannot parse event from WS broker")
			ctx := sdk.ContextWithStacktrace(context.TODO(), err)
			log.Warn(ctx, err.Error())
			return
		}
		api.websocketHatcheryOnMessage(e)
	})
	api.WSHatcheryBroker.Init(api.Router.Background, api.GoRoutines, pubSub)
	return nil
}

func (a *API) websocketHatcheryOnMessage(e sdk.WebsocketJobQueueEvent) {
	currentRegion := e.Region
	currentModel := e.ModelType

	// Randomize the order of client to prevent the old client to always received new events in priority
	clientIDs := a.WSHatcheryServer.server.ClientIDs()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(clientIDs), func(i, j int) { clientIDs[i], clientIDs[j] = clientIDs[j], clientIDs[i] })

	for _, id := range clientIDs {
		// Copy idx for goroutine
		clientID := id

		// Send the event to the client websocket within a goroutine
		a.GoRoutines.Exec(context.Background(), "websocket-"+clientID, func(ctx context.Context) {
			c := a.WSHatcheryServer.GetClientData(clientID)
			if c == nil {
				return
			}

			c.mutex.Lock()
			canHandleJob := c.filter.Region == currentRegion && c.filter.ModelType == currentModel
			c.mutex.Unlock()
			if !canHandleJob {
				return
			}
			log.Debug(ctx, "api.websocketHatcheryOnMessage> send data to client %s for hatchery %s", clientID, c.AuthConsumer.AuthConsumerHatchery.HatcheryID)
			if err := a.WSHatcheryServer.server.SendToClient(clientID, sdk.WebsocketHatcheryEvent{
				Status: "OK",
				Event:  e,
			}); err != nil {
				log.Debug(ctx, "websocketOnMessage> can't send to client %s it will be removed: %+v", clientID, err)
				a.WSHatcheryServer.RemoveClient(clientID)
			}
		})
	}
}

func (s *websocketHatcheryServer) AddClient(c websocket.Client, data *websocketHatcheryData) {
	s.server.AddClient(c)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.clientData[c.UUID()] = data
}

func (s *websocketHatcheryServer) RemoveClient(uuid string) {
	s.server.RemoveClient(uuid)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.clientData, uuid)
}

func (s *websocketHatcheryServer) GetClientData(uuid string) *websocketHatcheryData {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	data, ok := s.clientData[uuid]
	if !ok {
		return nil
	}
	return data
}
