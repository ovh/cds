package api

import (
	"context"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/stretchr/testify/require"
	"net/url"
	"strings"
	"testing"
	"time"
)

func Test_websocketWrongFilters(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	u, _ := assets.InsertAdminUser(t, api.mustDB())
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan sdk.WebsocketFilter)

	client := cdsclient.New(cdsclient.Config{
		Host:                              tsURL,
		User:                              u.Username,
		InsecureSkipVerifyTLS:             true,
		BuitinConsumerAuthenticationToken: jws,
	})
	go client.WebsocketEventsListen(context.TODO(), chanMessageToSend, chanMessageReceived)

	// Subscribe to project without project key
	chanMessageToSend <- sdk.WebsocketFilter{
		Type:       sdk.WebsocketFilterTypeProject,
		ProjectKey: "",
	}
	response := <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "wrong request", response.Error)

	// Subscribe to application without application name
	chanMessageToSend <- sdk.WebsocketFilter{
		Type:            sdk.WebsocketFilterTypeApplication,
		ProjectKey:      "Key",
		ApplicationName: "",
	}
	response = <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "wrong request", response.Error)

	// Subscribe to application without project key
	chanMessageToSend <- sdk.WebsocketFilter{
		Type:            sdk.WebsocketFilterTypeApplication,
		ProjectKey:      "",
		ApplicationName: "App1",
	}
	response = <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "wrong request", response.Error)

	// Subscribe to pipeline without pipeline name
	chanMessageToSend <- sdk.WebsocketFilter{
		Type:         sdk.WebsocketFilterTypeApplication,
		ProjectKey:   "Key",
		PipelineName: "",
	}
	response = <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "wrong request", response.Error)

	// Subscribe to pipeline without project key
	chanMessageToSend <- sdk.WebsocketFilter{
		Type:         sdk.WebsocketFilterTypePipeline,
		ProjectKey:   "",
		PipelineName: "PipName",
	}
	response = <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "wrong request", response.Error)

	// Subscribe to environment without environment name
	chanMessageToSend <- sdk.WebsocketFilter{
		Type:            sdk.WebsocketFilterTypeEnvironment,
		ProjectKey:      "Key",
		EnvironmentName: "",
	}
	response = <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "wrong request", response.Error)

	// Subscribe to environment without project key
	chanMessageToSend <- sdk.WebsocketFilter{
		Type:            sdk.WebsocketFilterTypeEnvironment,
		ProjectKey:      "",
		EnvironmentName: "EnvNmae",
	}
	response = <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "wrong request", response.Error)

	// Subscribe to workflow without workflow name
	chanMessageToSend <- sdk.WebsocketFilter{
		Type:         sdk.WebsocketFilterTypeWorkflow,
		ProjectKey:   "Key",
		WorkflowName: "",
	}
	response = <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "wrong request", response.Error)

	// Subscribe to workflow without project key
	chanMessageToSend <- sdk.WebsocketFilter{
		Type:         sdk.WebsocketFilterTypeWorkflow,
		ProjectKey:   "",
		WorkflowName: "WorkflowName",
	}
	response = <-chanMessageReceived
	require.Equal(t, "KO", response.Status)
	require.Equal(t, "wrong request", response.Error)
}

func Test_websocketGetWorkflowEvent(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	u, _ := assets.InsertAdminUser(t, api.mustDB())
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, key, key)

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
	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj, &w))

	_, jws, err := builtin.NewConsumer(context.TODO(), api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan sdk.WebsocketFilter)

	client := cdsclient.New(cdsclient.Config{
		Host:                              tsURL,
		User:                              u.Username,
		InsecureSkipVerifyTLS:             true,
		BuitinConsumerAuthenticationToken: jws,
	})
	go client.WebsocketEventsListen(context.TODO(), chanMessageToSend, chanMessageReceived)

	chanMessageToSend <- sdk.WebsocketFilter{
		Type:              sdk.WebsocketFilterTypeWorkflow,
		ProjectKey:        key,
		WorkflowName:      w.Name,
		WorkflowRunNumber: 1,
	}
	// Waiting websocket to update filter
	time.Sleep(1 * time.Second)

	api.websocketBroker.messages <- sdk.Event{ProjectKey: "blabla", WorkflowName: "toto", EventType: "sdk.EventRunWorkflow", WorkflowRunNum: 1}
	api.websocketBroker.messages <- sdk.Event{ProjectKey: proj.Key, WorkflowName: w.Name, EventType: "sdk.EventRunWorkflow", WorkflowRunNum: 1}
	response := <-chanMessageReceived
	require.Equal(t, "OK", response.Status)
	require.Equal(t, response.Event.EventType, "sdk.EventRunWorkflow")
	require.Equal(t, response.Event.ProjectKey, proj.Key)
	require.Equal(t, response.Event.WorkflowName, w.Name)
	require.Equal(t, 0, len(chanMessageReceived))
}

func Test_websocketDeconnection(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	u, _ := assets.InsertAdminUser(t, api.mustDB())
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, key, key)

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
	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj, &w))

	_, jws, err := builtin.NewConsumer(context.TODO(), api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(),
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
	err = con.WriteJSON(sdk.WebsocketFilter{
		Type:         sdk.WebsocketFilterTypeWorkflow,
		ProjectKey:   key,
		WorkflowName: w.Name,
	})
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

	require.Equal(t, len(api.websocketBroker.clients), 0)
}
