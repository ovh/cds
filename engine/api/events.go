package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// eventsBrokerSubscribe is the information needed to subscribe
type eventsBrokerSubscribe struct {
	UIID   string
	User   *sdk.User
	Events map[string][]sdk.EventSubscription
	Queue  chan string
}

// lastUpdateBroker keeps connected client of the current route,
type eventsBroker struct {
	clients  map[string]eventsBrokerSubscribe
	messages chan sdk.Event
	mutex    *sync.Mutex
	dbFunc   func() *gorp.DbMap
	cache    cache.Store
}

//Init the eventsBroker
func (b *eventsBroker) Init(c context.Context) {
	// Start cache Subscription
	event.Subscribe(b.messages)

	// Start processing events
	go b.Start(c)
}

func (b *eventsBroker) UpdateUserPermissions(username string) {
	var user *sdk.User

	// get the user
	b.mutex.Lock()
	for _, c := range b.clients {
		if c.User.Username == username {
			user = c.User
			break
		}
	}
	b.mutex.Unlock()

	if user == nil {
		return
	}
	// load permission without being in the mutex lock
	if err := loadUserPermissions(b.dbFunc(), b.cache, user); err != nil {
		log.Error("eventsBroker.UpdateUserPermissions> Cannot load user permission:%s", err)
	}

	// then, relock map and update user
	b.mutex.Lock()
	for _, c := range b.clients {
		if c.User.Username == username {
			c.User = user
			break
		}
	}
	b.mutex.Unlock()
}

// Start the broker
func (b *eventsBroker) Start(c context.Context) {
	for {
		select {
		case <-c.Done():
			// Close all channels
			b.mutex.Lock()
			for c := range b.clients {
				delete(b.clients, c)
			}
			b.mutex.Unlock()
			if c.Err() != nil {
				log.Error("eventsBroker.CacheSubscribe> Exiting: %v", c.Err())
				return
			}
		case receivedEvent := <-b.messages:
			bEvent, err := json.Marshal(receivedEvent)
			if err != nil {
				continue
			}

			b.mutex.Lock()
			for _, i := range b.clients {
				if i.Queue != nil {
					manageEvent(receivedEvent, string(bEvent), i)
				}

			}
			b.mutex.Unlock()
		}
	}
}

func (b *eventsBroker) ServeHTTP() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		uuid := FormString(r, "uuid")

		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return nil
		}

		if uuid == "" {
			uuidSK, errS := sessionstore.NewSessionKey()
			if errS != nil {
				return sdk.WrapError(errS, "eventsBroker.Serve> Cannot generate UUID")
			}
			uuid = string(uuidSK)
		}

		user := getUser(ctx)
		if err := loadUserPermissions(b.dbFunc(), b.cache, user); err != nil {
			return sdk.WrapError(err, "eventsBroker.Serve Cannot load user permission")
		}

		messageChan := eventsBrokerSubscribe{
			UIID:   string(uuid),
			User:   user,
			Events: make(map[string][]sdk.EventSubscription),
			Queue:  make(chan string, 10), // chan buffered, to avoid goroutine Start() wait on push in queue
		}

		// Add this client to the map of those that should receive updates
		b.mutex.Lock()
		b.clients[uuid] = messageChan
		b.mutex.Unlock()

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
				b.mutex.Lock()
				close(messageChan.Queue)
				delete(b.clients, messageChan.UIID)
				b.mutex.Unlock()
				break leave
			case <-w.(http.CloseNotifier).CloseNotify():
				b.mutex.Lock()
				close(messageChan.Queue)
				delete(b.clients, messageChan.UIID)
				b.mutex.Unlock()
				break leave
			case <-tick.C:
				f.Flush()
			}
		}

		return nil
	}
}

func manageEvent(event sdk.Event, eventS string, subscriber eventsBrokerSubscribe) {
	if strings.HasPrefix(event.EventType, "sdk.EventRunWorkflow") {
		key := event.ProjectKey
		name := event.WorkflowName
		// check if user has subscribed to runs list
		s, ok := subscriber.Events[sdk.EventSubsWorkflowRuns]
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
		s, ok = subscriber.Events[sdk.EventSubWorkflowRun]
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

	// Update on Project, Workflow, Application, Pipeline, Environment
	if subscriber.User.Admin {
		subscriber.Queue <- eventS
		return
	}

	if strings.HasPrefix(event.EventType, "sdk.EventProject") {
		if permission.ProjectPermission(event.ProjectKey, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventWorkflow") {
		if permission.WorkflowPermission(event.ProjectKey, event.WorkflowName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventApplication") {
		if permission.ApplicationPermission(event.ProjectKey, event.ApplicationName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventPipeline") {
		if permission.PipelinePermission(event.ProjectKey, event.PipelineName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
		}
		return
	}
	if strings.HasPrefix(event.EventType, "sdk.EventEnvironment") {
		if permission.EnvironmentPermission(event.ProjectKey, event.EnvironmentName, subscriber.User) >= permission.PermissionRead {
			subscriber.Queue <- eventS
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

		api.eventsBroker.mutex.Lock()
		defer api.eventsBroker.mutex.Unlock()
		data := api.eventsBroker.clients[payload.UUID]
		if data.Events == nil {
			data.Events = make(map[string][]sdk.EventSubscription)
		}

		if payload.WorkflowName != "" {
			if payload.WorkflowRuns {
				// Subscribe to all workflow run
				runs, ok := data.Events[sdk.EventSubsWorkflowRuns]
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
				data.Events[sdk.EventSubsWorkflowRuns] = runs
			}

			if payload.WorkflowNum > 0 {
				// Subscribe to the given workflow run
				runs, ok := data.Events[sdk.EventSubWorkflowRun]
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
				data.Events[sdk.EventSubWorkflowRun] = runs
			}
		}

		api.eventsBroker.clients[payload.UUID] = data
		return nil
	}
}

func (api *API) eventUnsubscribeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var payload sdk.EventSubscription
		if err := UnmarshalBody(r, &payload); err != nil {
			return sdk.WrapError(err, "eventUnsubscribeHandler> Unable to get body")
		}

		api.eventsBroker.mutex.Lock()
		defer api.eventsBroker.mutex.Unlock()
		data := api.eventsBroker.clients[payload.UUID]

		if payload.WorkflowName != "" {
			if payload.WorkflowRuns {
				// Subscribe to all workflow run
				if runs, ok := data.Events[sdk.EventSubsWorkflowRuns]; ok {
					found := false
					index := 0
					for i, es := range runs {
						if es.ProjectKey == payload.ProjectKey && es.WorkflowName == payload.WorkflowName {
							found = true
							index = i
							break
						}
					}
					if found {
						runs = append(runs[:index], runs[index+1:]...)
						data.Events[sdk.EventSubsWorkflowRuns] = runs
					}
				}

			}

			if payload.WorkflowNum > 0 {
				// Subscribe to the given workflow run
				if runs, ok := data.Events[sdk.EventSubWorkflowRun]; ok {
					found := false
					index := 0
					for i, es := range runs {
						if es.ProjectKey == payload.ProjectKey && es.WorkflowName == payload.WorkflowName &&
							es.WorkflowNum == payload.WorkflowNum {
							found = true
							index = i
							break
						}
					}
					if found {
						runs = append(runs[:index], runs[index+1:]...)
						data.Events[sdk.EventSubWorkflowRun] = runs
					}
				}
			}
		}
		api.eventsBroker.clients[payload.UUID] = data
		return nil
	}
}
