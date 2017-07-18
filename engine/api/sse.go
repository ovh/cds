package main

import (
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/cache"
	"context"
)

type BrokerSubscribe struct {
	ID int
	Queue chan string
}
// Broker keep connected client of the current route,
type Broker struct {
	clients     map[int]chan string
	newClients  chan BrokerSubscribe
	exitClients chan int
	messages    chan string
}

func LastUpdateEventBus(ctx context.Context) {
	pubSub := cache.Subscribe("lastUpdate")
	if pubSub == nil {
		return
	}

	for {
		select {
		case ctx.Done():
			// TODO Stop all

		}
	}
}

// Start the broker
func (b *Broker) Start() {
	go func() {
		for {
			select {
			case s := <-b.newClients:
				b.clients[s.ID] = s.Queue
			case s := <-b.exitClients:
				delete(b.clients, s)
				close(s)
			case msg := <-b.messages:
				for s := range b.clients {
					s <- msg
				}
			}
		}
	}()
}

func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error{

	// Make sure that the writer supports flushing.
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return nil
	}
	messageChan := make(chan string)

	// Add this client to the map of those that should receive updates
	b.newClients <- messageChan

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {

		select {
		case <- r.Context().Done():
			// Close
			b.exitClients <- messageChan
		 case msg, open := <-messageChan:
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
