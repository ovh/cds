package websocket

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewServer() *Server {
	return &Server{clients: make(map[string]Client)}
}

type Server struct {
	mutex   sync.RWMutex
	clients map[string]Client
}

func (s *Server) AddClient(c Client) {
	log.Debug("websocket.Server.AddClient> add client %s", c.UUID())
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.clients[c.UUID()] = c
}

func (s *Server) RemoveClient(uuid string) {
	log.Debug("websocket.Server.RemoveClient> remove client %s", uuid)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	c, has := s.clients[uuid]
	if !has {
		return
	}
	c.Close()
	delete(s.clients, uuid)
}

func (s *Server) ClientIDs() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	ids := make([]string, 0, len(s.clients))
	for k := range s.clients {
		ids = append(ids, k)
	}
	return ids
}

func (s *Server) SendToClient(uuid string, i interface{}) error {
	s.mutex.RLock()
	c, has := s.clients[uuid]
	s.mutex.RUnlock()
	if !has {
		return sdk.WithStack(fmt.Errorf("invalid given client uuid %s", uuid))
	}
	return c.Send(i)
}

func (s *Server) Close() {
	clientsIDs := s.ClientIDs()
	for _, id := range clientsIDs {
		s.RemoveClient(id)
	}
}
