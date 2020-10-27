package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func Test_websocketWrongFilters(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	require.NoError(t, api.initWebsocket("events_pubsub_test"))

	u, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)
	chanErrorReceived := make(chan error)

	client := cdsclient.New(cdsclient.Config{
		Host:                              tsURL,
		User:                              u.Username,
		InsecureSkipVerifyTLS:             true,
		BuitinConsumerAuthenticationToken: jws,
	})
	go client.WebsocketEventsListen(context.TODO(), sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived, chanErrorReceived)

	// Subscribe to project without project key
	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type:       sdk.WebsocketFilterTypeProject,
		ProjectKey: "",
	}}
	response := <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "missing project key", response.Error)

	// Subscribe to application without application name
	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type:            sdk.WebsocketFilterTypeApplication,
		ProjectKey:      "Key",
		ApplicationName: "",
	}}
	response = <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "missing project key or application name", response.Error)
}

func Test_websocketFilterRetroCompatibility(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	c := &websocketClientData{
		AuthConsumer: *localConsumer,
	}
	buf, err := json.Marshal([]sdk.WebsocketFilter{{
		Type: sdk.WebsocketFilterTypeGlobal,
	}})
	require.NoError(t, err)
	require.NoError(t, c.updateEventFilters(context.TODO(), nil, buf))
	require.Len(t, c.filters, 1)
	require.Equal(t, sdk.WebsocketFilterTypeGlobal, c.filters[0].Type)

	buf, err = json.Marshal(sdk.WebsocketFilter{
		Type: sdk.WebsocketFilterTypeQueue,
	})
	require.NoError(t, err)
	require.NoError(t, c.updateEventFilters(context.TODO(), nil, buf))
	require.Len(t, c.filters, 1)
	require.Equal(t, sdk.WebsocketFilterTypeQueue, c.filters[0].Type)
}

func Test_websocketGetWorkflowEvent(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	require.NoError(t, api.initWebsocket("events_pubsub_test"))

	u, jwt := assets.InsertAdminUser(t, db)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	w := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypeFork,
			},
		},
	}
	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &w))

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)
	chanErrorReceived := make(chan error)

	client := cdsclient.New(cdsclient.Config{
		Host:                  tsURL,
		User:                  u.Username,
		InsecureSkipVerifyTLS: true,
		SessionToken:          jwt,
	})
	go client.WebsocketEventsListen(context.TODO(), sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived, chanErrorReceived)
	var lastResponse *sdk.WebsocketEvent
	go func() {
		for e := range chanMessageReceived {
			lastResponse = &e
		}
	}()

	f := sdk.WebsocketFilter{
		Type:         sdk.WebsocketFilterTypeWorkflow,
		ProjectKey:   proj.Key,
		WorkflowName: w.Name,
	}
	chanMessageToSend <- []sdk.WebsocketFilter{f}
	// Waiting websocket to update filter
	time.Sleep(1 * time.Second)
	require.Nil(t, lastResponse)
	require.Len(t, api.WSServer.server.ClientIDs(), 1)
	for _, id := range api.WSServer.server.ClientIDs() {
		data := api.WSServer.GetClientData(id)
		require.Len(t, data.filters, 1)
		require.Equal(t, sdk.WebsocketFilterTypeWorkflow, data.filters[0].Type)
	}

	api.websocketOnMessage(sdk.Event{ProjectKey: "blabla", WorkflowName: "toto", EventType: "sdk.EventRunWorkflow"})
	api.websocketOnMessage(sdk.Event{ProjectKey: proj.Key, WorkflowName: w.Name, EventType: "sdk.EventRunWorkflow"})
	time.Sleep(1 * time.Second)
	require.NotNil(t, lastResponse)
	require.Equal(t, "OK", lastResponse.Status)
	require.Equal(t, "sdk.EventRunWorkflow", lastResponse.Event.EventType)
	require.Equal(t, proj.Key, lastResponse.Event.ProjectKey)
	require.Equal(t, w.Name, lastResponse.Event.WorkflowName)
	require.Len(t, chanMessageReceived, 0)
}

func Test_websocketDeconnection(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	require.NoError(t, api.initWebsocket("events_pubsub_test"))

	u, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	w := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypeFork,
			},
		},
	}
	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &w))

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

	// Open websocket
	client := cdsclient.New(cdsclient.Config{
		Host:                              tsURL,
		User:                              u.Username,
		InsecureSkipVerifyTLS:             true,
		BuitinConsumerAuthenticationToken: jws,
	})
	resp, err := client.AuthConsumerSignin(sdk.ConsumerBuiltin, sdk.AuthConsumerSigninRequest{"token": jws})
	require.NoError(t, err)
	token := resp.Token

	uHost, err := url.Parse(tsURL)
	require.NoError(t, err)
	urlWebsocket := url.URL{
		Scheme: strings.Replace(uHost.Scheme, "http", "ws", -1),
		Host:   uHost.Host,
		Path:   "/ws",
	}
	headers := make(map[string][]string)
	date := sdk.FormatDateRFC5322(time.Now())
	headers["Date"] = []string{date}
	headers["X-CDS-RemoteTime"] = []string{date}
	auth := "Bearer " + token
	headers["Authorization"] = []string{auth}
	con, _, err := client.HTTPWebsocketClient().Dial(urlWebsocket.String(), headers)
	require.NoError(t, err)
	defer con.Close() // nolint

	// Waiting the websocket add the client
	time.Sleep(1 * time.Second)

	// Send filter
	err = con.WriteJSON([]sdk.WebsocketFilter{{
		Type:         sdk.WebsocketFilterTypeWorkflow,
		ProjectKey:   key,
		WorkflowName: w.Name,
	}})
	require.NoError(t, err)

	// Waiting websocket to update filter
	time.Sleep(1 * time.Second)

	// Send message to client
	go func() {
		for i := 0; i < 100; i++ {
			api.websocketOnMessage(sdk.Event{ProjectKey: proj.Key, WorkflowName: w.Name, EventType: "sdk.EventWorkflow"})
			time.Sleep(200 * time.Millisecond)
		}
	}()
	// Kill client
	con.Close()

	time.Sleep(1 * time.Second)

	require.Len(t, api.WSServer.server.ClientIDs(), 0)
}

func TestWebsocketNoEventLoose(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	pubSubKey := "events_pubsub_test_" + sdk.RandomString(10)
	event.OverridePubSubKey(pubSubKey)
	require.NoError(t, event.Initialize(context.TODO(), api.mustDB(), api.Cache))
	require.NoError(t, api.initWebsocket(pubSubKey))

	_, jwt := assets.InsertAdminUser(t, db)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)

	// First client
	chan1MessageReceived := make(chan sdk.WebsocketEvent)
	chan1MessageToSend := make(chan []sdk.WebsocketFilter)
	chan1ErrorReceived := make(chan error)
	client1 := cdsclient.New(cdsclient.Config{
		Host:                  tsURL,
		InsecureSkipVerifyTLS: true,
		SessionToken:          jwt,
	})
	go client1.WebsocketEventsListen(context.TODO(), sdk.NewGoRoutines(), chan1MessageToSend, chan1MessageReceived, chan1ErrorReceived)
	var client1EventCount int64
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-chan1ErrorReceived:
				require.NoError(t, err)
			case evt := <-chan1MessageReceived:
				if evt.Event.EventType != fmt.Sprintf("%T", sdk.EventFake{}) {
					continue
				}
				var f sdk.EventFake
				require.NoError(t, json.Unmarshal(evt.Event.Payload, &f))
				require.Equal(t, client1EventCount, f.Data)
				client1EventCount++
			}
		}
	}()

	// Second client
	chan2MessageReceived := make(chan sdk.WebsocketEvent)
	chan2MessageToSend := make(chan []sdk.WebsocketFilter)
	chan2ErrorReceived := make(chan error)
	client2 := cdsclient.New(cdsclient.Config{
		Host:                  tsURL,
		InsecureSkipVerifyTLS: true,
		SessionToken:          jwt,
	})
	var client2EventCount int64
	go client2.WebsocketEventsListen(context.TODO(), sdk.NewGoRoutines(), chan2MessageToSend, chan2MessageReceived, chan2ErrorReceived)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-chan2ErrorReceived:
				require.NoError(t, err)
			case evt := <-chan2MessageReceived:
				if evt.Event.EventType != fmt.Sprintf("%T", sdk.EventFake{}) {
					continue
				}
				var f sdk.EventFake
				require.NoError(t, json.Unmarshal(evt.Event.Payload, &f))
				require.Equal(t, client2EventCount, f.Data)
				client2EventCount++
			}
		}
	}()

	filterGlobal := sdk.WebsocketFilter{Type: sdk.WebsocketFilterTypeGlobal}
	chan1MessageToSend <- []sdk.WebsocketFilter{filterGlobal}
	chan2MessageToSend <- []sdk.WebsocketFilter{filterGlobal}

	// Waiting websocket to update filter
	for {
		clientIDs := api.WSServer.server.ClientIDs()
		if len(clientIDs) == 2 {
			data1 := api.WSServer.GetClientData(clientIDs[0])
			data2 := api.WSServer.GetClientData(clientIDs[1])
			if data1 != nil && data2 != nil && len(data1.filters) == 1 && len(data2.filters) == 1 {
				break
			}
		}
		time.Sleep(time.Second)
	}

	// Send events
	countEvent := int64(100)
	for i := int64(0); i < countEvent; i++ {
		event.Publish(context.TODO(), sdk.EventFake{Data: i}, nil)
		time.Sleep(time.Millisecond)
	}

	// Waiting client to receive all events
	for {
		if client1EventCount == countEvent && client2EventCount == countEvent {
			break
		}
		time.Sleep(time.Second)
	}

	// Let 1 second for clients to consume events
	assert.Equal(t, int64(countEvent), client1EventCount, "client 1 loose some events")
	assert.Equal(t, int64(countEvent), client2EventCount, "client 2 loose some events")
}
