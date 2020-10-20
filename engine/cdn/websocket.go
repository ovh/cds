package cdn

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ovh/cds/engine/websocket"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

const wbBrokerPubSubKey = "cdn_ws_broker_pubsub"

func (s *Service) initWebsocket() error {
	log.Info(s.Router.Background, "Initializing WS server")
	s.WSServer = &websocketServer{
		server:     websocket.NewServer(),
		clientData: make(map[string]*websocketClientData),
	}

	log.Info(s.Router.Background, "Initializing WS events broker")
	pubSub, err := s.Cache.Subscribe(wbBrokerPubSubKey)
	if err != nil {
		return sdk.WrapError(err, "unable to subscribe to %s", wbBrokerPubSubKey)
	}
	s.WSBroker = websocket.NewBroker()
	s.WSBroker.OnMessage(func(m []byte) {
		telemetry.Record(s.Router.Background, s.Metrics.WSEvents, 1)

		var e sdk.CDNWSEvent
		if err := json.Unmarshal(m, &e); err != nil {
			err = sdk.WrapError(err, "cannot parse event from WS broker")
			log.WarningWithFields(s.Router.Background, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			return
		}

		s.websocketOnMessage(e)
	})
	s.WSBroker.Init(s.Router.Background, s.GoRoutines, pubSub)

	tickerMetrics := time.NewTicker(10 * time.Second)
	defer tickerMetrics.Stop()
	tickerPublish := time.NewTicker(100 * time.Millisecond)
	defer tickerMetrics.Stop()
	s.GoRoutines.Run(s.Router.Background, "cdn.initWebsocket.SendWSEvents", func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				telemetry.Record(s.Router.Background, s.Metrics.WSClients, 0)
				return
			case <-tickerMetrics.C:
				telemetry.Record(s.Router.Background, s.Metrics.WSClients, int64(len(s.WSServer.server.ClientIDs())))
			case <-tickerPublish.C:
				if err := s.sendWSEvent(ctx); err != nil {
					log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				}
			}
		}
	})
	return nil
}

func (s *Service) publishWSEvent(item sdk.CDNItem) {
	s.WSEventsMutex.Lock()
	defer s.WSEventsMutex.Unlock()
	if s.WSEvents == nil {
		s.WSEvents = make(map[string]sdk.CDNWSEvent)
	}
	s.WSEvents[item.ID] = sdk.CDNWSEvent{
		ItemType: item.Type,
		APIRef:   item.APIRefHash,
	}
}

func (s *Service) sendWSEvent(ctx context.Context) error {
	s.WSEventsMutex.Lock()
	es := make([]sdk.CDNWSEvent, 0, len(s.WSEvents))
	for _, v := range s.WSEvents {
		es = append(es, v)
	}
	s.WSEvents = nil
	s.WSEventsMutex.Unlock()

	for _, e := range es {
		buf, err := json.Marshal(e)
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := s.Cache.Publish(ctx, wbBrokerPubSubKey, string(buf)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) websocketOnMessage(e sdk.CDNWSEvent) {
	// Randomize the order of client to prevent the old client to always received new events in priority
	clientIDs := s.WSServer.server.ClientIDs()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(clientIDs), func(i, j int) { clientIDs[i], clientIDs[j] = clientIDs[j], clientIDs[i] })

	for _, id := range clientIDs {
		c := s.WSServer.GetClientData(id)
		if c == nil || c.itemFilter == nil || !(c.itemFilter.ItemType == e.ItemType && c.itemFilter.APIRef == e.APIRef) {
			continue
		}
		c.TriggerUpdate()
	}
}

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

type websocketClientData struct {
	sessionID           string
	mutexData           sync.Mutex
	itemFilter          *sdk.CDNStreamFilter
	itemUnit            *sdk.CDNItemUnit
	scoreNextLineToSend int64
	mutexTrigger        sync.Mutex
	triggeredUpdate     bool
}

func (d *websocketClientData) TriggerUpdate() {
	d.mutexTrigger.Lock()
	defer d.mutexTrigger.Unlock()
	d.triggeredUpdate = true
}

func (d *websocketClientData) ConsumeTrigger() (triggered bool) {
	d.mutexTrigger.Lock()
	defer d.mutexTrigger.Unlock()
	triggered = d.triggeredUpdate
	d.triggeredUpdate = false
	return
}

func (d *websocketClientData) UpdateFilter(msg []byte) error {
	var filter sdk.CDNStreamFilter
	if err := json.Unmarshal(msg, &filter); err != nil {
		return sdk.WithStack(err)
	}
	if err := filter.Validate(); err != nil {
		return err
	}

	d.mutexData.Lock()
	defer d.mutexData.Unlock()

	d.itemFilter = &filter
	d.scoreNextLineToSend = filter.Offset
	d.itemUnit = nil // reset verified will trigger a new permission check
	return nil
}
