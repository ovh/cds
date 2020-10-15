package cdn

import (
	"context"
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
		s.websocketOnMessage(string(m))
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

func (s *Service) publishWSEvent(itemID string) {
	s.WSEventsMutex.Lock()
	defer s.WSEventsMutex.Unlock()
	if s.WSEvents == nil {
		s.WSEvents = make(map[string]struct{})
	}
	s.WSEvents[itemID] = struct{}{}
}

func (s *Service) sendWSEvent(ctx context.Context) error {
	s.WSEventsMutex.Lock()
	itemIDs := make([]string, 0, len(s.WSEvents))
	for k := range s.WSEvents {
		itemIDs = append(itemIDs, k)
	}
	s.WSEvents = nil
	s.WSEventsMutex.Unlock()

	for _, itemID := range itemIDs {
		if err := s.Cache.Publish(ctx, wbBrokerPubSubKey, itemID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) websocketOnMessage(itemID string) {
	// Randomize the order of client to prevent the old client to always received new events in priority
	clientIDs := s.WSServer.server.ClientIDs()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(clientIDs), func(i, j int) { clientIDs[i], clientIDs[j] = clientIDs[j], clientIDs[i] })

	for _, id := range clientIDs {
		c := s.WSServer.GetClientData(id)
		if c == nil || c.itemID != itemID {
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
	itemID              string
	mutex               sync.Mutex
	triggeredUpdate     bool
	scoreNextLineToSend int64
}

func (d *websocketClientData) TriggerUpdate() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.triggeredUpdate = true
}

func (d *websocketClientData) ConsumeTrigger() (triggered bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	triggered = d.triggeredUpdate
	d.triggeredUpdate = false
	return
}
