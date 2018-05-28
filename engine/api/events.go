package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	locksKey           = cache.Key("sseevents", "locks")
	eventsKey          = cache.Key("sseevents")
	errLockUnavailable = fmt.Errorf("errLockUnavailable")
)

// eventsBrokerSubscribe is the information needed to subscribe
type eventsBrokerSubscribe struct {
	UUID  string
	User  *sdk.User
	Queue chan string
}

// lastUpdateBroker keeps connected client of the current route,
type eventsBroker struct {
	clients  map[string]eventsBrokerSubscribe
	messages chan sdk.Event
	mutex    *sync.Mutex
	dbFunc   func() *gorp.DbMap
	cache    cache.Store
}

// AddClient add a client to the client map
func (b *eventsBroker) addClient(uuid string, messageChan eventsBrokerSubscribe) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.clients[uuid] = messageChan
}

func (b *eventsBroker) getSubEvents(uuid string) (map[string][]sdk.EventSubscription, bool) {
	var subEvents map[string][]sdk.EventSubscription
	is := b.cache.Get(cache.Key(eventsKey, uuid), &subEvents)
	return subEvents, is
}

func (b *eventsBroker) setSubEvents(uuid string, subEvents map[string][]sdk.EventSubscription) {
	b.cache.SetWithTTL(cache.Key(eventsKey, uuid), subEvents, 600)
}

func (b *eventsBroker) deleteSubEvents(uuid string) {
	b.cache.Delete(cache.Key(eventsKey, uuid))
}

// CleanAll cleans all clients
func (b *eventsBroker) cleanAll() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.clients != nil {
		for c, v := range b.clients {
			close(v.Queue)
			delete(b.clients, c)
			// Clean cache subscription
			if !b.lockCache(v.UUID) {
				log.Warning("CleanAll> Cannot get lock for %s", cache.Key(locksKey, v.UUID))
				continue
			}

			log.Info("CleanALL store subscribe for %s", v.UUID)
			b.deleteSubEvents(v.UUID)
			b.unlockCache(v.UUID)
		}
	}
}

func (b *eventsBroker) lockCache(uuid string) bool {
	return b.cache.Lock(cache.Key(locksKey, uuid), 5*time.Second, 100, 5)
}

func (b *eventsBroker) unlockCache(uuid string) {
	b.cache.Unlock(cache.Key(locksKey, uuid))
}

// CleanClient cleans a client
func (b *eventsBroker) cleanClient(client eventsBrokerSubscribe) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Close channel
	close(client.Queue)
	// Delete client from map
	delete(b.clients, client.UUID)

	// Clean cache subscription
	if !b.lockCache(client.UUID) {
		log.Warning("CleanClient> Cannot get lock for %s", cache.Key(locksKey, client.UUID))
		return
	}

	defer b.unlockCache(client.UUID)
	log.Info("Clean store subscribe for %s", client.UUID)
	b.deleteSubEvents(client.UUID)
}

func (b *eventsBroker) setUser(user *sdk.User) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, c := range b.clients {
		if c.User.Username == user.Username {
			c.User = user
			break
		}
	}
}

func (b *eventsBroker) getUser(username string) *sdk.User {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, c := range b.clients {
		if c.User.Username == username {
			return c.User
		}
	}
	return nil
}

//Init the eventsBroker
func (b *eventsBroker) Init(c context.Context) {
	// Start cache Subscription
	go func() {
		defer func() {
			if re := recover(); re != nil {
				var err error
				switch t := re.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = re.(error)
				case sdk.Error:
					err = re.(sdk.Error)
				default:
					err = sdk.ErrUnknownError
				}
				log.Error("[PANIC] eventsBroker.Init.cacheSubscribe> recover %s", err)
				trace := make([]byte, 4096)
				count := runtime.Stack(trace, true)
				log.Error("[PANIC] eventsBroker.Init.cacheSubscribe> Stacktrace of %d bytes\n%s\n", count, trace)
			}
		}()
		cacheSubscribe(c, b.messages, b.cache)
	}()

	go func() {
		defer func() {
			b.mutex.Unlock()
			if re := recover(); re != nil {
				var err error
				switch t := re.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = re.(error)
				case sdk.Error:
					err = re.(sdk.Error)
				default:
					err = sdk.ErrUnknownError
				}
				log.Error("[PANIC] eventsBroker.Init.Start> recover %s", err)
				trace := make([]byte, 4096)
				count := runtime.Stack(trace, false)
				log.Error("[PANIC] eventsBroker.Init.Start> Stacktrace of %d bytes\n%s\n", count, trace)
				fmt.Println(string(trace))
			}
		}()
		b.Start(c)
	}()
}

func cacheSubscribe(c context.Context, cacheMsgChan chan<- sdk.Event, store cache.Store) {
	pubSub := store.Subscribe("events_pubsub")
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("events.cacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case <-tick.C:
			msg, err := store.GetMessageFromSubscription(c, pubSub)
			if err != nil {
				log.Warning("events.cacheSubscribe> Cannot get message %s: %s", msg, err)
				continue
			}
			var e sdk.Event
			if err := json.Unmarshal([]byte(msg), &e); err != nil {
				log.Warning("events.cacheSubscribe> Cannot unmarshal event %s: %s", msg, err)
				continue
			}

			switch e.EventType {
			case "sdk.EventPipelineBuild", "sdk.EventJob":
				continue
			}
			cacheMsgChan <- e
		}
	}
}

func (b *eventsBroker) UpdateUserPermissions(username string) {
	var user *sdk.User

	user = b.getUser(username)

	if user == nil {
		return
	}
	// load permission without being in the mutex lock
	if err := loadUserPermissions(b.dbFunc(), b.cache, user); err != nil {
		log.Error("eventsBroker.UpdateUserPermissions> Cannot load user permission:%s", err)
	}

	// then, relock map and update user
	b.setUser(user)

}

// Start the broker
func (b *eventsBroker) Start(c context.Context) {
	for {
		select {
		case <-c.Done():
			b.cleanAll()
			if c.Err() != nil {
				log.Error("eventsBroker.CacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case receivedEvent := <-b.messages:
			bEvent, err := json.Marshal(receivedEvent)
			if err != nil {
				continue
			}
			b.manageEvent(receivedEvent, string(bEvent))
		}
	}
}

func (b *eventsBroker) ServeHTTP() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return nil
		}

		uuidSK, errS := sessionstore.NewSessionKey()
		if errS != nil {
			return sdk.WrapError(errS, "eventsBroker.Serve> Cannot generate UUID")
		}
		uuid := string(uuidSK)
		user := getUser(ctx)
		if err := loadUserPermissions(b.dbFunc(), b.cache, user); err != nil {
			return sdk.WrapError(err, "eventsBroker.Serve Cannot load user permission")
		}

		messageChan := eventsBrokerSubscribe{
			UUID:  uuid,
			User:  user,
			Queue: make(chan string, 10), // chan buffered, to avoid goroutine Start() wait on push in queue
		}

		// Add this client to the map of those that should receive updates
		b.addClient(uuid, messageChan)

		// Set the headers related to event streaming.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		fmt.Fprintf(w, "data: ACK: %s \n\n", uuid)
		f.Flush()

		go func() {
			for msg := range messageChan.Queue {
				w.Write([]byte("data: "))
				w.Write([]byte(msg))
				w.Write([]byte("\n\n"))
				f.Flush()
			}
		}()

		tick := time.NewTicker(time.Second)
		defer tick.Stop()
	leave:
		for {
			select {
			case <-ctx.Done():
				b.cleanClient(messageChan)
				break leave
			case <-w.(http.CloseNotifier).CloseNotify():
				log.Info("events.Http: client deconnected")
				b.cleanClient(messageChan)
				break leave
			case <-tick.C:
				f.Flush()
			}
		}

		return nil
	}
}

func (b *eventsBroker) manageEvent(receivedEvent sdk.Event, eventS string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, i := range b.clients {
		if i.Queue != nil {
			b.handleEvent(receivedEvent, eventS, i)
		}

	}
}

func (b *eventsBroker) handleEvent(event sdk.Event, eventS string, subscriber eventsBrokerSubscribe) {
	if strings.HasPrefix(event.EventType, "sdk.EventProject") {
		if subscriber.User.Admin || permission.ProjectPermission(event.ProjectKey, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventWorkflow") {
		if subscriber.User.Admin || permission.WorkflowPermission(event.ProjectKey, event.WorkflowName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventApplication") {
		if subscriber.User.Admin || permission.ApplicationPermission(event.ProjectKey, event.ApplicationName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventPipeline") {
		if subscriber.User.Admin || permission.PipelinePermission(event.ProjectKey, event.PipelineName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventEnvironment") {
		if subscriber.User.Admin || permission.EnvironmentPermission(event.ProjectKey, event.EnvironmentName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventBroadcast") {
		if subscriber.User.Admin || event.ProjectKey == "" || permission.AccessToProject(event.ProjectKey, subscriber.User, permission.PermissionRead) {
			subscriber.Queue <- eventS
		}
		return
	}

	if !b.lockCache(subscriber.UUID) {
		log.Warning("manageEvent> Cannot get lock for %s", cache.Key(locksKey, subscriber.UUID))
		return
	}
	defer b.unlockCache(subscriber.UUID)

	events, ok := b.getSubEvents(subscriber.UUID)
	if !ok {
		log.Debug("Nothing in cache for uuid: %s", subscriber.UUID)
		return
	}

	if strings.HasPrefix(event.EventType, "sdk.EventRunWorkflow") {
		key := event.ProjectKey
		name := event.WorkflowName

		s, ok := events[sdk.EventSubsWorkflowRuns]
		if ok && event.EventType == "sdk.EventRunWorkflow" {
			sent := false
			for _, e := range s {
				if e.ProjectKey == key && e.WorkflowName == name {
					sent = true
					subscriber.Queue <- eventS
					break
				}
			}
			if sent {
				return
			}
		}
		// check if user has subscribed to this specific run
		num := event.WorkflowRunNum
		s, ok = events[sdk.EventSubWorkflowRun]
		if ok && (event.EventType == "sdk.EventRunWorkflowNode" || event.EventType == "sdk.EventRunWorkflowNodeJob") {
			for _, e := range s {
				if e.ProjectKey == key && e.WorkflowName == name && e.WorkflowNum == num {
					subscriber.Queue <- eventS
					break
				}
			}
		}
		return
	}
}

func (api *API) eventSubscribeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var payload sdk.EventSubscription
		if err := UnmarshalBody(r, &payload); err != nil {
			return sdk.WrapError(err, "eventSubscribeHandler> Unable to get body")
		}
		u := getUser(ctx)

		// check permission
		if b := permission.AccessToProject(payload.ProjectKey, u, permission.PermissionRead); !b {
			return sdk.WrapError(sdk.ErrForbidden, "eventSubscribeHandler> cannot access to project %s", payload.ProjectKey)
		}
		if payload.WorkflowName != "" {
			if b := permission.AccessToWorkflow(payload.ProjectKey, payload.WorkflowName, u, permission.PermissionRead); !b {
				return sdk.WrapError(sdk.ErrForbidden, "eventSubscribeHandler> cannot access to workflow")
			}
		}

		if !api.eventsBroker.lockCache(payload.UUID) {
			return sdk.WrapError(fmt.Errorf("unable to get lock"), "eventSubscribeHandler")
		}
		defer api.eventsBroker.unlockCache(payload.UUID)

		events, ok := api.eventsBroker.getSubEvents(payload.UUID)
		if !ok {
			events = make(map[string][]sdk.EventSubscription)
		}

		if payload.WorkflowName != "" {
			if payload.WorkflowRuns {
				// Subscribe to all workflow run
				runs, ok := events[sdk.EventSubsWorkflowRuns]
				if !ok && !payload.Overwrite {
					runs = make([]sdk.EventSubscription, 0)
				}
				if payload.Overwrite {
					runs = make([]sdk.EventSubscription, 1)
					runs[0] = payload
				} else {
					found := false
					for _, es := range runs {
						if es.ProjectKey == payload.ProjectKey && es.WorkflowName == payload.WorkflowName {
							found = true
							break
						}
					}
					if !found {
						runs = append(runs, payload)
					}
				}
				events[sdk.EventSubsWorkflowRuns] = runs
			}

			if payload.WorkflowNum > 0 {
				// Subscribe to the given workflow run
				runs, ok := events[sdk.EventSubWorkflowRun]
				if !ok {
					runs = make([]sdk.EventSubscription, 0)
				}
				if payload.Overwrite {
					runs = make([]sdk.EventSubscription, 1)
					runs[0] = payload
				} else {
					found := false
					for _, es := range runs {
						if es.ProjectKey == payload.ProjectKey && es.WorkflowName == payload.WorkflowName &&
							es.WorkflowNum == payload.WorkflowNum {
							found = true
							break
						}
					}
					if !found {
						runs = append(runs, payload)
					}
				}
				events[sdk.EventSubWorkflowRun] = runs
			}
		}

		api.eventsBroker.setSubEvents(payload.UUID, events)
		return nil
	}
}
