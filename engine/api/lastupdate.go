package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LastUpdateBrokerSubscribe is the information need to to subscribe
type LastUpdateBrokerSubscribe struct {
	User  *sdk.User
	Queue chan string
}

// LastUpdateBroker keeps connected client of the current route,
type LastUpdateBroker struct {
	clients     map[chan string]*sdk.User
	newClients  chan LastUpdateBrokerSubscribe
	exitClients chan chan string
	messages    chan string
}

var lastUpdateBroker *LastUpdateBroker

func Initialize(c context.Context, DBFunc func() *gorp.DbMap) {
	lastUpdateBroker = &LastUpdateBroker{
		make(map[chan string]*sdk.User),
		make(chan LastUpdateBrokerSubscribe),
		make(chan (chan string)),
		make(chan string),
	}

	// Start cache Subscription
	go CacheSubscribe(c, lastUpdateBroker.messages)

	// Start processing events
	go lastUpdateBroker.Start(c, DBFunc())
}

// CacheSubscribe subscribe to a channel and push received message in a channel
func CacheSubscribe(c context.Context, cacheMsgChan chan<- string) {
	pubSub := cache.Subscribe("lastupdates")
	tick := time.NewTicker(250 * time.Millisecond).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Warning("lastUpdate.CacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case <-tick:
			msg, err := cache.GetMessageFromSubscription(pubSub, c)
			if err != nil {
				log.Warning("lastUpdate.CacheSubscribe> Cannot get message")
				time.Sleep(5 * time.Second)
				continue
			}
			cacheMsgChan <- msg
		}
	}
}

// Start the broker
func (b *LastUpdateBroker) Start(c context.Context, db gorp.SqlExecutor) {

	for {
		select {
		case <-c.Done():
			// Close all channels
			for c := range b.clients {
				delete(b.clients, c)
				close(c)
			}
			if c.Err() != nil {
				log.Warning("lastUpdate.CacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case s := <-b.newClients:
			// Register new client
			b.clients[s.Queue] = s.User
		case s := <-b.exitClients:
			// Deleting client
			delete(b.clients, s)
			close(s)
		case msg := <-b.messages:
			var lastModif sdk.LastModification
			if err := json.Unmarshal([]byte(msg), &lastModif); err != nil {
				log.Warning("lastUpdate.CacheSubscribe> Cannot unmarshal message: %s", msg)
				continue
			}

			//Receive new message
			for c, u := range b.clients {
				if err := loadUserPermissions(db, u); err != nil {
					log.Warning("lastUpdate.CacheSubscribe> Cannot load auser permission: %s", err)
					continue
				}

				hasPermission := false
			groups:
				for _, g := range u.Groups {
					hasPermission = false
					switch lastModif.Type {
					case sdk.ApplicationLastModificationType:
						for _, ag := range g.ApplicationGroups {
							if ag.Application.Name == lastModif.Name && ag.Application.ProjectKey == lastModif.Key {
								hasPermission = true
								break groups
							}
						}
					case sdk.PipelineLastModificationType:
						for _, pg := range g.PipelineGroups {
							if pg.Pipeline.Name == lastModif.Name && pg.Pipeline.ProjectKey == lastModif.Key {
								hasPermission = true
								break groups
							}
						}
					}
				}
				if hasPermission {
					c <- msg
				}
			}
		}
	}

}

func (b *LastUpdateBroker) ServeHTTP(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {

	// Make sure that the writer supports flushing.
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return nil
	}
	messageChan := LastUpdateBrokerSubscribe{
		User:  c.User,
		Queue: make(chan string),
	}

	// Add this client to the map of those that should receive updates
	b.newClients <- messageChan

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {

		select {
		case <-r.Context().Done():
			// Close
			b.exitClients <- messageChan.Queue
		case msg, open := <-messageChan.Queue:
			if !open {
				break
			}

			// Message must start with data: https://developer.mozilla.org/fr/docs/Server-sent_events/Using_server-sent_events
			fmt.Fprintf(w, "data: %s", msg)
			f.Flush()
		}
	}
	return nil
}
