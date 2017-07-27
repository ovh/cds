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
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LastUpdateBrokerSubscribe is the information need to to subscribe
type LastUpdateBrokerSubscribe struct {
	UIID  string
	User  *sdk.User
	Queue chan string
}

// LastUpdateBroker keeps connected client of the current route,
type LastUpdateBroker struct {
	clients    map[string]*LastUpdateBrokerSubscribe
	newClients chan *LastUpdateBrokerSubscribe
	messages   chan string
}

var lastUpdateBroker *LastUpdateBroker

func InitLastUpdateBroker(c context.Context, DBFunc func() *gorp.DbMap) {
	lastUpdateBroker = &LastUpdateBroker{
		make(map[string]*LastUpdateBrokerSubscribe),
		make(chan *LastUpdateBrokerSubscribe),
		make(chan string),
	}

	// Start cache Subscription
	go CacheSubscribe(c, lastUpdateBroker.messages)

	// Start processing events
	go lastUpdateBroker.Start(c, DBFunc)
}

// CacheSubscribe subscribe to a channel and push received message in a channel
func CacheSubscribe(c context.Context, cacheMsgChan chan<- string) {
	pubSub := cache.Subscribe("lastUpdates")
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("lastUpdate.CacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case <-tick.C:
			msg, err := cache.GetMessageFromSubscription(c, pubSub)
			if err != nil {
				log.Warning("lastUpdate.CacheSubscribe> Cannot get message %s: %s", msg, err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Info("lastUpdate.CacheSubscribe> New Message from Cache")
			cacheMsgChan <- msg
		}
	}
}

// Start the broker
func (b *LastUpdateBroker) Start(c context.Context, DBFunc func() *gorp.DbMap) {
	for {
		db := DBFunc()
		select {
		case <-c.Done():
			// Close all channels
			for c, v := range b.clients {
				delete(b.clients, c)
				close(v.Queue)
			}
			if c.Err() != nil {
				log.Error("lastUpdate.CacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case s := <-b.newClients:
			// Register new client
			b.clients[s.UIID] = s
		case msg := <-b.messages:
			var lastModif sdk.LastModification
			if err := json.Unmarshal([]byte(msg), &lastModif); err != nil {
				log.Warning("lastUpdate.CacheSubscribe> Cannot unmarshal message: %s", msg)
				continue
			}

			//Receive new message
			for _, i := range b.clients {
				if err := loadUserPermissions(db, i.User); err != nil {
					log.Warning("lastUpdate.CacheSubscribe> Cannot load auser permission: %s", err)
					continue
				}

				hasPermission := false
				if i.User.Admin {
					hasPermission = true
				} else {
				groups:
					for _, g := range i.User.Groups {
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
				}

				if hasPermission {
					i.Queue <- msg
				}
			}
		}
	}

}

func (b *LastUpdateBroker) ServeHTTP(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	log.Info("LastUpdateBroker.ServeHTTP: Connection new User: %s", c.User.Username)
	// Make sure that the writer supports flushing.
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return nil
	}

	uuid, errS := sessionstore.NewSessionKey()
	if errS != nil {
		return sdk.WrapError(errS, "LastUpdateBroker.Serve> Cannot generate UUID")
	}
	messageChan := &LastUpdateBrokerSubscribe{
		UIID:  string(uuid),
		User:  c.User,
		Queue: make(chan string),
	}

	// Add this client to the map of those that should receive updates
	b.newClients <- messageChan

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

leave:
	for {
		select {
		case <-w.(http.CloseNotifier).CloseNotify():
			delete(b.clients, messageChan.UIID)
			close(messageChan.Queue)
			break leave
		case msg, open := <-messageChan.Queue:
			if !open {
				delete(b.clients, messageChan.UIID)
				break leave
			}
			log.Info("LastUpdateBroker.ServeHTTP: Sending msg to User %s: %s", c.User.Username, msg)
			fmt.Fprintf(w, "data: %s\n\n", msg)
			f.Flush()
		}
	}
	return nil
}
