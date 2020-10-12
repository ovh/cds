package api

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/stretchr/testify/require"
)

func Test_websocketWrongFilters(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	u, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)

	client := cdsclient.New(cdsclient.Config{
		Host:                              tsURL,
		User:                              u.Username,
		InsecureSkipVerifyTLS:             true,
		BuitinConsumerAuthenticationToken: jws,
	})
	go client.WebsocketEventsListen(context.TODO(), sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived)

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

	c := &websocketClient{
		AuthConsumer: localConsumer,
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

	client := cdsclient.New(cdsclient.Config{
		Host:                  tsURL,
		User:                  u.Username,
		InsecureSkipVerifyTLS: true,
		SessionToken:          jwt,
	})
	go client.WebsocketEventsListen(context.TODO(), sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived)
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
	require.Len(t, api.websocketBroker.clients, 1)
	for _, c := range api.websocketBroker.clients {
		require.Len(t, c.filters, 1)
		require.Equal(t, sdk.WebsocketFilterTypeWorkflow, c.filters[0].Type)
	}

	api.websocketBroker.messages <- sdk.Event{ProjectKey: "blabla", WorkflowName: "toto", EventType: "sdk.EventRunWorkflow"}
	api.websocketBroker.messages <- sdk.Event{ProjectKey: proj.Key, WorkflowName: w.Name, EventType: "sdk.EventRunWorkflow"}
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
			api.websocketBroker.messages <- sdk.Event{ProjectKey: proj.Key, WorkflowName: w.Name, EventType: "sdk.EventWorkflow"}
			time.Sleep(200 * time.Millisecond)
		}
	}()
	// Kill client
	con.Close()

	time.Sleep(1 * time.Second)

	require.Len(t, api.websocketBroker.clients, 0)
}
