package hatchery_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

func init() {
	log.Initialize(context.TODO(), &log.Conf{Level: "debug"})
}

func InitWebsocketTestServer(t *testing.T) *httptest.Server {
	upgrader := websocket.Upgrader{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer c.Close()

		j := sdk.EventRunWorkflowJob{
			ID:     1,
			Status: sdk.StatusWaiting,
		}
		bts, err := json.Marshal(j)
		require.NoError(t, err)
		jevent := sdk.WebsocketEvent{
			Status: "OK",
			Event: sdk.Event{
				EventType: fmt.Sprintf("%T", j),
				Status:    sdk.StatusWaiting,
				Payload:   bts,
			},
		}
		require.NoError(t, c.WriteJSON(jevent))
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				require.NoError(t, err)
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				require.NoError(t, err)
			}
		}
	}))
	return s
}

func InitMock(t *testing.T, url string) {
	privKey, _ := jws.NewRandomRSAKey()
	privKeyPEM, _ := jws.ExportPrivateKey(privKey)
	pubKey, _ := jws.ExportPublicKey(privKey)

	require.NoError(t, authentication.Init("cds-api-test", privKeyPEM))
	id := sdk.UUID()
	consumerID := sdk.UUID()
	hatcheryAuthenticationToken, _ := authentication.NewSessionJWT(&sdk.AuthSession{
		ID:         id,
		ConsumerID: consumerID,
		ExpireAt:   time.Now().Add(time.Hour),
	})

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		if request.Body == nil {
			return
		}
		bodyContent, err := ioutil.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = ioutil.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
		}
	}

	gock.New(url).Post("/auth/consumer/builtin/signin").
		Reply(201).
		JSON(
			sdk.AuthConsumerSigninResponse{
				Token: hatcheryAuthenticationToken,
				User: &sdk.AuthentifiedUser{
					Username: "admin",
				},
			},
		).AddHeader("X-Api-Pub-Signing-Key", base64.StdEncoding.EncodeToString(pubKey))

	gock.New(url).Get("/download/worker/darwin/amd64").Times(1).
		Reply(200).
		Body(bytes.NewBuffer([]byte("nop"))).
		AddHeader("Content-Type", "application/octet-stream")

	gock.New(url).Get("/download/worker/linux/amd64").Times(1).
		Reply(200).
		Body(bytes.NewBuffer([]byte("nop"))).
		AddHeader("Content-Type", "application/octet-stream")

	gock.New(url).Post("/services/register").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(sdk.Service{})

	gock.New(url).Post("/services/heartbeat").
		HeaderPresent("Authorization").
		Reply(204)

	gock.New(url).Get("/worker").Times(6).
		Reply(200).
		JSON([]sdk.Worker{})

	gock.New(url).Get("/queue/workflows/1/infos").Times(1).
		Reply(200).
		JSON(sdk.WorkflowNodeJobRun{
			ID:     1,
			Status: sdk.StatusWaiting,
			Header: sdk.WorkflowRunHeaders{
				"Test": "Test",
			},
		})

	gock.New(url).Post("/queue/workflows/1/spawn/infos").Times(2).Reply(200)

	gock.New(url).Post("/queue/workflows/1/book").
		Reply(204)

	gock.New(url).Get("/queue/workflows").Times(1).
		Reply(200).
		JSON([]sdk.WorkflowRun{})

	gock.Observe(checkRequest)

}

func newMockSSERoundTripper(t *testing.T, ctx context.Context) *MockSSERoundTripper {
	var m = MockSSERoundTripper{
		t: t,
		c: ctx,
	}
	m.reader, m.writer = io.Pipe()
	return &m
}

type MockSSERoundTripper struct {
	t      *testing.T
	c      context.Context
	writer *io.PipeWriter
	reader *io.PipeReader
}

func TestMarshalEvent(t *testing.T) {
	j := sdk.EventRunWorkflowJob{
		ID:     1,
		Status: sdk.StatusWaiting,
	}
	bts, _ := json.Marshal(j)
	msg, err := json.Marshal(sdk.Event{
		EventType: fmt.Sprintf("%T", sdk.EventRunWorkflowJob{}),
		Payload:   bts,
	})

	require.NoError(t, err)
	t.Logf(string(msg))
}

func (m *MockSSERoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	dump, _ := httputil.DumpRequest(req, false)
	m.t.Logf(string(dump))
	resp := new(http.Response)
	resp.Header = http.Header{
		"Content-Type": []string{"text/event-stream"},
	}

	resp.Body = m.reader

	go func() {
		m.writer.Write([]byte(fmt.Sprintf("data: ACK: %s \n\n", sdk.UUID())))

		time.Sleep(5 * time.Second)

		j := sdk.EventRunWorkflowJob{
			ID:     1,
			Status: sdk.StatusWaiting,
		}
		bts, _ := json.Marshal(j)
		msg, err := json.Marshal(sdk.Event{
			EventType: fmt.Sprintf("%T", sdk.EventRunWorkflowJob{}),
			Payload:   bts,
		})

		if err != nil {
			m.t.Fatal(err)
		}

		m.t.Logf("sending event on SSE: %v", string(msg))

		m.writer.Write([]byte("data: "))
		m.writer.Write(msg)
		m.writer.Write([]byte("\n\n"))
		m.t.Logf("event sent on SSE: %v", string(msg))

		<-m.c.Done()
		m.writer.Close()
	}()

	return resp, nil
}
