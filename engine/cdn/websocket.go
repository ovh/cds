package cdn

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

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
	tickerMetrics := time.NewTicker(10 * time.Second)
	defer tickerMetrics.Stop()
	s.GoRoutines.Run(s.Router.Background, "cdn.initWebsocket.WSServer", func(ctx context.Context) {
		for {
			select {
			case <-tickerMetrics.C:
				telemetry.Record(s.Router.Background, s.Metrics.WSClients, int64(len(s.WSServer.server.ClientIDs())))
			case <-ctx.Done():
				telemetry.Record(s.Router.Background, s.Metrics.WSClients, 0)
				return
			}
		}
	})

	log.Info(s.Router.Background, "Initializing WS events broker")
	pubSub, err := s.Cache.Subscribe(wbBrokerPubSubKey)
	if err != nil {
		return sdk.WrapError(err, "unable to subscribe to events_pubsub")
	}
	s.WSBroker = websocket.NewBroker()
	s.WSBroker.OnMessage(func(m []byte) {
		telemetry.Record(s.Router.Background, s.Metrics.WSEvents, 1)

		var i sdk.CDNItem
		if err := json.Unmarshal(m, &i); err != nil {
			err = sdk.WrapError(err, "cannot parse event from WS broker")
			log.WarningWithFields(s.Router.Background, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			return
		}

		s.websocketOnMessage(i)
	})
	s.WSBroker.Init(s.Router.Background, s.GoRoutines, pubSub)
	return nil
}

func (s *Service) publishWSEvent(ctx context.Context, i sdk.CDNItem) error {
	b, err := json.Marshal(i)
	if err != nil {
		return sdk.WrapError(err, "cannot marshal event")
	}
	return s.Cache.Publish(ctx, wbBrokerPubSubKey, string(b))
}

func (s *Service) websocketOnMessage(i sdk.CDNItem) {
	// Randomize the order of client to prevent the old client to always received new events in priority
	clientIDs := s.WSServer.server.ClientIDs()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(clientIDs), func(i, j int) { clientIDs[i], clientIDs[j] = clientIDs[j], clientIDs[i] })

	for _, id := range clientIDs {
		c := s.WSServer.GetClientData(id)
		if c.itemID != i.ID {
			continue
		}
		c.chanItemUpdate <- struct{}{}
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
	c := s.clientData[uuid]
	close(c.chanItemUpdate)
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
	chanItemUpdate      chan struct{}
	scoreNextLineToSend int64
}
